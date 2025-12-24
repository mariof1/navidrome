package nativeapi

import (
	"encoding/json"
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

		albumRepo := api.ds.Album(r.Context())
		recentlyPlayed, err := albumRepo.GetAll(model.QueryOptions{
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

		starred, err := albumRepo.GetAll(model.QueryOptions{
			Sort:    "starred_at",
			Order:   "DESC",
			Max:     limit,
			Filters: squirrel.Gt{"starred": 0},
		})
		if err != nil {
			log.Error(r.Context(), "Error building home recommendations", "section", "starred", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		recentlyAdded, err := albumRepo.GetAll(model.QueryOptions{
			Sort:  "recently_added",
			Order: "DESC",
			Max:   limit,
		})
		if err != nil {
			log.Error(r.Context(), "Error building home recommendations", "section", "recentlyAdded", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		mostPlayed, err := albumRepo.GetAll(model.QueryOptions{
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

		onRepeat, err := albumRepo.GetAll(model.QueryOptions{
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

		rediscover, err := albumRepo.GetAll(model.QueryOptions{
			Sort:  "play_count",
			Order: "DESC",
			Max:   limit,
			Filters: squirrel.And{
				squirrel.Gt{"play_count": 0},
				squirrel.Lt{"play_date": rediscoverCutoff},
			},
		})
		if err != nil {
			log.Error(r.Context(), "Error building home recommendations", "section", "rediscover", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		discoverFresh, err := albumRepo.GetAll(model.QueryOptions{
			Sort:  "recently_added",
			Order: "DESC",
			Max:   limit,
			Filters: squirrel.And{
				squirrel.Eq{"play_count": 0},
			},
		})
		if err != nil {
			log.Error(r.Context(), "Error building home recommendations", "section", "discoverFresh", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		random, err := albumRepo.GetAll(model.QueryOptions{
			Sort:  "random",
			Order: "ASC",
			Max:   limit,
			Seed:  seed,
		})
		if err != nil {
			log.Error(r.Context(), "Error building home recommendations", "section", "random", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Derive a small set of “seed” artists from the user's recent listening patterns.
		// Note: this is intentionally lightweight and based on existing per-user play_count/play_date.
		seenArtists := map[string]struct{}{}
		var seedArtistIDs []string
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
		appendArtists(onRepeat)
		appendArtists(recentlyPlayed)
		appendArtists(mostPlayed)

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
		dailyMix1, err := albumRepo.GetAll(model.QueryOptions{
			Sort:    "random",
			Order:   "ASC",
			Max:     limit,
			Seed:    seed + "-dm1",
			Filters: dailyMix1Filters,
		})
		if err != nil {
			log.Error(r.Context(), "Error building home recommendations", "section", "dailyMix1", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dailyMix2Filters := squirrel.Sqlizer(nil)
		if len(mix2IDs) > 0 {
			dailyMix2Filters = squirrel.Eq{"album_artist_id": mix2IDs}
		}
		dailyMix2, err := albumRepo.GetAll(model.QueryOptions{
			Sort:    "random",
			Order:   "ASC",
			Max:     limit,
			Seed:    seed + "-dm2",
			Filters: dailyMix2Filters,
		})
		if err != nil {
			log.Error(r.Context(), "Error building home recommendations", "section", "dailyMix2", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dailyMix3, err := albumRepo.GetAll(model.QueryOptions{
			Sort:  "random",
			Order: "ASC",
			Max:   limit,
			Seed:  seed + "-dm3",
			Filters: squirrel.Or{
				squirrel.Expr("play_date IS NULL"),
				squirrel.Lt{"play_date": rediscoverCutoff},
			},
		})
		if err != nil {
			log.Error(r.Context(), "Error building home recommendations", "section", "dailyMix3", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		inspiredBy := model.Albums{}
		if len(seedArtistIDs) > 0 {
			inspiredBy, err = albumRepo.GetAll(model.QueryOptions{
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
			if err != nil {
				log.Error(r.Context(), "Error building home recommendations", "section", "inspiredBy", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		resp := homeRecommendationsResponse{Sections: []homeRecommendationsSection{
			{
				ID:       "dailyMix1",
				Resource: "album",
				To:       "",
				Items:    dailyMix1,
			},
			{
				ID:       "dailyMix2",
				Resource: "album",
				To:       "",
				Items:    dailyMix2,
			},
			{
				ID:       "dailyMix3",
				Resource: "album",
				To:       "",
				Items:    dailyMix3,
			},
			{
				ID:       "inspiredBy",
				Resource: "album",
				To:       "",
				Items:    inspiredBy,
			},
			{
				ID:       "recentlyPlayed",
				Resource: "album",
				To:       "/album/recentlyPlayed?sort=play_date&order=DESC&filter={\"recently_played\":true}",
				Items:    recentlyPlayed,
			},
			{
				ID:       "starred",
				Resource: "album",
				To:       "/album/starred?sort=starred_at&order=DESC&filter={\"starred\":true}",
				Items:    starred,
			},
			{
				ID:       "recentlyAdded",
				Resource: "album",
				To:       "/album/recentlyAdded?sort=recently_added&order=DESC&filter={}",
				Items:    recentlyAdded,
			},
			{
				ID:       "mostPlayed",
				Resource: "album",
				To:       "/album/mostPlayed?sort=play_count&order=DESC&filter={\"recently_played\":true}",
				Items:    mostPlayed,
			},
			{
				ID:       "onRepeat",
				Resource: "album",
				To:       "",
				Items:    onRepeat,
			},
			{
				ID:       "rediscover",
				Resource: "album",
				To:       "",
				Items:    rediscover,
			},
			{
				ID:       "discoverFresh",
				Resource: "album",
				To:       "",
				Items:    discoverFresh,
			},
			{
				ID:       "random",
				Resource: "album",
				To:       "/album/random?sort=random&order=ASC&filter={}",
				Items:    random,
			},
		}}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			log.Error(r.Context(), "Error encoding home recommendations", err)
		}
	}
}
