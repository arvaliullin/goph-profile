package worker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/kafka"
	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/arvaliullin/goph-profile/pkg/imageutil"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	log     *slog.Logger
}

// NewProcessor создает процессор.
func NewProcessor(repo AvatarStore, storage ports.ObjectStorage, log *slog.Logger) *Processor {
	return &Processor{repo: repo, storage: storage, log: log}
}

func (p *Processor) markProcessingFailed(ctx context.Context, id uuid.UUID) {
	if err := p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusFailed); err != nil {
		observability.LoggerWithTrace(ctx, p.log).Warn("set processing status to failed", "error", err, "avatar_id", id.String())
	}
}

// HandleUpload обрабатывает события avatar.upload.
func (p *Processor) HandleUpload(ctx context.Context, raw []byte) error {
	ctx, span := otel.Tracer("avatard-worker").Start(ctx, "worker.handle_upload")
	defer span.End()
	ev, err := kafka.UnmarshalUploadEvent(raw)
	if err != nil {
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	span.SetAttributes(attribute.String("avatar.id", ev.AvatarID), attribute.String("avatar.s3_key", ev.S3Key))
	id, err := uuid.Parse(ev.AvatarID)
	if err != nil {
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	a, err := p.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			observability.LoggerWithTrace(ctx, p.log).Info("upload event skipped: avatar not found", "avatar_id", ev.AvatarID)
			observability.ObserveWorkerJob("avatard", "upload", "skipped")
			return nil
		}
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	if a.ProcessingStatus == domain.ProcessingStatusCompleted {
		observability.LoggerWithTrace(ctx, p.log).Info("upload skipped: processing already completed", "avatar_id", id.String())
		observability.ObserveWorkerJob("avatard", "upload", "skipped")
		return nil
	}
	if err = p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusProcessing); err != nil {
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	observability.LoggerWithTrace(ctx, p.log).Info("avatar upload processing started", "avatar_id", id.String(), "s3_key", ev.S3Key)
	rc, err := p.storage.Get(ctx, ev.S3Key)
	if err != nil {
		p.markProcessingFailed(ctx, id)
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		p.markProcessingFailed(ctx, id)
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	w, h, err := imageutil.Dimensions(bytes.NewReader(data))
	if err != nil {
		p.markProcessingFailed(ctx, id)
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	if err := p.repo.UpdateOriginalDimensions(ctx, id, w, h); err != nil {
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	keys := map[string]string{}
	for _, label := range []string{domain.Thumbnail100, domain.Thumbnail300} {
		f := imageutil.FormatFromMIME(a.MimeType)
		out, _, err := imageutil.DecodeAndResize(bytes.NewReader(data), label, f)
		if err != nil {
			p.markProcessingFailed(ctx, id)
			span.RecordError(err)
			observability.ObserveWorkerJob("avatard", "upload", "error")
			return err
		}
		tk := thumbKey(id, label, a.MimeType)
		if err := p.storage.Put(ctx, tk, bytes.NewReader(out), int64(len(out)), mimeForThumb(a.MimeType)); err != nil {
			p.markProcessingFailed(ctx, id)
			span.RecordError(err)
			observability.ObserveWorkerJob("avatard", "upload", "error")
			return err
		}
		keys[label] = tk
	}
	if err := p.repo.UpdateThumbnailKeys(ctx, id, keys); err != nil {
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	if err := p.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusCompleted); err != nil {
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	observability.LoggerWithTrace(ctx, p.log).Info("avatar upload processing completed",
		"avatar_id", id.String(),
		"width", w,
		"height", h,
	)
	observability.ObserveWorkerJob("avatard", "upload", "success")
	return nil
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
	ctx, span := otel.Tracer("avatard-worker").Start(ctx, "worker.handle_delete")
	defer span.End()
	ev, err := kafka.UnmarshalDeleteEvent(raw)
	if err != nil {
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "delete", "error")
		return err
	}
	span.SetAttributes(attribute.String("avatar.id", ev.AvatarID), attribute.Int("s3.keys", len(ev.S3Keys)))
	if err := p.storage.DeleteMany(ctx, ev.S3Keys); err != nil {
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "delete", "error")
		return err
	}
	observability.LoggerWithTrace(ctx, p.log).Info("avatar delete event processed",
		"avatar_id", ev.AvatarID,
		"s3_keys", len(ev.S3Keys),
	)
	observability.ObserveWorkerJob("avatard", "delete", "success")
	return nil
}
