package persistence

import (
	"context"
	"time"

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

func (r *userEventRepository) TopAlbumArtistIDs(limit int, now time.Time) ([]string, error) {
	if limit <= 0 {
		limit = 1
	}

	userID := loggedUser(r.ctx).ID
	if userID == invalidUserId {
		return nil, nil
	}

	// Consider only recent history for responsiveness.
	windowStart := now.Add(-30 * 24 * time.Hour).UTC().Unix()
	nowUnix := now.UTC().Unix()
	if nowUnix <= 0 {
		nowUnix = time.Now().UTC().Unix()
	}

	// Event scoring (simple + time-decayed):
	// - scrobble: +1
	// - repeat: +2
	// - skip: -1
	// decay: 1 / (1 + ageDays)
	// Map song -> album_artist via media_file table.
	scoreExpr := "((case user_events.event_type " +
		"when 'repeat' then 2.0 " +
		"when 'scrobble' then 1.0 " +
		"when 'skip' then -1.0 " +
		"else 0.0 end) / (1.0 + ((? - user_events.occurred_at) / 86400.0)))"

	sel := Select("media_file.album_artist_id").
		From(r.tableName + " user_events").
		Join("media_file on media_file.id = user_events.entity_id").
		Where(Eq{"user_events.user_id": userID}).
		Where(Eq{"user_events.entity_type": "song"}).
		Where(Eq{"user_events.event_type": []string{"scrobble", "repeat", "skip"}}).
		Where(GtOrEq{"user_events.occurred_at": windowStart}).
		Where(Expr("media_file.album_artist_id != ''")).
		GroupBy("media_file.album_artist_id").
		OrderByClause("sum("+scoreExpr+") desc", nowUnix).
		Limit(uint64(limit))

	var ids []string
	err := r.queryAllSlice(sel, &ids)
	if err == model.ErrNotFound {
		return []string{}, nil
	}
	return ids, err
}
