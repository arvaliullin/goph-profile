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

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/pkg/imageutil"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

const imageSniffPrefixBytes = 512

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
	if userID == "" {
		return nil, domain.ErrMissingUserID
	}
	if size > s.maxBytes {
		return nil, domain.ErrFileTooLarge
	}
	if size <= 0 {
		return nil, domain.ErrMissingFile
	}
	prefix, rest, err := imageutil.ReadSniffPrefix(r, imageSniffPrefixBytes)
	if err != nil {
		return nil, err
	}
	mime, err := imageutil.NormalizeContentType(contentType, prefix)
	if err != nil {
		return nil, domain.ErrInvalidFormat
	}
	lr := io.LimitReader(rest, s.maxBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > s.maxBytes {
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
		return nil, err
	}
	a.UploadStatus = domain.UploadStatusCompleted
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	if err := s.pub.PublishUpload(ctx, ports.AvatarUploadEvent{
		AvatarID: id.String(),
		UserID:   userID,
		S3Key:    key,
	}); err != nil {
		return nil, fmt.Errorf("publish upload: %w", err)
	}
	return a, nil
}

// GetImage возвращает байты изображения для варианта размера и формата.
func (s *Service) GetImage(ctx context.Context, id uuid.UUID, size, format string) (io.ReadCloser, string, string, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
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
			return nil, "", "", errors.Join(err, cerr)
		}
		return nil, "", "", err
	}
	if fmime == outMime {
		return rc, outMime, etag, nil
	}
	data, rerr := io.ReadAll(rc)
	cerr := rc.Close()
	if rerr != nil {
		return nil, "", "", rerr
	}
	if cerr != nil {
		return nil, "", "", cerr
	}
	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", "", err
	}
	b, err := imageutil.EncodeToBytes(img, fmime)
	if err != nil {
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
	rc, err := s.storage.Get(ctx, a.S3Key)
	if err != nil {
		return nil, "", "", err
	}
	if format == "" {
		return rc, a.MimeType, etagForKey(a.S3Key), nil
	}
	fmime, err := imageutil.FormatFromQuery(format)
	if err != nil {
		if cerr := rc.Close(); cerr != nil {
			return nil, "", "", errors.Join(err, cerr)
		}
		return nil, "", "", err
	}
	if fmime == a.MimeType {
		return rc, a.MimeType, etagForKey(a.S3Key), nil
	}
	data, rerr := io.ReadAll(rc)
	cerr := rc.Close()
	if rerr != nil {
		return nil, "", "", rerr
	}
	if cerr != nil {
		return nil, "", "", cerr
	}
	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", "", err
	}
	b, err := imageutil.EncodeToBytes(img, fmime)
	if err != nil {
		return nil, "", "", err
	}
	sum := sha256.Sum256(b)
	return io.NopCloser(bytes.NewReader(b)), fmime, hex.EncodeToString(sum[:]), nil
}

// GetImageForUser возвращает последний аватар пользователя.
func (s *Service) GetImageForUser(ctx context.Context, userID string) (io.ReadCloser, string, string, error) {
	a, err := s.repo.GetLatestByUserID(ctx, userID)
	if err != nil {
		return nil, "", "", err
	}
	return s.getOriginalEncoded(ctx, a, "")
}

// Metadata возвращает доменный аватар по id для сборки HTTP-метаданных.
func (s *Service) Metadata(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	return s.repo.GetByID(ctx, id)
}

// ListMetadata возвращает список аватаров пользователя для сборки HTTP-метаданных.
func (s *Service) ListMetadata(ctx context.Context, userID string) ([]domain.Avatar, error) {
	list, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []domain.Avatar{}, nil
	}
	return list, nil
}

// Delete удаляет аватар при совпадении владельца.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a.UserID != userID {
		return domain.ErrForbidden
	}
	keys := []string{a.S3Key}
	for _, k := range a.ThumbnailS3Keys {
		if k != "" {
			keys = append(keys, k)
		}
	}
	ok, err := s.repo.SoftDelete(ctx, id, userID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotFound
	}
	return s.pub.PublishDelete(ctx, ports.AvatarDeleteEvent{AvatarID: id.String(), S3Keys: keys})
}

// DeleteForUser удаляет последний аватар пользователя.
func (s *Service) DeleteForUser(ctx context.Context, userID, requestUserID string) error {
	if userID != requestUserID {
		return domain.ErrForbidden
	}
	a, err := s.repo.GetLatestByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.Delete(ctx, a.ID, requestUserID)
}
