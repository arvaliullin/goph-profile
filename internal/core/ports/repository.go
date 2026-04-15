package ports

import (
	"context"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/google/uuid"
)

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

// AvatarRepository описывает сохранение и выборку метаданных аватаров в БД.
type AvatarRepository interface {
	Create(ctx context.Context, a *domain.Avatar) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error)
	GetLatestByUserID(ctx context.Context, userID string) (*domain.Avatar, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.Avatar, error)
	SoftDelete(ctx context.Context, id uuid.UUID, userID string) (bool, error)
	SoftDeleteLatestByUserID(ctx context.Context, userID, requestUserID string) (bool, error)
	UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateThumbnailKeys(ctx context.Context, id uuid.UUID, keys map[string]string) error
	UpdateOriginalDimensions(ctx context.Context, id uuid.UUID, w, h int) error
}
