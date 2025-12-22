package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcastEpisodeStatusAndProgress, downAddPodcastEpisodeStatusAndProgress)
}

func upAddPodcastEpisodeStatusAndProgress(ctx context.Context, tx *sql.Tx) error {
	exec := createExecuteFunc(ctx, tx)

	tasks := []execFunc{
		exec(`CREATE TABLE IF NOT EXISTS podcast_episode_status (
user_id text not null,
episode_id text not null,
watched integer not null default 0,
updated_at datetime,
primary key(user_id, episode_id),
foreign key(episode_id) references podcast_episode(id) on delete cascade
);`),
		exec(`CREATE INDEX IF NOT EXISTS idx_podcast_episode_status_user_updated ON podcast_episode_status(user_id, updated_at DESC);`),
		exec(`CREATE TABLE IF NOT EXISTS podcast_episode_progress (
user_id text not null,
episode_id text not null,
position integer not null default 0,
duration integer not null default 0,
updated_at datetime,
primary key(user_id, episode_id),
foreign key(episode_id) references podcast_episode(id) on delete cascade
);`),
		exec(`CREATE INDEX IF NOT EXISTS idx_podcast_episode_progress_user_updated ON podcast_episode_progress(user_id, updated_at DESC);`),
	}

	for _, task := range tasks {
		if err := task(); err != nil {
			return err
		}
	}
	return nil
}

func downAddPodcastEpisodeStatusAndProgress(ctx context.Context, tx *sql.Tx) error {
	return nil
}
