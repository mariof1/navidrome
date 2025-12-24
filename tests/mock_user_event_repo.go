package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockUserEventRepo struct {
	Events []model.UserEvent
}

func (m *MockUserEventRepo) Record(event model.UserEvent) error {
	m.Events = append(m.Events, event)
	return nil
}

func (m *MockUserEventRepo) TopAlbumArtistIDs(limit int, now time.Time) ([]string, error) {
	return []string{}, nil
}

func CreateMockUserEventRepo() *MockUserEventRepo {
	return &MockUserEventRepo{}
}
