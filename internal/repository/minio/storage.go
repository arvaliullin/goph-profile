package minio

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	ctx, span := otel.Tracer("minio-storage").Start(ctx, "s3.put")
	defer span.End()
	span.SetAttributes(
		attribute.String("s3.bucket", s.bucket),
		attribute.String("s3.key", key),
		attribute.Int64("s3.size", size),
	)
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Get скачивает объект.
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	ctx, span := otel.Tracer("minio-storage").Start(ctx, "s3.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("s3.bucket", s.bucket),
		attribute.String("s3.key", key),
	)
	o, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return o, nil
}

// Delete удаляет один объект.
func (s *Storage) Delete(ctx context.Context, key string) error {
	ctx, span := otel.Tracer("minio-storage").Start(ctx, "s3.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("s3.bucket", s.bucket),
		attribute.String("s3.key", key),
	)
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// isNoSuchKeyOrBucket возвращает true для отсутствующего ключа или бакета.
func isNoSuchKeyOrBucket(err error) bool {
	if err == nil {
		return false
	}
	code := minio.ToErrorResponse(err).Code
	return code == "NoSuchKey" || code == "NoSuchBucket"
}

// DeleteMany удаляет объекты по возможности; отсутствующие ключи не считаются ошибкой.
func (s *Storage) DeleteMany(ctx context.Context, keys []string) error {
	ctx, span := otel.Tracer("minio-storage").Start(ctx, "s3.delete_many")
	defer span.End()
	span.SetAttributes(
		attribute.String("s3.bucket", s.bucket),
		attribute.Int("s3.keys_count", len(keys)),
	)
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
	if first != nil {
		span.RecordError(first)
	}
	return first
}

// Ping проверяет доступ к бакету.
func (s *Storage) Ping(ctx context.Context) error {
	ctx, span := otel.Tracer("minio-storage").Start(ctx, "s3.ping")
	defer span.End()
	span.SetAttributes(attribute.String("s3.bucket", s.bucket))
	_, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
