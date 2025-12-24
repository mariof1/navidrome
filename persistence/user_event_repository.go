package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type userEventRepository struct {
	sqlRepository
}

func NewUserEventRepository(ctx context.Context, db dbx.Builder) model.UserEventRepository {
	r := &userEventRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "user_events"
	return r
}

func (r *userEventRepository) Record(event model.UserEvent) error {
	userID := event.UserID
	if userID == "" {
		userID = loggedUser(r.ctx).ID
	}

	values := map[string]any{
		"user_id":     userID,
		"event_type":  event.EventType,
		"entity_type": event.EntityType,
		"entity_id":   event.EntityID,
		"query":       event.Query,
		"player_id":   event.PlayerID,
		"position":    event.Position,
		"occurred_at": event.OccurredAt.UTC().Unix(),
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err := r.executeSQL(insert)
	return err
}
