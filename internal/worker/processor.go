package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/pkg/imageutil"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AvatarStore контракт доступа к аватарам в PostgreSQL для процессора.
type AvatarStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error)
	UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateThumbnailKeys(ctx context.Context, id uuid.UUID, keys map[string]string) error
	UpdateOriginalDimensions(ctx context.Context, id uuid.UUID, w, h int) error
}

// Processor обрабатывает сообщения Kafka.
type Processor struct {
	repo    AvatarStore
	storage ports.ObjectStorage
	log     zerolog.Logger
}

// NewProcessor создает процессор.
func NewProcessor(repo AvatarStore, storage ports.ObjectStorage, log zerolog.Logger) *Processor {
	return &Processor{repo: repo, storage: storage, log: log}
}

// HandleUpload обрабатывает события avatar.upload.
func (p *Processor) HandleUpload(ctx context.Context, raw []byte) error {
	var ev ports.AvatarUploadEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	id, err := uuid.Parse(ev.AvatarID)
	if err != nil {
		return err
	}
	a, err := p.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	if a.ProcessingStatus == domain.ProcessingStatusCompleted {
		return nil
	}
	if err = p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusProcessing); err != nil {
		return err
	}
	rc, err := p.storage.Get(ctx, ev.S3Key)
	if err != nil {
		_ = p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusFailed)
		return err
	}
	data, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		_ = p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusFailed)
		return err
	}
	w, h, err := imageutil.Dimensions(bytes.NewReader(data))
	if err != nil {
		_ = p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusFailed)
		return err
	}
	if err := p.repo.UpdateOriginalDimensions(ctx, id, w, h); err != nil {
		return err
	}
	keys := map[string]string{}
	for _, label := range []string{domain.Thumbnail100, domain.Thumbnail300} {
		f := imageutil.FormatFromMIME(a.MimeType)
		out, _, err := imageutil.DecodeAndResize(bytes.NewReader(data), label, f)
		if err != nil {
			_ = p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusFailed)
			return err
		}
		tk := thumbKey(id, label, a.MimeType)
		if err := p.storage.Put(ctx, tk, bytes.NewReader(out), int64(len(out)), mimeForThumb(a.MimeType)); err != nil {
			_ = p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusFailed)
			return err
		}
		keys[label] = tk
	}
	if err := p.repo.UpdateThumbnailKeys(ctx, id, keys); err != nil {
		return err
	}
	return p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusCompleted)
}

func thumbKey(id uuid.UUID, label string, origMime string) string {
	ext := ".jpg"
	switch origMime {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	}
	return "avatars/thumbnails/" + id.String() + "/" + label + ext
}

func mimeForThumb(orig string) string {
	switch orig {
	case "image/png":
		return "image/png"
	case "image/webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// HandleDelete удаляет объекты из хранилища.
func (p *Processor) HandleDelete(ctx context.Context, raw []byte) error {
	var ev ports.AvatarDeleteEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	return p.storage.DeleteMany(ctx, ev.S3Keys)
}
