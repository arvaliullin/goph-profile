package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AvatarRepository реализует ports.AvatarRepository.
type AvatarRepository struct {
	pool *pgxpool.Pool
}

// NewAvatarRepository создает репозиторий.
func NewAvatarRepository(pool *pgxpool.Pool) *AvatarRepository {
	return &AvatarRepository{pool: pool}
}

func (r *AvatarRepository) Create(ctx context.Context, a *domain.Avatar) error {
	var thumbs []byte
	var err error
	if len(a.ThumbnailS3Keys) > 0 {
		thumbs, err = json.Marshal(a.ThumbnailS3Keys)
		if err != nil {
			return err
		}
	}
	q := `
INSERT INTO avatars (id, user_id, file_name, mime_type, size_bytes, s3_key, thumbnail_s3_keys, upload_status, processing_status, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11)
`
	_, err = r.pool.Exec(ctx, q,
		a.ID, a.UserID, a.FileName, a.MimeType, a.SizeBytes, a.S3Key, thumbs,
		a.UploadStatus, a.ProcessingStatus, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

const selectAvatar = `
SELECT id, user_id, file_name, mime_type, size_bytes, s3_key, thumbnail_s3_keys,
       original_width, original_height,
       upload_status, processing_status, created_at, updated_at, deleted_at
FROM avatars
`

func scanOne(row pgx.Row) (*domain.Avatar, error) {
	var (
		a         domain.Avatar
		thumbsRaw []byte
		origW     *int32
		origH     *int32
		deletedAt *time.Time
	)
	err := row.Scan(
		&a.ID, &a.UserID, &a.FileName, &a.MimeType, &a.SizeBytes, &a.S3Key,
		&thumbsRaw, &origW, &origH,
		&a.UploadStatus, &a.ProcessingStatus, &a.CreatedAt, &a.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if len(thumbsRaw) > 0 {
		_ = json.Unmarshal(thumbsRaw, &a.ThumbnailS3Keys)
	}
	if a.ThumbnailS3Keys == nil {
		a.ThumbnailS3Keys = map[string]string{}
	}
	if origW != nil {
		v := int(*origW)
		a.OriginalWidth = &v
	}
	if origH != nil {
		v := int(*origH)
		a.OriginalHeight = &v
	}
	a.DeletedAt = deletedAt
	return &a, nil
}

// GetByID возвращает активный (не удаленный) аватар.
func (r *AvatarRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	q := selectAvatar + ` WHERE id = $1 AND deleted_at IS NULL`
	return scanOne(r.pool.QueryRow(ctx, q, id))
}

// GetLatestByUserID возвращает последний не удаленный аватар пользователя.
func (r *AvatarRepository) GetLatestByUserID(ctx context.Context, userID string) (*domain.Avatar, error) {
	q := selectAvatar + ` WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`
	return scanOne(r.pool.QueryRow(ctx, q, userID))
}

// ListByUserID список не удаленных аватаров пользователя, сначала новые.
func (r *AvatarRepository) ListByUserID(ctx context.Context, userID string) ([]domain.Avatar, error) {
	q := selectAvatar + ` WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Avatar, 0)
	for rows.Next() {
		a, err := scanOne(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// GetByIDIncludingDeleted выборка в том числе удаленных - для идемпотентности avatard.
func (r *AvatarRepository) GetByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	q := selectAvatar + ` WHERE id = $1`
	return scanOne(r.pool.QueryRow(ctx, q, id))
}

// SoftDelete помечает удаленным при совпадении владельца; true если строка обновлена.
func (r *AvatarRepository) SoftDelete(ctx context.Context, id uuid.UUID, userID string) (bool, error) {
	q := `
UPDATE avatars SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
`
	tag, err := r.pool.Exec(ctx, q, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SoftDeleteLatestByUserID удаляет последний аватар пользователя, если запрос от владельца.
func (r *AvatarRepository) SoftDeleteLatestByUserID(ctx context.Context, userID, requestUserID string) (bool, error) {
	q := `
UPDATE avatars SET deleted_at = NOW(), updated_at = NOW()
WHERE id = (
  SELECT id FROM avatars WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1
) AND user_id = $2 AND deleted_at IS NULL
`
	tag, err := r.pool.Exec(ctx, q, userID, requestUserID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// UpdateProcessingStatus задает processing_status.
func (r *AvatarRepository) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status string) error {
	q := `UPDATE avatars SET processing_status = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdateThumbnailKeys объединяет JSON ключей превью.
func (r *AvatarRepository) UpdateThumbnailKeys(ctx context.Context, id uuid.UUID, keys map[string]string) error {
	b, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	q := `UPDATE avatars SET thumbnail_s3_keys = $2::jsonb, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, b)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdateOriginalDimensions задает ширину и высоту оригинала.
func (r *AvatarRepository) UpdateOriginalDimensions(ctx context.Context, id uuid.UUID, w, h int) error {
	q := `UPDATE avatars SET original_width = $2, original_height = $3, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, w, h)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update dimensions: %w", domain.ErrNotFound)
	}
	return nil
}
