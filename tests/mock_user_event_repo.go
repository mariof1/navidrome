package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockUserEventRepo struct {
	Events []model.UserEvent

	TopArtistIDs []string
	TopArtistErr error
}

func (m *MockUserEventRepo) Record(event model.UserEvent) error {
	m.Events = append(m.Events, event)
	return nil
}

func (m *MockUserEventRepo) TopAlbumArtistIDs(limit int, now time.Time) ([]string, error) {
	if m.TopArtistErr != nil {
		return nil, m.TopArtistErr
	}
	if len(m.TopArtistIDs) == 0 {
		return []string{}, nil
	}
	if limit <= 0 || limit >= len(m.TopArtistIDs) {
		return append([]string{}, m.TopArtistIDs...), nil
	}
	return append([]string{}, m.TopArtistIDs[:limit]...), nil
}

func CreateMockUserEventRepo() *MockUserEventRepo {
	return &MockUserEventRepo{}
}
