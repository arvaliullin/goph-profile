package minio

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Storage реализует ports.ObjectStorage для MinIO/S3.
type Storage struct {
	client *minio.Client
	bucket string
}

// New создает бакет при отсутствии и возвращает хранилище.
func New(ctx context.Context, endpoint, accessKey, secretKey string, secure bool, bucket string) (*Storage, error) {
	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	exists, err := cl.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := cl.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			eresp := minio.ToErrorResponse(err)
			if eresp.Code != "BucketAlreadyOwnedByYou" && eresp.Code != "BucketAlreadyExists" {
				return nil, err
			}
		}
	}
	return &Storage{client: cl, bucket: bucket}, nil
}

// Put загружает объект.
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

// Get скачивает объект.
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	o, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return o, nil
}

// Delete удаляет один объект.
func (s *Storage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func isNoSuchKeyOrBucket(err error) bool {
	if err == nil {
		return false
	}
	code := minio.ToErrorResponse(err).Code
	return code == "NoSuchKey" || code == "NoSuchBucket"
}

// DeleteMany удаляет объекты по возможности (best-effort).
// Повторное удаление уже отсутствующих ключей не считается ошибкой (идемпотентность для worker).
func (s *Storage) DeleteMany(ctx context.Context, keys []string) error {
	var first error
	for _, k := range keys {
		if k == "" {
			continue
		}
		if err := s.Delete(ctx, k); err != nil {
			if isNoSuchKeyOrBucket(err) {
				continue
			}
			if first == nil {
				first = fmt.Errorf("delete %s: %w", k, err)
			}
		}
	}
	return first
}

// Ping проверяет доступ к бакету.
func (s *Storage) Ping(ctx context.Context) error {
	_, err := s.client.BucketExists(ctx, s.bucket)
	return err
}
