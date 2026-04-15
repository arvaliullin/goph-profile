package ports

import (
	"context"
	"io"
)

//go:generate mockgen -source=storage.go -destination=mocks/storage_mock.go -package=mocks

// ObjectStorage описывает загрузку и выдачу бинарных объектов в S3-совместимом хранилище.
type ObjectStorage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	DeleteMany(ctx context.Context, keys []string) error
}
