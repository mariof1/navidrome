package model

import "time"

type UserEvent struct {
	UserID     string
	EventType  string
	EntityType string
	EntityID   string
	Query      string
	PlayerID   string
	Position   int
	OccurredAt time.Time
}

type UserEventRepository interface {
	Record(event UserEvent) error
	TopAlbumArtistIDs(limit int, now time.Time) ([]string, error)
}
