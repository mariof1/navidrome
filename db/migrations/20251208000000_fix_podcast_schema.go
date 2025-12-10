package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upFixPodcastSchema, downFixPodcastSchema)
}

func upFixPodcastSchema(ctx context.Context, tx *sql.Tx) error {
	exec := createExecuteFunc(ctx, tx)

	tasks := []execFunc{
		exec(`PRAGMA foreign_keys=off;`),
		exec(`DROP TABLE IF EXISTS podcast_channel_tmp;`),
		exec(`CREATE TABLE podcast_channel_tmp (
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
		exec(`INSERT INTO podcast_channel_tmp (id,title,rss_url,site_url,description,image_url,user_id,is_global,created_at,updated_at,last_refreshed_at,last_error)
SELECT CAST(id AS text), title, rss_url, site_url, description, image_url, CAST(user_id AS text), is_global, created_at, updated_at, last_refreshed_at, last_error FROM podcast_channel;`),
		exec(`DROP TABLE IF EXISTS podcast_channel;`),
		exec(`ALTER TABLE podcast_channel_tmp RENAME TO podcast_channel;`),
		exec(`CREATE INDEX IF NOT EXISTS idx_podcast_channel_user_global ON podcast_channel(user_id,is_global);`),
		exec(`DROP TABLE IF EXISTS podcast_episode_tmp;`),
		exec(`CREATE TABLE podcast_episode_tmp (
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
		exec(`INSERT INTO podcast_episode_tmp (id,channel_id,guid,title,description,audio_url,mime_type,duration,published_at,image_url,created_at,updated_at)
SELECT CAST(id AS text), CAST(channel_id AS text), guid, title, description, audio_url, mime_type, duration, published_at, image_url, created_at, updated_at FROM podcast_episode;`),
		exec(`DROP TABLE IF EXISTS podcast_episode;`),
		exec(`ALTER TABLE podcast_episode_tmp RENAME TO podcast_episode;`),
		exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_podcast_episode_channel_guid ON podcast_episode(channel_id,guid);`),
		exec(`CREATE INDEX IF NOT EXISTS idx_podcast_episode_channel_published ON podcast_episode(channel_id,published_at DESC);`),
		exec(`PRAGMA foreign_keys=on;`),
	}

	for _, task := range tasks {
		if err := task(); err != nil {
			return err
		}
	}
	return nil
}

func downFixPodcastSchema(ctx context.Context, tx *sql.Tx) error {
	return nil
}
