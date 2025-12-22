package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	RSSUrl string `json:"rssUrl"`
}

type podcastProgressPayload struct {
	Position int64 `json:"position"`
	Duration int64 `json:"duration"`
}

type podcastProgressResponse struct {
	Position  int64  `json:"position"`
	Duration  int64  `json:"duration"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type applePodcastSearchResult struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	FeedURL  string `json:"feedUrl"`
	ImageURL string `json:"imageUrl"`
	SiteURL  string `json:"siteUrl"`
}

func (api *Router) addPodcastRoute(r chi.Router) {
	r.Route("/podcast", func(r chi.Router) {
		r.Get("/continue", api.listContinueListening())
		r.Get("/search", api.searchApplePodcasts())
		r.Get("/", api.listPodcasts())
		r.Post("/", api.createPodcast())
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", api.getPodcast())
			r.Put("/", api.updatePodcast())
			r.Delete("/", api.deletePodcast())
			r.Get("/episodes", api.listPodcastEpisodes())
			r.Put("/episodes/{episodeId}/watched", api.setEpisodeWatched())
			r.Get("/episodes/{episodeId}/progress", api.getEpisodeProgress())
			r.Put("/episodes/{episodeId}/progress", api.setEpisodeProgress())
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

		channel, err := api.podcasts.AddChannel(r.Context(), payload.RSSUrl, &user)
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

		updatedChannel, err := api.podcasts.UpdateChannel(ctx, id, payload.RSSUrl)
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
	return user.IsAdmin || channel.UserID == user.ID
}

func canManagePodcast(channel *model.PodcastChannel, user model.User) bool {
	return user.IsAdmin || channel.UserID == user.ID
}

func (api *Router) setEpisodeProgress() http.HandlerFunc {
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

		var payload podcastProgressPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if payload.Position < 0 {
			payload.Position = 0
		}
		if payload.Duration < 0 {
			payload.Duration = 0
		}

		if err := repo.SetEpisodeProgress(user.ID, episodeID, payload.Position, payload.Duration); err != nil {
			log.Error(ctx, "Error saving episode progress", "episodeId", episodeID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (api *Router) getEpisodeProgress() http.HandlerFunc {
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

		pos, dur, updatedAt, err := repo.GetEpisodeProgress(user.ID, episodeID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(podcastProgressResponse{Position: 0, Duration: 0})
				return
			}
			log.Error(ctx, "Error loading episode progress", "episodeId", episodeID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := podcastProgressResponse{Position: pos, Duration: dur}
		if !updatedAt.IsZero() {
			resp.UpdatedAt = updatedAt.Format(time.RFC3339)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(ctx, "Error encoding progress response", err)
		}
	}
}

func (api *Router) listContinueListening() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)
		limit := 20
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}

		repo := api.ds.Podcast(ctx)
		items, err := repo.ListContinueListening(user.ID, limit)
		if err != nil {
			log.Error(ctx, "Error listing continue listening", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(items); err != nil {
			log.Error(ctx, "Error encoding continue listening response", err)
		}
	}
}

func (api *Router) searchApplePodcasts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		term := strings.TrimSpace(r.URL.Query().Get("term"))
		if term == "" {
			http.Error(w, "term required", http.StatusBadRequest)
			return
		}
		limit := 25
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}

		results, err := doApplePodcastSearch(ctx, term, limit)
		if err != nil {
			log.Error(ctx, "Apple podcast search failed", "term", term, err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			log.Error(ctx, "Error encoding search response", err)
		}
	}
}

func doApplePodcastSearch(ctx context.Context, term string, limit int) ([]applePodcastSearchResult, error) {
	// iTunes Search API (Apple Podcasts) returns RSS feed URL in `feedUrl`.
	type itunesResult struct {
		TrackName         string `json:"trackName"`
		ArtistName        string `json:"artistName"`
		FeedURL           string `json:"feedUrl"`
		ArtworkURL600     string `json:"artworkUrl600"`
		ArtworkURL100     string `json:"artworkUrl100"`
		CollectionViewURL string `json:"collectionViewUrl"`
	}
	type itunesResponse struct {
		ResultCount int           `json:"resultCount"`
		Results     []itunesResult `json:"results"`
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://itunes.apple.com/search", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("term", term)
	q.Set("media", "podcast")
	q.Set("entity", "podcast")
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("apple search request failed")
	}

	var parsed itunesResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	out := make([]applePodcastSearchResult, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		if strings.TrimSpace(r.FeedURL) == "" {
			continue
		}
		img := r.ArtworkURL600
		if img == "" {
			img = r.ArtworkURL100
		}
		out = append(out, applePodcastSearchResult{
			Title:    r.TrackName,
			Author:   r.ArtistName,
			FeedURL:  r.FeedURL,
			ImageURL: img,
			SiteURL:  r.CollectionViewURL,
		})
	}
	return out, nil
}
