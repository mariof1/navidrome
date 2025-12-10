package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	id "github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

// PodcastRepository handles podcast channel and episode storage.
type PodcastRepository interface {
	CreateChannel(channel *model.PodcastChannel) error
	UpdateChannel(channel *model.PodcastChannel) error
	DeleteChannel(id string) error
	GetChannel(id string) (*model.PodcastChannel, error)
	ListVisible(userID string, includeGlobal bool) (model.PodcastChannels, error)
	SaveEpisodes(channelID string, episodes model.PodcastEpisodes) error
	ListEpisodes(channelID string) (model.PodcastEpisodes, error)
	GetEpisode(id string) (*model.PodcastEpisode, error)
}

type sqlPodcastRepository struct {
	sqlRepository
}

func NewPodcastRepository(ctx context.Context, db dbx.Builder) PodcastRepository {
	r := &sqlPodcastRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.PodcastChannel{}, nil)
	return r
}

func (r *sqlPodcastRepository) CreateChannel(channel *model.PodcastChannel) error {
	now := time.Now()
	channel.CreatedAt = now
	channel.UpdatedAt = now

	id, err := r.put(channel.ID, channel)
	if err != nil {
		return err
	}
	channel.ID = id
	return nil
}

func (r *sqlPodcastRepository) UpdateChannel(channel *model.PodcastChannel) error {
	channel.UpdatedAt = time.Now()
	_, err := r.put(channel.ID, channel)
	return err
}

func (r *sqlPodcastRepository) DeleteChannel(id string) error {
	if err := r.delete(Eq{"id": id}); err != nil {
		return err
	}
	_, err := r.executeSQL(Delete("podcast_episode").Where(Eq{"channel_id": id}))
	return err
}

func (r *sqlPodcastRepository) GetChannel(id string) (*model.PodcastChannel, error) {
	var channel model.PodcastChannel
	sel := r.newSelect().Columns("*").Where(Eq{"podcast_channel.id": id})
	err := r.queryOne(sel, &channel)
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

func (r *sqlPodcastRepository) ListVisible(userID string, includeGlobal bool) (model.PodcastChannels, error) {
	sel := r.newSelect().Columns("*")
	if includeGlobal {
		sel = sel.Where(Or{Eq{"podcast_channel.user_id": userID}, Eq{"podcast_channel.is_global": true}})
	} else {
		sel = sel.Where(Eq{"podcast_channel.user_id": userID})
	}
	var channels model.PodcastChannels
	err := r.queryAll(sel, &channels)
	return channels, err
}

func (r *sqlPodcastRepository) SaveEpisodes(channelID string, episodes model.PodcastEpisodes) error {
	for i := range episodes {
		episodes[i].ChannelID = channelID
		if episodes[i].ID == "" {
			episodes[i].ID = id.NewRandom()
		}
		now := time.Now()
		episodes[i].CreatedAt = now
		episodes[i].UpdatedAt = now

		sq := Insert("podcast_episode").
			Columns("id", "channel_id", "guid", "title", "description", "audio_url", "mime_type", "duration", "published_at", "image_url", "created_at", "updated_at").
			Values(
				episodes[i].ID,
				channelID,
				episodes[i].GUID,
				episodes[i].Title,
				episodes[i].Description,
				episodes[i].AudioURL,
				episodes[i].MimeType,
				episodes[i].Duration,
				episodes[i].PublishedAt,
				episodes[i].ImageURL,
				episodes[i].CreatedAt,
				episodes[i].UpdatedAt,
			).
			Suffix("on conflict (channel_id, guid) do nothing")
		if _, err := r.executeSQL(sq); err != nil {
			return err
		}
	}
	return nil
}

func (r *sqlPodcastRepository) ListEpisodes(channelID string) (model.PodcastEpisodes, error) {
	var episodes model.PodcastEpisodes
	sel := Select("*").From("podcast_episode").Where(Eq{"channel_id": channelID}).OrderBy("published_at desc")
	err := r.queryAll(sel, &episodes)
	return episodes, err
}

func (r *sqlPodcastRepository) GetEpisode(id string) (*model.PodcastEpisode, error) {
	var episode model.PodcastEpisode
	sel := Select("*").From("podcast_episode").Where(Eq{"podcast_episode.id": id})
	err := r.queryOne(sel, &episode)
	if err != nil {
		return nil, err
	}
	return &episode, nil
}
