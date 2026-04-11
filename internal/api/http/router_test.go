package httpserver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arvaliullin/goph-profile/internal/api/http/handlers"
	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	_ "github.com/arvaliullin/goph-profile/docs"
)

type routerStub struct{}

func (routerStub) Upload(ctx context.Context, userID string, fileName string, contentType string, r io.Reader, size int64) (*domain.Avatar, error) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	return &domain.Avatar{ID: id, UserID: userID, CreatedAt: time.Unix(0, 0).UTC()}, nil
}

func (routerStub) GetImage(ctx context.Context, id uuid.UUID, size, format string) (io.ReadCloser, string, string, error) {
	return io.NopCloser(bytes.NewReader([]byte("x"))), "image/jpeg", "e", nil
}

func (routerStub) GetImageForUser(ctx context.Context, userID string) (io.ReadCloser, string, string, error) {
	return io.NopCloser(bytes.NewReader([]byte("x"))), "image/jpeg", "e", nil
}

func (routerStub) Metadata(ctx context.Context, id uuid.UUID, baseURL string) (map[string]any, error) {
	return map[string]any{"id": id.String()}, nil
}

func (routerStub) ListMetadata(ctx context.Context, userID string, baseURL string) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (routerStub) Delete(ctx context.Context, id uuid.UUID, userID string) error { return nil }

func (routerStub) DeleteForUser(ctx context.Context, userID, requestUserID string) error { return nil }

func TestNewRouterHealth(t *testing.T) {
	t.Parallel()
	log := zerolog.Nop()
	h := &handlers.Health{}
	av := handlers.NewAvatarHTTP(routerStub{}, 1024, "")
	r := NewRouter(Deps{Log: log, Health: h, Avatar: av})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestNewRouterSwaggerUI(t *testing.T) {
	t.Parallel()
	log := zerolog.Nop()
	h := &handlers.Health{}
	av := handlers.NewAvatarHTTP(routerStub{}, 1024, "")
	r := NewRouter(Deps{Log: log, Health: h, Avatar: av})
	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
