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
	ListVisible(userID string) (model.PodcastChannels, error)
	SaveEpisodes(channelID string, episodes model.PodcastEpisodes) error
	ListEpisodes(channelID string) (model.PodcastEpisodes, error)
	GetEpisode(id string) (*model.PodcastEpisode, error)
	SetEpisodeStatus(userID, episodeID string, watched bool) error
	ListEpisodeStatuses(userID string, episodeIDs []string) (map[string]bool, error)
	SetEpisodeProgress(userID, episodeID string, position, duration int64) error
	GetEpisodeProgress(userID, episodeID string) (position, duration int64, updatedAt time.Time, err error)
	ListContinueListening(userID string, limit int) ([]model.PodcastContinueItem, error)
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
	if _, err := r.executeSQL(Delete("podcast_episode").Where(Eq{"channel_id": id})); err != nil {
		return err
	}
	return r.delete(Eq{"id": id})
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

func (r *sqlPodcastRepository) ListVisible(userID string) (model.PodcastChannels, error) {
	sel := r.newSelect().Columns("*")
	// Podcasts are per-user subscriptions. Global/shared channels are deprecated.
	sel = sel.Where(Eq{"podcast_channel.user_id": userID})
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

func (r *sqlPodcastRepository) SetEpisodeStatus(userID, episodeID string, watched bool) error {
	now := time.Now()
	sq := Insert("podcast_episode_status").
		Columns("user_id", "episode_id", "watched", "updated_at").
		Values(userID, episodeID, watched, now).
		Suffix("on conflict(user_id, episode_id) do update set watched=excluded.watched, updated_at=excluded.updated_at")
	if _, err := r.executeSQL(sq); err != nil {
		return err
	}
	// If an episode is marked watched, clear any stored progress.
	if watched {
		_, _ = r.executeSQL(Delete("podcast_episode_progress").Where(Eq{"user_id": userID, "episode_id": episodeID}))
	}
	return nil
}

func (r *sqlPodcastRepository) ListEpisodeStatuses(userID string, episodeIDs []string) (map[string]bool, error) {
	if len(episodeIDs) == 0 {
		return map[string]bool{}, nil
	}
	var results []struct {
		EpisodeID string `db:"episode_id"`
		Watched   bool   `db:"watched"`
	}
	sq := Select("episode_id", "watched").From("podcast_episode_status").
		Where(And{Eq{"user_id": userID}, Eq{"episode_id": episodeIDs}})
	if err := r.queryAll(sq, &results); err != nil {
		if err == model.ErrNotFound {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	statuses := make(map[string]bool, len(results))
	for _, row := range results {
		statuses[row.EpisodeID] = row.Watched
	}
	return statuses, nil
}

func (r *sqlPodcastRepository) SetEpisodeProgress(userID, episodeID string, position, duration int64) error {
	now := time.Now()
	if position < 0 {
		position = 0
	}
	if duration < 0 {
		duration = 0
	}
	sq := Insert("podcast_episode_progress").
		Columns("user_id", "episode_id", "position", "duration", "updated_at").
		Values(userID, episodeID, position, duration, now).
		Suffix("on conflict(user_id, episode_id) do update set position=excluded.position, duration=excluded.duration, updated_at=excluded.updated_at")
	_, err := r.executeSQL(sq)
	return err
}

func (r *sqlPodcastRepository) GetEpisodeProgress(userID, episodeID string) (position, duration int64, updatedAt time.Time, err error) {
	var row struct {
		Position  int64     `db:"position"`
		Duration  int64     `db:"duration"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	sq := Select("position", "duration", "updated_at").From("podcast_episode_progress").Where(Eq{"user_id": userID, "episode_id": episodeID})
	if err := r.queryOne(sq, &row); err != nil {
		return 0, 0, time.Time{}, err
	}
	return row.Position, row.Duration, row.UpdatedAt, nil
}

func (r *sqlPodcastRepository) ListContinueListening(userID string, limit int) ([]model.PodcastContinueItem, error) {
	if limit <= 0 {
		limit = 20
	}

	sq := Select(
		"pep.episode_id as episode_id",
		"pep.position as position",
		"pep.duration as duration",
		"pep.updated_at as progress_updated_at",

		"pe.id as pe_id",
		"pe.channel_id as pe_channel_id",
		"pe.guid as pe_guid",
		"pe.title as pe_title",
		"pe.description as pe_description",
		"pe.audio_url as pe_audio_url",
		"pe.mime_type as pe_mime_type",
		"pe.duration as pe_duration",
		"pe.published_at as pe_published_at",
		"pe.image_url as pe_image_url",
		"pe.created_at as pe_created_at",
		"pe.updated_at as pe_updated_at",

		"pc.id as pc_id",
		"pc.title as pc_title",
		"pc.rss_url as pc_rss_url",
		"pc.site_url as pc_site_url",
		"pc.description as pc_description",
		"pc.image_url as pc_image_url",
		"pc.user_id as pc_user_id",
		"pc.is_global as pc_is_global",
		"pc.created_at as pc_created_at",
		"pc.updated_at as pc_updated_at",
		"pc.last_refreshed_at as pc_last_refreshed_at",
		"pc.last_error as pc_last_error",
	).
		From("podcast_episode_progress pep").
		Join("podcast_episode pe on pe.id = pep.episode_id").
		Join("podcast_channel pc on pc.id = pe.channel_id").
		LeftJoin("podcast_episode_status pes on pes.user_id = pep.user_id and pes.episode_id = pep.episode_id").
		Where(And{Eq{"pep.user_id": userID}, Or{Eq{"pes.watched": nil}, Eq{"pes.watched": 0}}}).
		OrderBy("pep.updated_at desc").
		Limit(uint64(limit))

	var rows []struct {
		EpisodeID        string    `db:"episode_id"`
		Position         int64     `db:"position"`
		Duration         int64     `db:"duration"`
		ProgressUpdated  time.Time `db:"progress_updated_at"`
		PEID            string    `db:"pe_id"`
		PEChannelID     string    `db:"pe_channel_id"`
		PEGUID          string    `db:"pe_guid"`
		PETitle         string    `db:"pe_title"`
		PEDescription   string    `db:"pe_description"`
		PEAudioURL      string    `db:"pe_audio_url"`
		PEMimeType      string    `db:"pe_mime_type"`
		PEDuration      int64     `db:"pe_duration"`
		PEPublishedAt   time.Time `db:"pe_published_at"`
		PEImageURL      string    `db:"pe_image_url"`
		PECreatedAt     time.Time `db:"pe_created_at"`
		PEUpdatedAt     time.Time `db:"pe_updated_at"`
		PCID            string     `db:"pc_id"`
		PCTitle         string     `db:"pc_title"`
		PCRSSURL        string     `db:"pc_rss_url"`
		PCSiteURL       string     `db:"pc_site_url"`
		PCDescription   string     `db:"pc_description"`
		PCImageURL      string     `db:"pc_image_url"`
		PCUserID        string     `db:"pc_user_id"`
		PCIsGlobal      bool       `db:"pc_is_global"`
		PCCreatedAt     time.Time  `db:"pc_created_at"`
		PCUpdatedAt     time.Time  `db:"pc_updated_at"`
		PCLastRefreshed *time.Time `db:"pc_last_refreshed_at"`
		PCLastError     string     `db:"pc_last_error"`
	}

	if err := r.queryAll(sq, &rows); err != nil {
		if err == model.ErrNotFound {
			return []model.PodcastContinueItem{}, nil
		}
		return nil, err
	}

	items := make([]model.PodcastContinueItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.PodcastContinueItem{
			Channel: model.PodcastChannel{
				ID:              row.PCID,
				Title:           row.PCTitle,
				RSSURL:          row.PCRSSURL,
				SiteURL:         row.PCSiteURL,
				Description:     row.PCDescription,
				ImageURL:        row.PCImageURL,
				UserID:          row.PCUserID,
				IsGlobal:        row.PCIsGlobal,
				CreatedAt:       row.PCCreatedAt,
				UpdatedAt:       row.PCUpdatedAt,
				LastRefreshedAt: row.PCLastRefreshed,
				LastError:       row.PCLastError,
			},
			Episode: model.PodcastEpisode{
				ID:          row.PEID,
				ChannelID:   row.PEChannelID,
				GUID:        row.PEGUID,
				Title:       row.PETitle,
				Description: row.PEDescription,
				AudioURL:    row.PEAudioURL,
				MimeType:    row.PEMimeType,
				Duration:    row.PEDuration,
				PublishedAt: row.PEPublishedAt,
				ImageURL:    row.PEImageURL,
				CreatedAt:   row.PECreatedAt,
				UpdatedAt:   row.PEUpdatedAt,
			},
			Position:  row.Position,
			Duration:  row.Duration,
			UpdatedAt: row.ProgressUpdated,
		})
	}

	return items, nil
}
