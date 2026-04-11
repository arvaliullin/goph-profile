package worker

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type fakeStorage struct {
	lastKeys []string
}

func (f *fakeStorage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	return nil
}

func (f *fakeStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeStorage) Delete(ctx context.Context, key string) error { return nil }

func (f *fakeStorage) DeleteMany(ctx context.Context, keys []string) error {
	f.lastKeys = append([]string(nil), keys...)
	return nil
}

type noopRepo struct{}

func (noopRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	return nil, domain.ErrNotFound
}

func (noopRepo) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}

func (noopRepo) UpdateThumbnailKeys(ctx context.Context, id uuid.UUID, keys map[string]string) error {
	return nil
}

func (noopRepo) UpdateOriginalDimensions(ctx context.Context, id uuid.UUID, w, h int) error {
	return nil
}

func TestHandleDelete(t *testing.T) {
	t.Parallel()
	st := &fakeStorage{}
	p := NewProcessor(noopRepo{}, st, zerolog.Nop())
	ev := ports.AvatarDeleteEvent{AvatarID: "x", S3Keys: []string{"a", "b"}}
	b, err := json.Marshal(ev)
	require.NoError(t, err)
	require.NoError(t, p.HandleDelete(context.Background(), b))
	require.Equal(t, []string{"a", "b"}, st.lastKeys)
}
