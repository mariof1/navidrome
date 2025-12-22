package tests

import (
	"errors"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockPodcastRepo struct {
	model.PodcastRepository
	Channels map[string]*model.PodcastChannel
	Episodes map[string]model.PodcastEpisodes
	Statuses map[string]map[string]bool
	Progress map[string]map[string]struct {
		Position  int64
		Duration  int64
		UpdatedAt time.Time
	}
	Err      bool
}

func (m *MockPodcastRepo) ensureMaps() {
	if m.Channels == nil {
		m.Channels = map[string]*model.PodcastChannel{}
	}
	if m.Episodes == nil {
		m.Episodes = map[string]model.PodcastEpisodes{}
	}
	if m.Statuses == nil {
		m.Statuses = map[string]map[string]bool{}
	}
	if m.Progress == nil {
		m.Progress = map[string]map[string]struct {
			Position  int64
			Duration  int64
			UpdatedAt time.Time
		}{}
	}
}

func (m *MockPodcastRepo) CreateChannel(channel *model.PodcastChannel) error {
	if m.Err {
		return errors.New("error")
	}
	m.ensureMaps()
	if channel.ID == "" {
		channel.ID = id.NewRandom()
	}
	now := time.Now()
	channel.CreatedAt = now
	channel.UpdatedAt = now
	m.Channels[channel.ID] = channel
	return nil
}

func (m *MockPodcastRepo) UpdateChannel(channel *model.PodcastChannel) error {
	if m.Err {
		return errors.New("error")
	}
	m.ensureMaps()
	if _, ok := m.Channels[channel.ID]; !ok {
		return model.ErrNotFound
	}
	channel.UpdatedAt = time.Now()
	m.Channels[channel.ID] = channel
	return nil
}

func (m *MockPodcastRepo) DeleteChannel(id string) error {
	if m.Err {
		return errors.New("error")
	}
	m.ensureMaps()
	delete(m.Channels, id)
	delete(m.Episodes, id)
	return nil
}

func (m *MockPodcastRepo) GetChannel(id string) (*model.PodcastChannel, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	m.ensureMaps()
	if ch, ok := m.Channels[id]; ok {
		return ch, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockPodcastRepo) ListVisible(userID string) (model.PodcastChannels, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	m.ensureMaps()
	var res model.PodcastChannels
	for _, ch := range m.Channels {
		if ch.UserID == userID {
			res = append(res, *ch)
		}
	}
	return res, nil
}

func (m *MockPodcastRepo) SaveEpisodes(channelID string, episodes model.PodcastEpisodes) error {
	if m.Err {
		return errors.New("error")
	}
	m.ensureMaps()
	for i := range episodes {
		if episodes[i].ID == "" {
			episodes[i].ID = id.NewRandom()
		}
		episodes[i].ChannelID = channelID
		if episodes[i].CreatedAt.IsZero() {
			episodes[i].CreatedAt = time.Now()
		}
		if episodes[i].UpdatedAt.IsZero() {
			episodes[i].UpdatedAt = episodes[i].CreatedAt
		}
	}
	m.Episodes[channelID] = append(m.Episodes[channelID], episodes...)
	return nil
}

func (m *MockPodcastRepo) ListEpisodes(channelID string) (model.PodcastEpisodes, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	m.ensureMaps()
	return m.Episodes[channelID], nil
}

func (m *MockPodcastRepo) SetEpisodeStatus(userID, episodeID string, watched bool) error {
	if m.Err {
		return errors.New("error")
	}
	m.ensureMaps()
	if _, ok := m.Statuses[userID]; !ok {
		m.Statuses[userID] = map[string]bool{}
	}
	m.Statuses[userID][episodeID] = watched
	return nil
}

func (m *MockPodcastRepo) ListEpisodeStatuses(userID string, episodeIDs []string) (map[string]bool, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	m.ensureMaps()
	res := map[string]bool{}
	for _, id := range episodeIDs {
		if watched, ok := m.Statuses[userID][id]; ok {
			res[id] = watched
		}
	}
	return res, nil
}

func (m *MockPodcastRepo) SetEpisodeProgress(userID, episodeID string, position, duration int64) error {
	if m.Err {
		return errors.New("error")
	}
	m.ensureMaps()
	if _, ok := m.Progress[userID]; !ok {
		m.Progress[userID] = map[string]struct {
			Position  int64
			Duration  int64
			UpdatedAt time.Time
		}{}
	}
	m.Progress[userID][episodeID] = struct {
		Position  int64
		Duration  int64
		UpdatedAt time.Time
	}{Position: position, Duration: duration, UpdatedAt: time.Now()}
	return nil
}

func (m *MockPodcastRepo) GetEpisodeProgress(userID, episodeID string) (position, duration int64, updatedAt time.Time, err error) {
	if m.Err {
		return 0, 0, time.Time{}, errors.New("error")
	}
	m.ensureMaps()
	if m.Progress[userID] == nil {
		return 0, 0, time.Time{}, model.ErrNotFound
	}
	row, ok := m.Progress[userID][episodeID]
	if !ok {
		return 0, 0, time.Time{}, model.ErrNotFound
	}
	return row.Position, row.Duration, row.UpdatedAt, nil
}

func (m *MockPodcastRepo) ListContinueListening(userID string, limit int) ([]model.PodcastContinueItem, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	m.ensureMaps()
	return []model.PodcastContinueItem{}, nil
}
