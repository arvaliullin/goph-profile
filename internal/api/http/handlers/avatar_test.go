package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arvaliullin/goph-profile/internal/api/http/middleware"
	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubSvc struct {
	uploadErr error
	meta      map[string]any
}

func (s *stubSvc) Upload(ctx context.Context, userID string, fileName string, contentType string, r io.Reader, size int64) (*domain.Avatar, error) {
	if s.uploadErr != nil {
		return nil, s.uploadErr
	}
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	return &domain.Avatar{
		ID: id, UserID: userID, CreatedAt: time.Unix(0, 0).UTC(),
	}, nil
}

func (s *stubSvc) GetImage(ctx context.Context, id uuid.UUID, size, format string) (io.ReadCloser, string, string, error) {
	return io.NopCloser(bytes.NewReader([]byte("x"))), "image/jpeg", "etag", nil
}

func (s *stubSvc) GetImageForUser(ctx context.Context, userID string) (io.ReadCloser, string, string, error) {
	return io.NopCloser(bytes.NewReader([]byte("x"))), "image/jpeg", "e", nil
}

func (s *stubSvc) Metadata(ctx context.Context, id uuid.UUID, baseURL string) (map[string]any, error) {
	if s.meta != nil {
		return s.meta, nil
	}
	return map[string]any{"id": id.String(), "url": "/api/v1/avatars/" + id.String()}, nil
}

func (s *stubSvc) ListMetadata(ctx context.Context, userID string, baseURL string) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (s *stubSvc) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	return domain.ErrForbidden
}

func (s *stubSvc) DeleteForUser(ctx context.Context, userID, requestUserID string) error {
	return nil
}

func TestAvatarHTTP_GetImage(t *testing.T) {
	t.Parallel()
	h := NewAvatarHTTP(&stubSvc{}, 1024, "")
	r := chi.NewRouter()
	r.Get("/avatars/{avatarID}", h.GetImage)
	req := httptest.NewRequest(http.MethodGet, "/avatars/11111111-1111-1111-1111-111111111111", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
}

func TestAvatarHTTP_UploadMissingUser(t *testing.T) {
	t.Parallel()
	h := NewAvatarHTTP(&stubSvc{}, 1024, "")
	r := chi.NewRouter()
	r.Post("/avatars", h.Upload)
	req := httptest.NewRequest(http.MethodPost, "/avatars", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMiddlewareUserID(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Use(middleware.UserIDFromHeader)
	r.Get("/x", func(w http.ResponseWriter, req *http.Request) {
		id, ok := middleware.UserID(req.Context())
		if !ok {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		if _, err := w.Write([]byte(id)); err != nil {
			return
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-User-ID", "u")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u", rec.Body.String())
}
