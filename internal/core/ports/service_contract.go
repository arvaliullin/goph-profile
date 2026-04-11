package ports

import (
	"context"
	"io"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/google/uuid"
)

//go:generate mockgen -source=service_contract.go -destination=mocks/service_contract_mock.go -package=mocks

// AvatarService описывает сценарии работы с аватарками для HTTP-слоя.
type AvatarService interface {
	Upload(ctx context.Context, userID string, fileName string, contentType string, r io.Reader, size int64) (*domain.Avatar, error)
	GetImage(ctx context.Context, id uuid.UUID, size, format string) (data io.ReadCloser, mime string, etag string, err error)
	GetImageForUser(ctx context.Context, userID string) (data io.ReadCloser, mime string, etag string, err error)
	Metadata(ctx context.Context, id uuid.UUID) (*domain.Avatar, error)
	ListMetadata(ctx context.Context, userID string) ([]domain.Avatar, error)
	Delete(ctx context.Context, id uuid.UUID, userID string) error
	DeleteForUser(ctx context.Context, userID, requestUserID string) error
}
