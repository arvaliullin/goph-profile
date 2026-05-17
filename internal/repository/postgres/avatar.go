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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// AvatarRepository реализует ports.AvatarRepository.
type AvatarRepository struct {
	pool *pgxpool.Pool
}

var repoTracer = otel.Tracer("postgres-avatar-repository")

// NewAvatarRepository создает репозиторий.
func NewAvatarRepository(pool *pgxpool.Pool) *AvatarRepository {
	return &AvatarRepository{pool: pool}
}

func (r *AvatarRepository) Create(ctx context.Context, a *domain.Avatar) error {
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.create")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", a.ID.String()), attribute.String("user.id", a.UserID))
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
	if err != nil {
		span.RecordError(err)
	}
	return err
}

const selectAvatar = `
SELECT id, user_id, file_name, mime_type, size_bytes, s3_key, thumbnail_s3_keys,
       original_width, original_height,
       upload_status, processing_status, created_at, updated_at, deleted_at
FROM avatars
`

// scanOne читает одну строку avatars в доменную модель.
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
		if err := json.Unmarshal(thumbsRaw, &a.ThumbnailS3Keys); err != nil {
			return nil, fmt.Errorf("thumbnail_s3_keys: %w", err)
		}
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
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.get_by_id")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", id.String()))
	q := selectAvatar + ` WHERE id = $1 AND deleted_at IS NULL`
	a, err := scanOne(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		span.RecordError(err)
	}
	return a, err
}

// GetLatestByUserID возвращает последний не удаленный аватар пользователя.
func (r *AvatarRepository) GetLatestByUserID(ctx context.Context, userID string) (*domain.Avatar, error) {
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.get_latest_by_user")
	defer span.End()
	span.SetAttributes(attribute.String("user.id", userID))
	q := selectAvatar + ` WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`
	a, err := scanOne(r.pool.QueryRow(ctx, q, userID))
	if err != nil {
		span.RecordError(err)
	}
	return a, err
}

// ListByUserID список не удаленных аватаров пользователя, сначала новые.
func (r *AvatarRepository) ListByUserID(ctx context.Context, userID string) ([]domain.Avatar, error) {
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.list_by_user")
	defer span.End()
	span.SetAttributes(attribute.String("user.id", userID))
	q := selectAvatar + ` WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Avatar, 0)
	for rows.Next() {
		a, scanErr := scanOne(rows)
		if scanErr != nil {
			span.RecordError(scanErr)
			return nil, scanErr
		}
		out = append(out, *a)
	}
	err = rows.Err()
	if err != nil {
		span.RecordError(err)
	}
	return out, err
}

// GetByIDIncludingDeleted выборка в том числе удаленных - для идемпотентности avatard.
func (r *AvatarRepository) GetByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.get_by_id_including_deleted")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", id.String()))
	q := selectAvatar + ` WHERE id = $1`
	a, err := scanOne(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		span.RecordError(err)
	}
	return a, err
}

// SoftDelete помечает удаленным при совпадении владельца; true если строка обновлена.
func (r *AvatarRepository) SoftDelete(ctx context.Context, id uuid.UUID, userID string) (bool, error) {
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.soft_delete")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", id.String()), attribute.String("user.id", userID))
	q := `
UPDATE avatars SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
`
	tag, err := r.pool.Exec(ctx, q, id, userID)
	if err != nil {
		span.RecordError(err)
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SoftDeleteLatestByUserID удаляет последний аватар пользователя, если запрос от владельца.
func (r *AvatarRepository) SoftDeleteLatestByUserID(ctx context.Context, userID, requestUserID string) (bool, error) {
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.soft_delete_latest_by_user")
	defer span.End()
	span.SetAttributes(attribute.String("user.id", userID), attribute.String("request.user_id", requestUserID))
	q := `
UPDATE avatars SET deleted_at = NOW(), updated_at = NOW()
WHERE id = (
  SELECT id FROM avatars WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1
) AND user_id = $2 AND deleted_at IS NULL
`
	tag, err := r.pool.Exec(ctx, q, userID, requestUserID)
	if err != nil {
		span.RecordError(err)
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// UpdateProcessingStatus задает processing_status.
func (r *AvatarRepository) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status string) error {
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.update_processing_status")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", id.String()), attribute.String("processing.status", status))
	q := `UPDATE avatars SET processing_status = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, status)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if tag.RowsAffected() == 0 {
		span.RecordError(domain.ErrNotFound)
		return domain.ErrNotFound
	}
	return nil
}

// UpdateThumbnailKeys объединяет JSON ключей превью.
func (r *AvatarRepository) UpdateThumbnailKeys(ctx context.Context, id uuid.UUID, keys map[string]string) error {
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.update_thumbnail_keys")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", id.String()), attribute.Int("thumbnails.count", len(keys)))
	b, err := json.Marshal(keys)
	if err != nil {
		span.RecordError(err)
		return err
	}
	q := `UPDATE avatars SET thumbnail_s3_keys = $2::jsonb, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, b)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if tag.RowsAffected() == 0 {
		span.RecordError(domain.ErrNotFound)
		return domain.ErrNotFound
	}
	return nil
}

// UpdateOriginalDimensions задает ширину и высоту оригинала.
func (r *AvatarRepository) UpdateOriginalDimensions(ctx context.Context, id uuid.UUID, w, h int) error {
	ctx, span := repoTracer.Start(ctx, "postgres.avatar.update_original_dimensions")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", id.String()), attribute.Int("image.width", w), attribute.Int("image.height", h))
	q := `UPDATE avatars SET original_width = $2, original_height = $3, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, w, h)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if tag.RowsAffected() == 0 {
		err = fmt.Errorf("update dimensions: %w", domain.ErrNotFound)
		span.RecordError(err)
		return err
	}
	return nil
}
