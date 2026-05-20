package breaker

import (
	"context"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/google/uuid"
)

// Repository оборачивает AvatarRepository circuit breaker.
type Repository struct {
	inner ports.AvatarRepository
	br    *Breaker
}

// WrapRepository возвращает AvatarRepository с circuit breaker.
func WrapRepository(inner ports.AvatarRepository, br *Breaker) ports.AvatarRepository {
	if br == nil {
		return inner
	}
	return &Repository{inner: inner, br: br}
}

// Create сохраняет аватар.
func (r *Repository) Create(ctx context.Context, a *domain.Avatar) error {
	_, err := r.br.Execute(ctx, func() (any, error) {
		return nil, r.inner.Create(ctx, a)
	})
	return err
}

// GetByID возвращает аватар по id.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	v, err := r.br.Execute(ctx, func() (any, error) {
		return r.inner.GetByID(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return v.(*domain.Avatar), nil
}

// GetLatestByUserID возвращает последний аватар пользователя.
func (r *Repository) GetLatestByUserID(ctx context.Context, userID string) (*domain.Avatar, error) {
	v, err := r.br.Execute(ctx, func() (any, error) {
		return r.inner.GetLatestByUserID(ctx, userID)
	})
	if err != nil {
		return nil, err
	}
	return v.(*domain.Avatar), nil
}

// ListByUserID возвращает список аватаров пользователя.
func (r *Repository) ListByUserID(ctx context.Context, userID string) ([]domain.Avatar, error) {
	v, err := r.br.Execute(ctx, func() (any, error) {
		return r.inner.ListByUserID(ctx, userID)
	})
	if err != nil {
		return nil, err
	}
	return v.([]domain.Avatar), nil
}

// SoftDelete помечает аватар удалённым.
func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID, userID string) (bool, error) {
	v, err := r.br.Execute(ctx, func() (any, error) {
		return r.inner.SoftDelete(ctx, id, userID)
	})
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}

// SoftDeleteLatestByUserID удаляет последний аватар пользователя.
func (r *Repository) SoftDeleteLatestByUserID(ctx context.Context, userID, requestUserID string) (bool, error) {
	v, err := r.br.Execute(ctx, func() (any, error) {
		return r.inner.SoftDeleteLatestByUserID(ctx, userID, requestUserID)
	})
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}

// UpdateProcessingStatus обновляет статус обработки.
func (r *Repository) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.br.Execute(ctx, func() (any, error) {
		return nil, r.inner.UpdateProcessingStatus(ctx, id, status)
	})
	return err
}

// UpdateThumbnailKeys обновляет ключи превью.
func (r *Repository) UpdateThumbnailKeys(ctx context.Context, id uuid.UUID, keys map[string]string) error {
	_, err := r.br.Execute(ctx, func() (any, error) {
		return nil, r.inner.UpdateThumbnailKeys(ctx, id, keys)
	})
	return err
}

// UpdateOriginalDimensions сохраняет размеры оригинала.
func (r *Repository) UpdateOriginalDimensions(ctx context.Context, id uuid.UUID, w, h int) error {
	_, err := r.br.Execute(ctx, func() (any, error) {
		return nil, r.inner.UpdateOriginalDimensions(ctx, id, w, h)
	})
	return err
}
