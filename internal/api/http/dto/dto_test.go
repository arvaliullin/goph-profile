package dto

import (
	"testing"
	"time"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromDomainAvatar_emptyBase(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	a := &domain.Avatar{
		ID:               id,
		UserID:           "u1",
		FileName:         "a.jpg",
		MimeType:         "image/jpeg",
		SizeBytes:        42,
		ThumbnailS3Keys:  map[string]string{},
		UploadStatus:     domain.UploadStatusCompleted,
		ProcessingStatus: domain.ProcessingStatusPending,
		CreatedAt:        time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:        time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
	}
	got := FromDomainAvatar(a, "")
	assert.Equal(t, id.String(), got.ID)
	assert.Equal(t, "/api/v1/avatars/"+id.String(), got.URL)
	assert.Empty(t, got.Dimensions)
	assert.Empty(t, got.Thumbnails)
}

func TestFromDomainAvatar_withBaseAndThumbs(t *testing.T) {
	id := uuid.New()
	w, h := 100, 200
	a := &domain.Avatar{
		ID:        id,
		UserID:    "u",
		FileName:  "x.png",
		MimeType:  "image/png",
		SizeBytes: 10,
		ThumbnailS3Keys: map[string]string{
			domain.Thumbnail100: "k100",
			domain.Thumbnail300: "k300",
		},
		OriginalWidth:    &w,
		OriginalHeight:   &h,
		UploadStatus:     domain.UploadStatusCompleted,
		ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt:        time.Unix(1, 0).UTC(),
		UpdatedAt:        time.Unix(2, 0).UTC(),
	}
	got := FromDomainAvatar(a, "https://api.example.com")
	require.Len(t, got.Thumbnails, 2)
	assert.Equal(t, "https://api.example.com/api/v1/avatars/"+id.String(), got.URL)
	assert.Equal(t, 100, got.Dimensions["width"])
	assert.Equal(t, 200, got.Dimensions["height"])
	assert.Equal(t, domain.Thumbnail100, got.Thumbnails[0].Size)
}

func TestFromDomainAvatars_empty(t *testing.T) {
	assert.Empty(t, FromDomainAvatars(nil, ""))
	assert.Empty(t, FromDomainAvatars([]domain.Avatar{}, ""))
}

func TestNewAvatarUploadResponse(t *testing.T) {
	id := uuid.New()
	a := &domain.Avatar{
		ID:        id,
		UserID:    "u",
		CreatedAt: time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC),
	}
	got := NewAvatarUploadResponse(a, "https://x.com")
	assert.Equal(t, id.String(), got.ID)
	assert.Equal(t, "u", got.UserID)
	assert.Equal(t, "processing", got.Status)
	assert.Equal(t, "https://x.com/api/v1/avatars/"+id.String(), got.URL)
	assert.Equal(t, "2024-05-06T07:08:09Z", got.CreatedAt)
}
