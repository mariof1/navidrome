package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcasts, downAddPodcasts)
}

func upAddPodcasts(ctx context.Context, tx *sql.Tx) error {
	exec := createExecuteFunc(ctx, tx)

	tasks := []execFunc{
		exec(`CREATE TABLE IF NOT EXISTS podcast_channel (
id varchar(255) not null primary key,
title varchar(255) default '' not null,
rss_url text not null,
site_url text,
description text,
image_url text,
user_id text,
is_global integer not null default 0,
created_at datetime,
updated_at datetime,
last_refreshed_at datetime,
last_error text
);`),
		exec(`CREATE INDEX IF NOT EXISTS idx_podcast_channel_user_global ON podcast_channel(user_id,is_global);`),
		exec(`CREATE TABLE IF NOT EXISTS podcast_episode (
id text primary key,
channel_id text not null,
guid text not null,
title text,
description text,
audio_url text,
mime_type text,
duration integer default 0,
published_at datetime,
image_url text,
created_at datetime default current_timestamp,
updated_at datetime default current_timestamp,
foreign key(channel_id) references podcast_channel(id)
);`),
		exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_podcast_episode_channel_guid ON podcast_episode(channel_id,guid);`),
		exec(`CREATE INDEX IF NOT EXISTS idx_podcast_episode_channel_published ON podcast_episode(channel_id,published_at DESC);`),
	}

	for _, task := range tasks {
		if err := task(); err != nil {
			return err
		}
	}
	return nil
}

func downAddPodcasts(ctx context.Context, tx *sql.Tx) error {
	return nil
}
