package dto

import (
	"fmt"
	"time"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
)

// AvatarUploadResponse ответ POST /api/v1/avatars.
type AvatarUploadResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// ThumbnailItem ссылка на превью в метаданных аватара.
type ThumbnailItem struct {
	Size string `json:"size"`
	URL  string `json:"url"`
}

// AvatarMetadataResponse JSON-метаданные аватара.
type AvatarMetadataResponse struct {
	ID               string          `json:"id"`
	UserID           string          `json:"user_id"`
	FileName         string          `json:"file_name"`
	MimeType         string          `json:"mime_type"`
	Size             int64           `json:"size"`
	Dimensions       map[string]int  `json:"dimensions"`
	Thumbnails       []ThumbnailItem `json:"thumbnails"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
	URL              string          `json:"url"`
	ProcessingStatus string          `json:"processing_status"`
	UploadStatus     string          `json:"upload_status"`
}

// NewAvatarUploadResponse формирует ответ загрузки из доменного аватара и базового URL API.
func NewAvatarUploadResponse(a *domain.Avatar, base string) *AvatarUploadResponse {
	meta := FromDomainAvatar(a, base)
	return &AvatarUploadResponse{
		ID:        a.ID.String(),
		UserID:    a.UserID,
		URL:       meta.URL,
		Status:    "processing",
		CreatedAt: a.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

// FromDomainAvatar строит метаданные для JSON; base - публичный префикс URL без завершающего слэша, пустая строка даёт относительные пути.
func FromDomainAvatar(a *domain.Avatar, base string) *AvatarMetadataResponse {
	path := fmt.Sprintf("/api/v1/avatars/%s", a.ID.String())
	url := path
	if base != "" {
		url = base + path
	}
	thumbs := make([]ThumbnailItem, 0)
	for _, label := range []string{domain.Thumbnail100, domain.Thumbnail300} {
		if _, ok := a.ThumbnailS3Keys[label]; ok {
			u := path + "?size=" + label
			if base != "" {
				u = base + u
			}
			thumbs = append(thumbs, ThumbnailItem{Size: label, URL: u})
		}
	}
	dim := map[string]int{}
	if a.OriginalWidth != nil && a.OriginalHeight != nil {
		dim["width"] = *a.OriginalWidth
		dim["height"] = *a.OriginalHeight
	}
	return &AvatarMetadataResponse{
		ID:               a.ID.String(),
		UserID:           a.UserID,
		FileName:         a.FileName,
		MimeType:         a.MimeType,
		Size:             a.SizeBytes,
		Dimensions:       dim,
		Thumbnails:       thumbs,
		CreatedAt:        a.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        a.UpdatedAt.UTC().Format(time.RFC3339),
		URL:              url,
		ProcessingStatus: a.ProcessingStatus,
		UploadStatus:     a.UploadStatus,
	}
}

// FromDomainAvatars преобразует список доменных аватаров в DTO.
func FromDomainAvatars(list []domain.Avatar, base string) []AvatarMetadataResponse {
	out := make([]AvatarMetadataResponse, len(list))
	for i := range list {
		out[i] = *FromDomainAvatar(&list[i], base)
	}
	return out
}
