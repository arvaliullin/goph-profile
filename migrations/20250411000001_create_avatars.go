package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upCreateAvatars, downCreateAvatars)
}

// upCreateAvatars создаёт таблицу avatars и индексы.
func upCreateAvatars(ctx context.Context, tx *sql.Tx) error {
	const q = `
CREATE TABLE IF NOT EXISTS avatars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL,
    s3_key VARCHAR(500) NOT NULL,
    thumbnail_s3_keys JSONB,
    original_width INT,
    original_height INT,
    upload_status VARCHAR(50) NOT NULL DEFAULT 'uploading',
    processing_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_avatars_user_id ON avatars (user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_avatars_status ON avatars (upload_status, processing_status);
`
	_, err := tx.ExecContext(ctx, q)
	return err
}

// downCreateAvatars удаляет таблицу avatars.
func downCreateAvatars(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS avatars`)
	return err
}
