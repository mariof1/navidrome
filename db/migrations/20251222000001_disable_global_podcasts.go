package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upDisableGlobalPodcasts, downDisableGlobalPodcasts)
}

func upDisableGlobalPodcasts(ctx context.Context, tx *sql.Tx) error {
	exec := createExecuteFunc(ctx, tx)

	// Ensure every podcast_channel has an owner. Previously, global channels could be shared.
	// We assign owner to the first admin user when missing, then disable the global flag.
	tasks := []execFunc{
		exec(`UPDATE podcast_channel
SET user_id = COALESCE(NULLIF(user_id, ''), (SELECT id FROM user WHERE is_admin = 1 ORDER BY id LIMIT 1))
WHERE user_id IS NULL OR user_id = '';`),
		exec(`UPDATE podcast_channel SET is_global = 0 WHERE is_global != 0;`),
	}

	for _, task := range tasks {
		if err := task(); err != nil {
			return err
		}
	}
	return nil
}

func downDisableGlobalPodcasts(ctx context.Context, tx *sql.Tx) error {
	return nil
}
