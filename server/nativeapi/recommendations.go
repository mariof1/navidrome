package nativeapi

import (
	"encoding/json"
	"net/http"
	"strconv"

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

		resp := homeRecommendationsResponse{Sections: []homeRecommendationsSection{
			{
				ID:       "recentlyPlayed",
				Resource: "album",
				To:       "/album/recentlyPlayed?sort=play_date&order=DESC&filter={\"recently_played\":true}",
				Items:    recentlyPlayed,
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
