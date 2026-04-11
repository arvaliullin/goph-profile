package domain

import (
	"time"

	"github.com/google/uuid"
)

// UploadStatus значения статуса загрузки аватара.
const (
	UploadStatusUploading = "uploading"
	UploadStatusCompleted = "completed"
)

// ProcessingStatus значения статуса асинхронной обработки изображения.
const (
	ProcessingStatusPending    = "pending"
	ProcessingStatusProcessing = "processing"
	ProcessingStatusCompleted  = "completed"
	ProcessingStatusFailed     = "failed"
)

// ThumbnailSize метки размеров в JSON и в query-параметрах.
const (
	Thumbnail100 = "100x100"
	Thumbnail300 = "300x300"
	SizeOriginal = "original"
)

// Avatar метаданные аватара в хранилище.
type Avatar struct {
	ID               uuid.UUID
	UserID           string
	FileName         string
	MimeType         string
	SizeBytes        int64
	S3Key            string
	OriginalWidth    *int
	OriginalHeight   *int
	ThumbnailS3Keys  map[string]string
	UploadStatus     string
	ProcessingStatus string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

// Dimensions размеры оригинала после декодирования (для метаданных).
type Dimensions struct {
	Width  int
	Height int
}
