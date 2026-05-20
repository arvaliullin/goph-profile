package breaker

import (
	"context"
	"io"

	"github.com/arvaliullin/goph-profile/internal/core/ports"
)

// Storage оборачивает ObjectStorage circuit breaker.
type Storage struct {
	inner ports.ObjectStorage
	br    *Breaker
}

// WrapStorage возвращает ObjectStorage с circuit breaker.
func WrapStorage(inner ports.ObjectStorage, br *Breaker) ports.ObjectStorage {
	if br == nil {
		return inner
	}
	return &Storage{inner: inner, br: br}
}

// Put загружает объект.
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.br.Execute(ctx, func() (any, error) {
		return nil, s.inner.Put(ctx, key, r, size, contentType)
	})
	return err
}

// Get скачивает объект.
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	v, err := s.br.Execute(ctx, func() (any, error) {
		return s.inner.Get(ctx, key)
	})
	if err != nil {
		return nil, err
	}
	return v.(io.ReadCloser), nil
}

// Delete удаляет объект.
func (s *Storage) Delete(ctx context.Context, key string) error {
	_, err := s.br.Execute(ctx, func() (any, error) {
		return nil, s.inner.Delete(ctx, key)
	})
	return err
}

// DeleteMany удаляет несколько объектов.
func (s *Storage) DeleteMany(ctx context.Context, keys []string) error {
	_, err := s.br.Execute(ctx, func() (any, error) {
		return nil, s.inner.DeleteMany(ctx, keys)
	})
	return err
}
