package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"image/color"
	"io"
	"log/slog"
	"testing"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type trackRepo struct {
	av     *domain.Avatar
	status string
}

func (t *trackRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	cp := *t.av
	return &cp, nil
}

func (t *trackRepo) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status string) error {
	t.status = status
	t.av.ProcessingStatus = status
	return nil
}

func (t *trackRepo) UpdateThumbnailKeys(ctx context.Context, id uuid.UUID, keys map[string]string) error {
	t.av.ThumbnailS3Keys = keys
	return nil
}

func (t *trackRepo) UpdateOriginalDimensions(ctx context.Context, id uuid.UUID, w, h int) error {
	return nil
}

type mapStore struct {
	data map[string][]byte
}

func (m *mapStore) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.data[key] = b
	return nil
}

func (m *mapStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	b, ok := m.data[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (m *mapStore) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mapStore) DeleteMany(ctx context.Context, keys []string) error {
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func TestHandleUploadFull(t *testing.T) {
	t.Parallel()
	img := imaging.New(32, 32, color.RGBA{A: 255})
	var buf bytes.Buffer
	require.NoError(t, imaging.Encode(&buf, img, imaging.JPEG))
	id := uuid.New()
	key := "avatars/u/" + id.String() + "/original"
	repo := &trackRepo{
		av: &domain.Avatar{
			ID: id, UserID: "u", MimeType: "image/jpeg", S3Key: key,
			ProcessingStatus: domain.ProcessingStatusPending,
		},
	}
	st := &mapStore{data: map[string][]byte{key: buf.Bytes()}}
	p := NewProcessor(repo, st, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ev := ports.AvatarUploadEvent{AvatarID: id.String(), UserID: "u", S3Key: key}
	raw, err := json.Marshal(ev)
	require.NoError(t, err)
	require.NoError(t, p.HandleUpload(context.Background(), raw))
	require.Equal(t, domain.ProcessingStatusCompleted, repo.av.ProcessingStatus)
	require.NotEmpty(t, repo.av.ThumbnailS3Keys)
}
