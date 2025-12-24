package nativeapi

import (
	"encoding/json"
	"hash/fnv"
	"math/rand"
	"net/http"
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
					return albumRepo.GetAll(model.QueryOptions{
						Sort:    "random",
						Order:   "ASC",
						Max:     limit,
						Seed:    dailyMix1Seed,
						Filters: dailyMix1Filters,
					})
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
					return albumRepo.GetAll(model.QueryOptions{
						Sort:    "random",
						Order:   "ASC",
						Max:     limit,
						Seed:    dailyMix2Seed,
						Filters: dailyMix2Filters,
					})
				},
			},
			{
				ID:       "dailyMix3",
				Resource: "album",
				To:       "",
				Kind:     "mix",
				Build: func() (model.Albums, error) {
					return albumRepo.GetAll(model.QueryOptions{
						Sort:  "random",
						Order: "ASC",
						Max:   limit,
						Seed:  dailyMix3Seed,
						Filters: squirrel.Or{
							squirrel.Expr("play_date IS NULL"),
							squirrel.Lt{"play_date": rediscoverCutoff},
						},
					})
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
					return albumRepo.GetAll(model.QueryOptions{
						Sort:  "random",
						Order: "ASC",
						Max:   limit,
						Seed:  seed + "-inspired",
						Filters: squirrel.And{
							squirrel.Eq{"album_artist_id": seedArtistIDs[0]},
							squirrel.Or{
								squirrel.Expr("play_date IS NULL"),
								squirrel.Lt{"play_date": inspiredByCutoff},
							},
						},
					})
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

		// Curate buckets to avoid flooding the Home UI.
		maxSections := 8
		pinned := []string{"dailyMix1"}

		selectedIDs := curateHomeSectionIDs(candidates, pinned, maxSections, seed)
		sections := make([]homeRecommendationsSection, 0, len(selectedIDs))
		for _, id := range selectedIDs {
			cand, ok := findCandidate(candidates, id)
			if !ok {
				continue
			}
			items, err := cand.Build()
			if err != nil {
				log.Error(r.Context(), "Error building home recommendations", "section", cand.ID, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(items) == 0 {
				continue
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

func findCandidate(cands []homeSectionCandidate, id string) (homeSectionCandidate, bool) {
	for _, c := range cands {
		if c.ID == id {
			return c, true
		}
	}
	return homeSectionCandidate{}, false
}

func curateHomeSectionIDs(cands []homeSectionCandidate, pinned []string, maxSections int, seed string) []string {
	if maxSections <= 0 {
		return nil
	}

	selected := make([]string, 0, maxSections)
	selectedSet := map[string]struct{}{}
	kindCount := map[string]int{}

	add := func(id string) {
		if len(selected) >= maxSections {
			return
		}
		if _, ok := selectedSet[id]; ok {
			return
		}
		cand, ok := findCandidate(cands, id)
		if !ok {
			return
		}
		// Lightweight diversity caps to avoid showing too many similar buckets.
		if cand.Kind == "mix" && kindCount["mix"] >= 2 {
			return
		}
		if cand.Kind == "favorites" && kindCount["favorites"] >= 1 {
			return
		}
		if cand.Kind == "rated" && kindCount["rated"] >= 1 {
			return
		}
		selected = append(selected, id)
		selectedSet[id] = struct{}{}
		kindCount[cand.Kind]++
	}

	for _, id := range pinned {
		add(id)
	}

	optional := make([]homeSectionCandidate, 0, len(cands))
	for _, c := range cands {
		if _, ok := selectedSet[c.ID]; ok {
			continue
		}
		optional = append(optional, c)
	}

	// Deterministic shuffle based on seed to keep Home stable per UI load.
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	rng.Shuffle(len(optional), func(i, j int) { optional[i], optional[j] = optional[j], optional[i] })

	for _, c := range optional {
		if len(selected) >= maxSections {
			break
		}
		add(c.ID)
	}

	return selected
}
