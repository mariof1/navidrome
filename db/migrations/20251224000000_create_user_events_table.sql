-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_events(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id VARCHAR(255) NOT NULL
        REFERENCES user(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(255) NOT NULL DEFAULT '',
    query TEXT NOT NULL DEFAULT '',
    player_id VARCHAR(255) NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0,
    occurred_at INTEGER NOT NULL
);

CREATE INDEX user_events_user_time ON user_events (user_id, occurred_at);
CREATE INDEX user_events_user_type ON user_events (user_id, event_type, entity_type);
CREATE INDEX user_events_entity ON user_events (entity_type, entity_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_events;
-- +goose StatementEnd
