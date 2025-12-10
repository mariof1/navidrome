package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
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
}

type sqlPodcastRepository struct {
	sqlRepository
}

func NewPodcastRepository(ctx context.Context, db dbx.Builder) PodcastRepository {
	r := &sqlPodcastRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel((*model.PodcastChannel)(nil), map[string]filterFunc{})
	return r
}

func (r *sqlPodcastRepository) CreateChannel(channel *model.PodcastChannel) error {
	id, err := r.insert(r.tableName, channel)
	if err != nil {
		return err
	}
	channel.ID = id
	return nil
}

func (r *sqlPodcastRepository) UpdateChannel(channel *model.PodcastChannel) error {
	return r.update(r.tableName, channel)
}

func (r *sqlPodcastRepository) DeleteChannel(id string) error {
	_, err := r.db.Delete(r.tableName, Eq{"id": id}).Execute()
	if err != nil {
		return err
	}
	_, err = r.db.Delete("podcast_episode", Eq{"channel_id": id}).Execute()
	return err
}

func (r *sqlPodcastRepository) GetChannel(id string) (*model.PodcastChannel, error) {
	var channel model.PodcastChannel
	err := r.db.Select("*").From(r.tableName).Where(Eq{"id": id}).One(&channel)
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

func (r *sqlPodcastRepository) ListVisible(userID string, includeGlobal bool) (model.PodcastChannels, error) {
	stmt := r.db.Select("*").From(r.tableName)
	if includeGlobal {
		stmt = stmt.Where(Or{Eq{"user_id": userID}, Eq{"is_global": true}})
	} else {
		stmt = stmt.Where(Eq{"user_id": userID})
	}
	var channels model.PodcastChannels
	err := stmt.All(&channels)
	return channels, err
}

func (r *sqlPodcastRepository) SaveEpisodes(channelID string, episodes model.PodcastEpisodes) error {
	for i := range episodes {
		episodes[i].ChannelID = channelID
		_, err := r.db.NewQuery("INSERT OR IGNORE INTO podcast_episode(channel_id, guid, title, description, audio_url, mime_type, duration, published_at, image_url) VALUES ({:channel_id}, {:guid}, {:title}, {:description}, {:audio_url}, {:mime_type}, {:duration}, {:published_at}, {:image_url})").BindStruct(episodes[i]).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *sqlPodcastRepository) ListEpisodes(channelID string) (model.PodcastEpisodes, error) {
	var episodes model.PodcastEpisodes
	err := r.db.Select("*").From("podcast_episode").Where(Eq{"channel_id": channelID}).OrderBy("published_at desc").All(&episodes)
	return episodes, err
}
