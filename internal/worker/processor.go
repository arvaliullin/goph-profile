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
	tracer := otel.Tracer("avatard-worker")
	ctx, span := tracer.Start(ctx, "worker.handle_upload")
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
	fetchCtx, fetchSpan := tracer.Start(ctx, "worker.handle_upload.storage_get")
	rc, err := p.storage.Get(fetchCtx, ev.S3Key)
	fetchSpan.End()
	if err != nil {
		p.markProcessingFailed(ctx, id)
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	defer rc.Close()
	_, readSpan := tracer.Start(ctx, "worker.handle_upload.read_original")
	data, err := io.ReadAll(rc)
	readSpan.End()
	if err != nil {
		p.markProcessingFailed(ctx, id)
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	_, dimSpan := tracer.Start(ctx, "worker.handle_upload.detect_dimensions")
	w, h, err := imageutil.Dimensions(bytes.NewReader(data))
	dimSpan.End()
	if err != nil {
		p.markProcessingFailed(ctx, id)
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	dimWriteCtx, dimWriteSpan := tracer.Start(ctx, "worker.handle_upload.persist_dimensions")
	if err := p.repo.UpdateOriginalDimensions(dimWriteCtx, id, w, h); err != nil {
		dimWriteSpan.End()
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	dimWriteSpan.End()
	keys := map[string]string{}
	for _, label := range []string{domain.Thumbnail100, domain.Thumbnail300} {
		resizeCtx, resizeSpan := tracer.Start(ctx, "worker.handle_upload.resize")
		resizeSpan.SetAttributes(attribute.String("thumbnail.label", label))
		f := imageutil.FormatFromMIME(a.MimeType)
		out, _, err := imageutil.DecodeAndResize(bytes.NewReader(data), label, f)
		if err != nil {
			resizeSpan.End()
			p.markProcessingFailed(ctx, id)
			span.RecordError(err)
			observability.ObserveWorkerJob("avatard", "upload", "error")
			return err
		}
		tk := thumbKey(id, label, a.MimeType)
		if err := p.storage.Put(resizeCtx, tk, bytes.NewReader(out), int64(len(out)), mimeForThumb(a.MimeType)); err != nil {
			resizeSpan.End()
			p.markProcessingFailed(ctx, id)
			span.RecordError(err)
			observability.ObserveWorkerJob("avatard", "upload", "error")
			return err
		}
		resizeSpan.End()
		keys[label] = tk
	}
	thumbCtx, thumbSpan := tracer.Start(ctx, "worker.handle_upload.persist_thumbnails")
	if err := p.repo.UpdateThumbnailKeys(thumbCtx, id, keys); err != nil {
		thumbSpan.End()
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	thumbSpan.End()
	doneCtx, doneSpan := tracer.Start(ctx, "worker.handle_upload.mark_completed")
	if err := p.repo.UpdateProcessingStatus(doneCtx, id, domain.ProcessingStatusCompleted); err != nil {
		doneSpan.End()
		span.RecordError(err)
		observability.ObserveWorkerJob("avatard", "upload", "error")
		return err
	}
	doneSpan.End()
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
