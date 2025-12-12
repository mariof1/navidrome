package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/podcast"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
)

type podcastChannelResponse struct {
	*model.PodcastChannel
	EpisodeCount int `json:"episodeCount,omitempty"`
}

type podcastCreatePayload struct {
	RSSUrl   string `json:"rssUrl"`
	IsGlobal bool   `json:"isGlobal"`
}

func (api *Router) addPodcastRoute(r chi.Router) {
	r.Route("/podcast", func(r chi.Router) {
		r.Get("/", api.listPodcasts())
		r.Post("/", api.createPodcast())
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", api.getPodcast())
			r.Put("/", api.updatePodcast())
			r.Delete("/", api.deletePodcast())
			r.Get("/episodes", api.listPodcastEpisodes())
			r.Put("/episodes/{episodeId}/watched", api.setEpisodeWatched())
		})
	})
}

func (api *Router) listPodcasts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := request.UserFrom(r.Context())

		channels, err := api.podcasts.ListChannelsForUser(&user)
		if err != nil {
			log.Error(r.Context(), "Error listing podcast channels", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		repo := api.ds.Podcast(r.Context())
		resp := make([]podcastChannelResponse, 0, len(channels))
		for i := range channels {
			episodes, err := repo.ListEpisodes(channels[i].ID)
			if err != nil {
				log.Error(r.Context(), "Error loading podcast episodes", "channelId", channels[i].ID, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			resp = append(resp, podcastChannelResponse{PodcastChannel: &channels[i], EpisodeCount: len(episodes)})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(r.Context(), "Error encoding podcast response", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (api *Router) createPodcast() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := request.UserFrom(r.Context())
		var payload podcastCreatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		payload.RSSUrl = strings.TrimSpace(payload.RSSUrl)
		if payload.RSSUrl == "" {
			http.Error(w, "rss url required", http.StatusBadRequest)
			return
		}

		channel, err := api.podcasts.AddChannel(r.Context(), payload.RSSUrl, &user, payload.IsGlobal)
		if err != nil {
			status := http.StatusInternalServerError
			msg := err.Error()
			var feedErr *podcast.FeedError
			if errors.As(err, &feedErr) {
				msg = feedErr.Message()
				switch feedErr.Code() {
				case podcast.ErrInvalidURL, podcast.ErrInvalidFeed:
					status = http.StatusBadRequest
				case podcast.ErrFetchFailed:
					status = http.StatusBadGateway
				}
			}
			log.Error(r.Context(), "Error creating podcast channel", "rssUrl", payload.RSSUrl, err)
			http.Error(w, msg, status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(channel); err != nil {
			log.Error(r.Context(), "Error encoding podcast response", err)
		}
	}
}

func (api *Router) getPodcast() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)
		id := chi.URLParam(r, "id")

		channel, err := api.podcasts.LoadChannelWithEpisodes(id)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, model.ErrNotFound) {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}

		if !canAccessPodcast(channel, user) {
			http.Error(w, "podcast not found", http.StatusNotFound)
			return
		}

		repo := api.ds.Podcast(ctx)
		channel.Episodes = applyWatchedStatuses(ctx, repo, user, channel.Episodes)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(channel); err != nil {
			log.Error(ctx, "Error encoding podcast response", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (api *Router) updatePodcast() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)
		id := chi.URLParam(r, "id")

		repo := api.ds.Podcast(ctx)
		channel, err := repo.GetChannel(id)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, model.ErrNotFound) {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}

		if !canManagePodcast(channel, user) {
			http.Error(w, "podcast not found", http.StatusNotFound)
			return
		}

		var payload podcastCreatePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		updatedChannel, err := api.podcasts.UpdateChannel(ctx, id, payload.RSSUrl, payload.IsGlobal)
		if err != nil {
			status := http.StatusInternalServerError
			msg := err.Error()
			var feedErr *podcast.FeedError
			if errors.As(err, &feedErr) {
				msg = feedErr.Message()
				switch feedErr.Code() {
				case podcast.ErrInvalidURL, podcast.ErrInvalidFeed:
					status = http.StatusBadRequest
				case podcast.ErrFetchFailed:
					status = http.StatusBadGateway
				}
			}
			log.Error(ctx, "Error updating podcast channel", "id", id, err)
			http.Error(w, msg, status)
			return
		}

		if len(updatedChannel.Episodes) == 0 {
			if episodes, err := repo.ListEpisodes(id); err == nil {
				updatedChannel.Episodes = episodes
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(updatedChannel); err != nil {
			log.Error(ctx, "Error encoding podcast response", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (api *Router) deletePodcast() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)
		id := chi.URLParam(r, "id")
		repo := api.ds.Podcast(ctx)

		channel, err := repo.GetChannel(id)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, model.ErrNotFound) {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}

		if !canManagePodcast(channel, user) {
			http.Error(w, "podcast not found", http.StatusNotFound)
			return
		}

		if err := repo.DeleteChannel(id); err != nil {
			log.Error(ctx, "Error deleting podcast", "id", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (api *Router) listPodcastEpisodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)
		id := chi.URLParam(r, "id")
		repo := api.ds.Podcast(ctx)

		channel, err := repo.GetChannel(id)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, model.ErrNotFound) {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}

		if !canAccessPodcast(channel, user) {
			http.Error(w, "podcast not found", http.StatusNotFound)
			return
		}

		episodes, err := repo.ListEpisodes(id)
		if err != nil {
			log.Error(ctx, "Error listing podcast episodes", "id", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		episodes = applyWatchedStatuses(ctx, repo, user, episodes)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(episodes); err != nil {
			log.Error(ctx, "Error encoding podcast episodes", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

type episodeStatusPayload struct {
	Watched bool `json:"watched"`
}

func (api *Router) setEpisodeWatched() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)
		channelID := chi.URLParam(r, "id")
		episodeID := chi.URLParam(r, "episodeId")
		repo := api.ds.Podcast(ctx)

		channel, err := repo.GetChannel(channelID)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, model.ErrNotFound) {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}

		if !canAccessPodcast(channel, user) {
			http.Error(w, "podcast not found", http.StatusNotFound)
			return
		}

		var payload episodeStatusPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := repo.SetEpisodeStatus(user.ID, episodeID, payload.Watched); err != nil {
			log.Error(ctx, "Error updating episode status", "episodeId", episodeID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func applyWatchedStatuses(ctx context.Context, repo model.PodcastRepository, user model.User, episodes model.PodcastEpisodes) model.PodcastEpisodes {
	ids := make([]string, 0, len(episodes))
	for _, ep := range episodes {
		ids = append(ids, ep.ID)
	}
	statuses, err := repo.ListEpisodeStatuses(user.ID, ids)
	if err != nil {
		log.Error(ctx, "Error loading episode statuses", err)
		return episodes
	}
	for i := range episodes {
		if watched, ok := statuses[episodes[i].ID]; ok {
			episodes[i].Watched = watched
		}
	}
	return episodes
}

func canAccessPodcast(channel *model.PodcastChannel, user model.User) bool {
	return channel.IsGlobal || user.IsAdmin || channel.UserID == user.ID
}

func canManagePodcast(channel *model.PodcastChannel, user model.User) bool {
	return user.IsAdmin || channel.UserID == user.ID
}
