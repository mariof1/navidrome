package nativeapi

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type homeRecommendationsResponse struct {
	Sections []homeRecommendationsSection `json:"sections"`
}

type homeRecommendationsSection struct {
	ID       string       `json:"id"`
	Resource string       `json:"resource"`
	To       string       `json:"to"`
	Items    model.Albums `json:"items"`
}

type homeSectionCandidate struct {
	ID       string
	Resource string
	To       string
	Kind     string
	Build    func() (model.Albums, error)
}

func overfetchMax(limit int) int {
	if limit <= 0 {
		return 0
	}
	max := limit * 5
	if max < limit {
		max = limit
	}
	if max > 200 {
		max = 200
	}
	return max
}

func albumScoreForMix(a model.Album, now time.Time) float64 {
	// Higher is better. This is intentionally simple (and explainable):
	// - Big boost for never-played albums
	// - Prefer albums not played recently
	// - Small familiarity boost for higher play counts
	neverPlayedBoost := 0.0
	if a.PlayCount == 0 {
		neverPlayedBoost = 1000
	}

	daysSincePlay := 0.0
	if a.PlayDate != nil {
		daysSincePlay = now.Sub(*a.PlayDate).Hours() / 24
		if daysSincePlay < 0 {
			daysSincePlay = 0
		}
		// Cap so ancient plays don't dominate too much.
		if daysSincePlay > 365 {
			daysSincePlay = 365
		}
	} else {
		// Treat unknown/never as "very old" without overpowering the never-played boost.
		daysSincePlay = 90
	}

	familiarity := math.Log1p(float64(a.PlayCount)) * 8
	return neverPlayedBoost + (daysSincePlay * 4) + familiarity
}

func selectDiverseAlbums(candidates model.Albums, limit int, now time.Time, excluded map[string]struct{}, maxPerArtist int, requiredArtistIDs []string) model.Albums {
	if limit <= 0 || len(candidates) == 0 {
		return nil
	}
	if maxPerArtist <= 0 {
		maxPerArtist = limit
	}

	// De-dupe and apply exclusions.
	seen := make(map[string]struct{}, len(candidates))
	filtered := make(model.Albums, 0, len(candidates))
	for _, a := range candidates {
		if a.ID == "" {
			continue
		}
		if _, ok := excluded[a.ID]; ok {
			continue
		}
		if _, ok := seen[a.ID]; ok {
			continue
		}
		seen[a.ID] = struct{}{}
		filtered = append(filtered, a)
	}
	if len(filtered) == 0 {
		return nil
	}

	scores := make(map[string]float64, len(filtered))
	for _, a := range filtered {
		scores[a.ID] = albumScoreForMix(a, now)
	}

	// Sort best-first. Candidate order is already pseudo-random due to DB seed;
	// this turns that random sample into a more relevant mix.
	sort.SliceStable(filtered, func(i, j int) bool {
		return scores[filtered[i].ID] > scores[filtered[j].ID]
	})

	artistKey := func(a model.Album) string {
		if a.AlbumArtistID != "" {
			return a.AlbumArtistID
		}
		// Fallback to keep compilations from being overly constrained.
		return a.ID
	}

	add := func(out *model.Albums, artistCounts map[string]int, picked map[string]struct{}, a model.Album, capPerArtist int) bool {
		if len(*out) >= limit {
			return false
		}
		if _, ok := picked[a.ID]; ok {
			return false
		}
		k := artistKey(a)
		if artistCounts[k] >= capPerArtist {
			return false
		}
		picked[a.ID] = struct{}{}
		artistCounts[k]++
		*out = append(*out, a)
		return true
	}

	out := make(model.Albums, 0, min(limit, len(filtered)))
	picked := make(map[string]struct{}, limit)
	artistCounts := make(map[string]int, limit)

	// First pass: ensure coverage of required seed artists (if possible).
	if len(requiredArtistIDs) > 0 {
		need := make(map[string]struct{}, len(requiredArtistIDs))
		for _, id := range requiredArtistIDs {
			if id != "" {
				need[id] = struct{}{}
			}
		}
		for _, req := range requiredArtistIDs {
			if _, ok := need[req]; !ok {
				continue
			}
			for _, a := range filtered {
				if a.AlbumArtistID != req {
					continue
				}
				if add(&out, artistCounts, picked, a, maxPerArtist) {
					delete(need, req)
					break
				}
			}
		}
	}

	// Second pass: fill respecting the per-artist cap.
	for _, a := range filtered {
		if len(out) >= limit {
			break
		}
		add(&out, artistCounts, picked, a, maxPerArtist)
	}

	// Final pass: if we still couldn't fill, relax the cap.
	if len(out) < limit {
		for _, a := range filtered {
			if len(out) >= limit {
				break
			}
			add(&out, artistCounts, picked, a, limit)
		}
	}

	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (api *Router) addRecommendationsRoute(r chi.Router) {
	r.Get("/recommendations/home", api.getHomeRecommendations())
}

func (api *Router) getHomeRecommendations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 12
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
				limit = n
			}
		}
		seed := r.URL.Query().Get("seed")

		now := time.Now().UTC()
		onRepeatCutoff := now.AddDate(0, 0, -14)
		rediscoverCutoff := now.AddDate(0, 0, -30)
		inspiredByCutoff := now.AddDate(0, 0, -7)
		continueListeningCutoff := now.AddDate(0, 0, -3)

		albumRepo := api.ds.Album(r.Context())
		excludedMixAlbumIDs := map[string]struct{}{}

		// Derive a small set of “seed” artists from the user's recent listening patterns.
		// Prefer user_events-derived scoring if available; fallback to play_count/play_date.
		seedArtistIDs, err := api.ds.UserEvent(r.Context()).TopAlbumArtistIDs(6, now)
		if err != nil {
			log.Trace(r.Context(), "Error retrieving top album artists", err)
			seedArtistIDs = nil
		}
		if len(seedArtistIDs) == 0 {
			seenArtists := map[string]struct{}{}
			appendArtists := func(albums model.Albums) {
				for _, a := range albums {
					if a.AlbumArtistID == "" {
						continue
					}
					if _, ok := seenArtists[a.AlbumArtistID]; ok {
						continue
					}
					seenArtists[a.AlbumArtistID] = struct{}{}
					seedArtistIDs = append(seedArtistIDs, a.AlbumArtistID)
				}
			}

			onRepeatSeed, err := albumRepo.GetAll(model.QueryOptions{
				Sort:  "play_count",
				Order: "DESC",
				Max:   limit,
				Filters: squirrel.And{
					squirrel.Gt{"play_count": 0},
					squirrel.GtOrEq{"play_date": onRepeatCutoff},
				},
			})
			if err != nil {
				log.Error(r.Context(), "Error building home recommendations", "section", "onRepeat", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			recentlyPlayedSeed, err := albumRepo.GetAll(model.QueryOptions{
				Sort:    "play_date",
				Order:   "DESC",
				Max:     limit,
				Filters: squirrel.Gt{"play_count": 0},
			})
			if err != nil {
				log.Error(r.Context(), "Error building home recommendations", "section", "recentlyPlayed", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			mostPlayedSeed, err := albumRepo.GetAll(model.QueryOptions{
				Sort:    "play_count",
				Order:   "DESC",
				Max:     limit,
				Filters: squirrel.Gt{"play_count": 0},
			})
			if err != nil {
				log.Error(r.Context(), "Error building home recommendations", "section", "mostPlayed", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			appendArtists(onRepeatSeed)
			appendArtists(recentlyPlayedSeed)
			appendArtists(mostPlayedSeed)
		}

		mix1IDs := seedArtistIDs
		if len(mix1IDs) > 3 {
			mix1IDs = mix1IDs[:3]
		}
		mix2IDs := []string{}
		if len(seedArtistIDs) > 3 {
			mix2IDs = seedArtistIDs[3:]
			if len(mix2IDs) > 3 {
				mix2IDs = mix2IDs[:3]
			}
		}

		dailyMix1Filters := squirrel.Sqlizer(nil)
		if len(mix1IDs) > 0 {
			dailyMix1Filters = squirrel.Eq{"album_artist_id": mix1IDs}
		}
		dailyMix1Seed := seed + "-dm1"

		dailyMix2Filters := squirrel.Sqlizer(nil)
		if len(mix2IDs) > 0 {
			dailyMix2Filters = squirrel.Eq{"album_artist_id": mix2IDs}
		}
		dailyMix2Seed := seed + "-dm2"

		dailyMix3Seed := seed + "-dm3"

		// Candidate bucket builders. We build only the chosen buckets (curated) to avoid
		// flooding the UI and to keep the endpoint efficient.
		candidates := []homeSectionCandidate{
			{
				ID:       "dailyMix1",
				Resource: "album",
				To:       "",
				Kind:     "mix",
				Build: func() (model.Albums, error) {
					poolMax := overfetchMax(limit)
					maxPerArtist := 2
					requiredArtists := []string(nil)
					filters := dailyMix1Filters
					if len(mix1IDs) > 0 {
						maxPerArtist = int(math.Ceil(float64(limit) / float64(len(mix1IDs))))
						if maxPerArtist < 2 {
							maxPerArtist = 2
						}
						requiredArtists = mix1IDs
					} else {
						// Cold-start fallback: build a general-purpose mix.
						filters = squirrel.Or{
							squirrel.Expr("play_date IS NULL"),
							squirrel.Lt{"play_date": rediscoverCutoff},
							squirrel.Eq{"play_count": 0},
						}
					}
					pool, err := albumRepo.GetAll(model.QueryOptions{
						Sort:    "random",
						Order:   "ASC",
						Max:     poolMax,
						Seed:    dailyMix1Seed,
						Filters: filters,
					})
					if err != nil {
						return nil, err
					}
					return selectDiverseAlbums(pool, limit, now, excludedMixAlbumIDs, maxPerArtist, requiredArtists), nil
				},
			},
			{
				ID:       "dailyMix2",
				Resource: "album",
				To:       "",
				Kind:     "mix",
				Build: func() (model.Albums, error) {
					if len(mix2IDs) == 0 {
						return nil, nil
					}
					poolMax := overfetchMax(limit)
					maxPerArtist := int(math.Ceil(float64(limit) / float64(len(mix2IDs))))
					if maxPerArtist < 2 {
						maxPerArtist = 2
					}
					pool, err := albumRepo.GetAll(model.QueryOptions{
						Sort:    "random",
						Order:   "ASC",
						Max:     poolMax,
						Seed:    dailyMix2Seed,
						Filters: dailyMix2Filters,
					})
					if err != nil {
						return nil, err
					}
					return selectDiverseAlbums(pool, limit, now, excludedMixAlbumIDs, maxPerArtist, mix2IDs), nil
				},
			},
			{
				ID:       "dailyMix3",
				Resource: "album",
				To:       "",
				Kind:     "mix",
				Build: func() (model.Albums, error) {
					poolMax := overfetchMax(limit)
					pool, err := albumRepo.GetAll(model.QueryOptions{
						Sort:  "random",
						Order: "ASC",
						Max:   poolMax,
						Seed:  dailyMix3Seed,
						Filters: squirrel.Or{
							squirrel.Expr("play_date IS NULL"),
							squirrel.Lt{"play_date": rediscoverCutoff},
							squirrel.Eq{"play_count": 0},
						},
					})
					if err != nil {
						return nil, err
					}
					return selectDiverseAlbums(pool, limit, now, excludedMixAlbumIDs, 2, nil), nil
				},
			},
			{
				ID:       "inspiredBy",
				Resource: "album",
				To:       "",
				Kind:     "mix",
				Build: func() (model.Albums, error) {
					if len(seedArtistIDs) == 0 {
						return nil, nil
					}
					poolMax := overfetchMax(limit)
					artistIDs := seedArtistIDs
					if len(artistIDs) > 2 {
						artistIDs = artistIDs[:2]
					}
					pool, err := albumRepo.GetAll(model.QueryOptions{
						Sort:  "random",
						Order: "ASC",
						Max:   poolMax,
						Seed:  seed + "-inspired",
						Filters: squirrel.And{
							squirrel.Eq{"album_artist_id": artistIDs},
							squirrel.Or{
								squirrel.Expr("play_date IS NULL"),
								squirrel.Lt{"play_date": inspiredByCutoff},
							},
						},
					})
					if err != nil {
						return nil, err
					}
					maxPerArtist := int(math.Ceil(float64(limit) / float64(len(artistIDs))))
					if maxPerArtist < 2 {
						maxPerArtist = 2
					}
					return selectDiverseAlbums(pool, limit, now, excludedMixAlbumIDs, maxPerArtist, artistIDs), nil
				},
			},
			{
				ID:       "continueListening",
				Resource: "album",
				To:       "",
				Kind:     "history",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:  "play_date",
						Order: "DESC",
						Max:   limit,
						Filters: squirrel.And{
							squirrel.Gt{"play_count": 0},
							squirrel.GtOrEq{"play_date": continueListeningCutoff},
						},
					})
				},
			},
			{
				ID:       "recentlyPlayed",
				Resource: "album",
				To:       "/album/recentlyPlayed?sort=play_date&order=DESC&filter={\"recently_played\":true}",
				Kind:     "history",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:    "play_date",
						Order:   "DESC",
						Max:     limit,
						Filters: squirrel.Gt{"play_count": 0},
					})
				},
			},
			{
				ID:       "starred",
				Resource: "album",
				To:       "/album/starred?sort=starred_at&order=DESC&filter={\"starred\":true}",
				Kind:     "favorites",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:    "starred_at",
						Order:   "DESC",
						Max:     limit,
						Filters: squirrel.Gt{"starred": 0},
					})
				},
			},
			{
				ID:       "forgottenFavorites",
				Resource: "album",
				To:       "",
				Kind:     "favorites",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:  "starred_at",
						Order: "DESC",
						Max:   limit,
						Filters: squirrel.And{
							squirrel.Gt{"starred": 0},
							squirrel.Or{squirrel.Expr("play_date IS NULL"), squirrel.Lt{"play_date": rediscoverCutoff}},
						},
					})
				},
			},
			{
				ID:       "recentlyAdded",
				Resource: "album",
				To:       "/album/recentlyAdded?sort=recently_added&order=DESC&filter={}",
				Kind:     "library",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{Sort: "recently_added", Order: "DESC", Max: limit})
				},
			},
			{
				ID:       "newReleases",
				Resource: "album",
				To:       "",
				Kind:     "library",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:    "max_year",
						Order:   "DESC",
						Max:     limit,
						Filters: squirrel.Gt{"max_year": 0},
					})
				},
			},
			{
				ID:       "topRated",
				Resource: "album",
				To:       "",
				Kind:     "rated",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:    "rated_at",
						Order:   "DESC",
						Max:     limit,
						Filters: squirrel.Gt{"rating": 0},
					})
				},
			},
			{
				ID:       "mostPlayed",
				Resource: "album",
				To:       "/album/mostPlayed?sort=play_count&order=DESC&filter={\"recently_played\":true}",
				Kind:     "history",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:    "play_count",
						Order:   "DESC",
						Max:     limit,
						Filters: squirrel.Gt{"play_count": 0},
					})
				},
			},
			{
				ID:       "onRepeat",
				Resource: "album",
				To:       "",
				Kind:     "history",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:  "play_count",
						Order: "DESC",
						Max:   limit,
						Filters: squirrel.And{
							squirrel.Gt{"play_count": 0},
							squirrel.GtOrEq{"play_date": onRepeatCutoff},
						},
					})
				},
			},
			{
				ID:       "rediscover",
				Resource: "album",
				To:       "",
				Kind:     "history",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:  "play_count",
						Order: "DESC",
						Max:   limit,
						Filters: squirrel.And{
							squirrel.Gt{"play_count": 0},
							squirrel.Lt{"play_date": rediscoverCutoff},
						},
					})
				},
			},
			{
				ID:       "discoverFresh",
				Resource: "album",
				To:       "",
				Kind:     "discovery",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:  "recently_added",
						Order: "DESC",
						Max:   limit,
						Filters: squirrel.And{
							squirrel.Eq{"play_count": 0},
						},
					})
				},
			},
			{
				ID:       "random",
				Resource: "album",
				To:       "/album/random?sort=random&order=ASC&filter={}",
				Kind:     "discovery",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{Sort: "random", Order: "ASC", Max: limit, Seed: seed})
				},
			},
		}

		// Return all non-empty recommendation sections.
		// Keep a stable ordering (as declared in candidates) so daily mixes stay grouped.
		sections := make([]homeRecommendationsSection, 0, len(candidates))
		for _, cand := range candidates {
			items, err := cand.Build()
			if err != nil {
				log.Error(r.Context(), "Error building home recommendations", "section", cand.ID, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(items) == 0 {
				continue
			}
			if cand.Kind == "mix" {
				for _, a := range items {
					if a.ID != "" {
						excludedMixAlbumIDs[a.ID] = struct{}{}
					}
				}
			}
			sections = append(sections, homeRecommendationsSection{ID: cand.ID, Resource: cand.Resource, To: cand.To, Items: items})
		}

		resp := homeRecommendationsResponse{Sections: sections}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			log.Error(r.Context(), "Error encoding home recommendations", err)
		}
	}
}
