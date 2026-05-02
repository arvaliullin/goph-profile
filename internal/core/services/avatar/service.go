package avatar

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/arvaliullin/goph-profile/pkg/imageutil"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const imageSniffPrefixBytes = 512

var serviceTracer = otel.Tracer("avatar-service")

// Service отвечает за загрузку и выдачу аватаров: сохранение оригинала в объектном хранилище,
// запись метаданных в БД, публикацию событий для постобработки, отдачу оригинала и миниатюр
// в нужном формате и мягкое удаление с уведомлением очереди.
type Service struct {
	repo     ports.AvatarRepository
	storage  ports.ObjectStorage
	pub      ports.EventPublisher
	clock    ports.Clock
	maxBytes int64
}

// New создает сервис аватаров.
func New(repo ports.AvatarRepository, storage ports.ObjectStorage, pub ports.EventPublisher, clock ports.Clock, maxBytes int64) *Service {
	if clock == nil {
		clock = ports.NewRealClock()
	}
	return &Service{repo: repo, storage: storage, pub: pub, clock: clock, maxBytes: maxBytes}
}

func objectKeyOriginal(userID string, id uuid.UUID) string {
	return fmt.Sprintf("avatars/%s/%s/original", userID, id.String())
}

// Upload сохраняет оригинал и ставит задачу обработки в очередь.
func (s *Service) Upload(ctx context.Context, userID string, fileName string, contentType string, r io.Reader, size int64) (*domain.Avatar, error) {
	started := time.Now()
	ctx, span := serviceTracer.Start(ctx, "avatar.upload")
	defer span.End()
	span.SetAttributes(
		attribute.String("user.id", userID),
		attribute.String("file.name", fileName),
		attribute.Int64("file.size", size),
	)
	if userID == "" {
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, domain.ErrMissingUserID
	}
	if size > s.maxBytes {
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, domain.ErrFileTooLarge
	}
	if size <= 0 {
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, domain.ErrMissingFile
	}
	prefix, rest, err := imageutil.ReadSniffPrefix(r, imageSniffPrefixBytes)
	if err != nil {
		span.RecordError(err)
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, err
	}
	mime, err := imageutil.NormalizeContentType(contentType, prefix)
	if err != nil {
		span.RecordError(err)
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, domain.ErrInvalidFormat
	}
	lr := io.LimitReader(rest, s.maxBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		span.RecordError(err)
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, err
	}
	if int64(len(data)) > s.maxBytes {
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, domain.ErrFileTooLarge
	}
	id := uuid.New()
	now := s.clock.Now()
	key := objectKeyOriginal(userID, id)
	a := &domain.Avatar{
		ID:               id,
		UserID:           userID,
		FileName:         filepath.Base(fileName),
		MimeType:         mime,
		SizeBytes:        int64(len(data)),
		S3Key:            key,
		ThumbnailS3Keys:  map[string]string{},
		UploadStatus:     domain.UploadStatusUploading,
		ProcessingStatus: domain.ProcessingStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.storage.Put(ctx, key, bytes.NewReader(data), int64(len(data)), mime); err != nil {
		span.RecordError(err)
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, err
	}
	a.UploadStatus = domain.UploadStatusCompleted
	if err := s.repo.Create(ctx, a); err != nil {
		span.RecordError(err)
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, err
	}
	if err := s.pub.PublishUpload(ctx, ports.AvatarUploadEvent{
		AvatarID: id.String(),
		UserID:   userID,
		S3Key:    key,
	}); err != nil {
		span.RecordError(err)
		observability.ObserveUpload("profiled", "error", userID, time.Since(started), 0)
		return nil, fmt.Errorf("publish upload: %w", err)
	}
	observability.ObserveUpload("profiled", "success", userID, time.Since(started), int64(len(data)))
	return a, nil
}

// GetImage возвращает байты изображения для варианта размера и формата.
func (s *Service) GetImage(ctx context.Context, id uuid.UUID, size, format string) (io.ReadCloser, string, string, error) {
	ctx, span := serviceTracer.Start(ctx, "avatar.get_image")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", id.String()), attribute.String("image.size", size), attribute.String("image.format", format))
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, "", "", err
	}
	size = strings.TrimSpace(size)
	if size == "" || size == domain.SizeOriginal {
		return s.getOriginalEncoded(ctx, a, format)
	}
	if a.ThumbnailS3Keys == nil {
		return nil, "", "", domain.ErrNotFound
	}
	sk, ok := a.ThumbnailS3Keys[size]
	if !ok || sk == "" {
		return nil, "", "", domain.ErrNotFound
	}
	rc, err := s.storage.Get(ctx, sk)
	if err != nil {
		span.RecordError(err)
		return nil, "", "", err
	}
	outMime := thumbMimeForFormat(format, a.MimeType)
	etag := etagForKey(sk)
	if format == "" || format == "original" {
		return rc, outMime, etag, nil
	}
	fmime, err := imageutil.FormatFromQuery(format)
	if err != nil {
		if cerr := rc.Close(); cerr != nil {
			span.RecordError(cerr)
			return nil, "", "", errors.Join(err, cerr)
		}
		span.RecordError(err)
		return nil, "", "", err
	}
	if fmime == outMime {
		return rc, outMime, etag, nil
	}
	data, rerr := io.ReadAll(rc)
	cerr := rc.Close()
	if rerr != nil {
		span.RecordError(rerr)
		return nil, "", "", rerr
	}
	if cerr != nil {
		span.RecordError(cerr)
		return nil, "", "", cerr
	}
	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		span.RecordError(err)
		return nil, "", "", err
	}
	b, err := imageutil.EncodeToBytes(img, fmime)
	if err != nil {
		span.RecordError(err)
		return nil, "", "", err
	}
	etag2 := sha256.Sum256(b)
	return io.NopCloser(bytes.NewReader(b)), fmime, hex.EncodeToString(etag2[:]), nil
}

func thumbMimeForFormat(format string, original string) string {
	if format == "" {
		return original
	}
	m, err := imageutil.FormatFromQuery(format)
	if err != nil {
		return original
	}
	return m
}

func etagForKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (s *Service) getOriginalEncoded(ctx context.Context, a *domain.Avatar, format string) (io.ReadCloser, string, string, error) {
	ctx, span := serviceTracer.Start(ctx, "avatar.get_original_encoded")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", a.ID.String()), attribute.String("avatar.s3_key", a.S3Key), attribute.String("image.format", format))
	rc, err := s.storage.Get(ctx, a.S3Key)
	if err != nil {
		span.RecordError(err)
		return nil, "", "", err
	}
	if format == "" {
		return rc, a.MimeType, etagForKey(a.S3Key), nil
	}
	fmime, err := imageutil.FormatFromQuery(format)
	if err != nil {
		if cerr := rc.Close(); cerr != nil {
			span.RecordError(cerr)
			return nil, "", "", errors.Join(err, cerr)
		}
		span.RecordError(err)
		return nil, "", "", err
	}
	if fmime == a.MimeType {
		return rc, a.MimeType, etagForKey(a.S3Key), nil
	}
	data, rerr := io.ReadAll(rc)
	cerr := rc.Close()
	if rerr != nil {
		span.RecordError(rerr)
		return nil, "", "", rerr
	}
	if cerr != nil {
		span.RecordError(cerr)
		return nil, "", "", cerr
	}
	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		span.RecordError(err)
		return nil, "", "", err
	}
	b, err := imageutil.EncodeToBytes(img, fmime)
	if err != nil {
		span.RecordError(err)
		return nil, "", "", err
	}
	sum := sha256.Sum256(b)
	return io.NopCloser(bytes.NewReader(b)), fmime, hex.EncodeToString(sum[:]), nil
}

// GetImageForUser возвращает последний аватар пользователя.
func (s *Service) GetImageForUser(ctx context.Context, userID string) (io.ReadCloser, string, string, error) {
	ctx, span := serviceTracer.Start(ctx, "avatar.get_image_for_user")
	defer span.End()
	span.SetAttributes(attribute.String("user.id", userID))
	a, err := s.repo.GetLatestByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return nil, "", "", err
	}
	return s.getOriginalEncoded(ctx, a, "")
}

// Metadata возвращает доменный аватар по id для сборки HTTP-метаданных.
func (s *Service) Metadata(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	ctx, span := serviceTracer.Start(ctx, "avatar.metadata")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", id.String()))
	return s.repo.GetByID(ctx, id)
}

// ListMetadata возвращает список аватаров пользователя для сборки HTTP-метаданных.
func (s *Service) ListMetadata(ctx context.Context, userID string) ([]domain.Avatar, error) {
	ctx, span := serviceTracer.Start(ctx, "avatar.list_metadata")
	defer span.End()
	span.SetAttributes(attribute.String("user.id", userID))
	list, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	if len(list) == 0 {
		return []domain.Avatar{}, nil
	}
	return list, nil
}

func (s *Service) deleteOwned(ctx context.Context, a *domain.Avatar, userID string) error {
	ctx, span := serviceTracer.Start(ctx, "avatar.delete_owned")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", a.ID.String()), attribute.String("user.id", userID))
	keys := []string{a.S3Key}
	for _, k := range a.ThumbnailS3Keys {
		if k != "" {
			keys = append(keys, k)
		}
	}
	ok, err := s.repo.SoftDelete(ctx, a.ID, userID)
	if err != nil {
		span.RecordError(err)
		observability.ObserveDelete("profiled", "error", userID)
		return err
	}
	if !ok {
		span.RecordError(domain.ErrNotFound)
		observability.ObserveDelete("profiled", "error", userID)
		return domain.ErrNotFound
	}
	observability.ObserveDeleteStorage("profiled", userID, a.SizeBytes)
	err = s.pub.PublishDelete(ctx, ports.AvatarDeleteEvent{AvatarID: a.ID.String(), S3Keys: keys})
	if err != nil {
		span.RecordError(err)
		observability.ObserveDelete("profiled", "error", userID)
		return err
	}
	observability.ObserveDelete("profiled", "success", userID)
	return err
}

// Delete удаляет аватар при совпадении владельца.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	ctx, span := serviceTracer.Start(ctx, "avatar.delete")
	defer span.End()
	span.SetAttributes(attribute.String("avatar.id", id.String()), attribute.String("user.id", userID))
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if a.UserID != userID {
		span.RecordError(domain.ErrForbidden)
		return domain.ErrForbidden
	}
	return s.deleteOwned(ctx, a, userID)
}

// DeleteForUser удаляет последний аватар пользователя.
func (s *Service) DeleteForUser(ctx context.Context, userID, requestUserID string) error {
	ctx, span := serviceTracer.Start(ctx, "avatar.delete_for_user")
	defer span.End()
	span.SetAttributes(attribute.String("user.id", userID), attribute.String("request.user_id", requestUserID))
	if userID != requestUserID {
		span.RecordError(domain.ErrForbidden)
		return domain.ErrForbidden
	}
	a, err := s.repo.GetLatestByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if a.UserID != requestUserID {
		span.RecordError(domain.ErrForbidden)
		return domain.ErrForbidden
	}
	return s.deleteOwned(ctx, a, requestUserID)
}
