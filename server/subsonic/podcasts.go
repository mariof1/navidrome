package subsonic

import (
	"net/http"
	"sort"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) GetPodcasts(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	channels, err := api.podcasts.ListChannelsForUser(&user)
	if err != nil {
		return nil, err
	}

	repo := api.ds.Podcast(r.Context())
	response := newResponse()
	podcasts := responses.Podcasts{}

	for i := range channels {
		episodes, err := repo.ListEpisodes(channels[i].ID)
		if err != nil {
			return nil, err
		}
		podcasts.Channel = append(podcasts.Channel, mapPodcastChannel(&channels[i], episodes))
	}

	response.Podcasts = &podcasts
	return response, nil
}

func (api *Router) GetNewestPodcasts(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())
	p := req.Params(r)
	count := int(p.Int64Or("count", 20))

	channels, err := api.podcasts.ListChannelsForUser(&user)
	if err != nil {
		return nil, err
	}

	repo := api.ds.Podcast(r.Context())
	type channelEpisode struct {
		channel model.PodcastChannel
		episode model.PodcastEpisode
	}
	var items []channelEpisode

	for i := range channels {
		eps, err := repo.ListEpisodes(channels[i].ID)
		if err != nil {
			return nil, err
		}
		for j := range eps {
			items = append(items, channelEpisode{channel: channels[i], episode: eps[j]})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		left := items[i].episode.PublishedAt
		right := items[j].episode.PublishedAt
		if left.Equal(right) {
			return items[i].episode.CreatedAt.After(items[j].episode.CreatedAt)
		}
		return left.After(right)
	})

	if count > 0 && len(items) > count {
		items = items[:count]
	}

	mapped := make(map[string]*responses.PodcastChannel)
	var channelOrder []string
	for i := range items {
		ch := mapped[items[i].channel.ID]
		if ch == nil {
			channel := mapPodcastChannel(&items[i].channel, nil)
			ch = &channel
			mapped[items[i].channel.ID] = ch
			channelOrder = append(channelOrder, items[i].channel.ID)
		}
		ch.Episode = append(ch.Episode, mapPodcastEpisode(items[i].episode, &items[i].channel))
	}

	podcasts := responses.Podcasts{}
	for _, id := range channelOrder {
		podcasts.Channel = append(podcasts.Channel, *mapped[id])
	}

	response := newResponse()
	response.Podcasts = &podcasts
	return response, nil
}

func (api *Router) RefreshPodcasts(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	channels, err := api.podcasts.ListChannelsForUser(&user)
	if err != nil {
		return nil, err
	}

	for i := range channels {
		if err := api.podcasts.RefreshChannel(r.Context(), channels[i].ID); err != nil {
			return nil, err
		}
	}

	return newResponse(), nil
}

func (api *Router) DownloadPodcastEpisode(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	repo := api.ds.Podcast(r.Context())
	episode, err := repo.GetEpisode(id)
	if err != nil {
		return nil, err
	}

	channel, err := repo.GetChannel(episode.ChannelID)
	if err != nil {
		return nil, err
	}

	user, _ := request.UserFrom(r.Context())
	if !canAccessPodcast(channel, user) {
		return nil, newError(responses.ErrorDataNotFound, "podcast not found")
	}

	http.Redirect(w, r, episode.AudioURL, http.StatusFound)
	return nil, nil
}

func mapPodcastChannel(channel *model.PodcastChannel, episodes model.PodcastEpisodes) responses.PodcastChannel {
	var lastUpdate *time.Time
	switch {
	case channel.LastRefreshedAt != nil && !channel.LastRefreshedAt.IsZero():
		lastUpdate = channel.LastRefreshedAt
	case !channel.UpdatedAt.IsZero():
		lastUpdate = &channel.UpdatedAt
	}

	mapped := responses.PodcastChannel{
		ID:               channel.ID,
		Title:            channel.Title,
		Url:              channel.RSSURL,
		Description:      channel.Description,
		Status:           "completed",
		CoverArt:         "",
		OriginalImageUrl: channel.ImageURL,
		LastUpdate:       lastUpdate,
	}

	for i := range episodes {
		mapped.Episode = append(mapped.Episode, mapPodcastEpisode(episodes[i], channel))
	}

	return mapped
}

func mapPodcastEpisode(episode model.PodcastEpisode, channel *model.PodcastChannel) responses.PodcastEpisode {
	var publishDate *time.Time
	if !episode.PublishedAt.IsZero() {
		publishDate = &episode.PublishedAt
	}

	cover := episode.ImageURL
	if cover == "" {
		cover = channel.ImageURL
	}

	return responses.PodcastEpisode{
		ID:               episode.ID,
		StreamID:         episode.ID,
		ChannelID:        episode.ChannelID,
		Title:            episode.Title,
		Description:      episode.Description,
		Status:           "completed",
		CoverArt:         "",
		OriginalImageUrl: cover,
		PublishDate:      publishDate,
		Duration:         episode.Duration,
		IsVideo:          false,
		OriginalUrl:      episode.AudioURL,
		AudioUrl:         episode.AudioURL,
	}
}

func canAccessPodcast(channel *model.PodcastChannel, user model.User) bool {
	return user.IsAdmin || channel.UserID == user.ID
}
