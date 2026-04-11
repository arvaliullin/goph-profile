package handlers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arvaliullin/goph-profile/internal/api/http/middleware"
	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type errSvc struct{ e error }

func (e errSvc) Upload(ctx context.Context, userID string, fileName string, contentType string, r io.Reader, size int64) (*domain.Avatar, error) {
	return nil, e.e
}
func (errSvc) GetImage(ctx context.Context, id uuid.UUID, size, format string) (io.ReadCloser, string, string, error) {
	return nil, "", "", domain.ErrNotFound
}
func (errSvc) GetImageForUser(ctx context.Context, userID string) (io.ReadCloser, string, string, error) {
	return nil, "", "", domain.ErrNotFound
}
func (errSvc) Metadata(ctx context.Context, id uuid.UUID, baseURL string) (map[string]any, error) {
	return nil, domain.ErrNotFound
}
func (errSvc) ListMetadata(ctx context.Context, userID string, baseURL string) ([]map[string]any, error) {
	return nil, domain.ErrNotFound
}
func (errSvc) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	return domain.ErrForbidden
}
func (errSvc) DeleteForUser(ctx context.Context, userID, requestUserID string) error {
	return domain.ErrForbidden
}

func multipartUpload(t *testing.T, user string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "a.jpg")
	require.NoError(t, err)
	_, err = fw.Write([]byte{0xff, 0xd8, 0xff})
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/avatars", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if user != "" {
		req.Header.Set("X-User-ID", user)
	}
	return req
}

func TestAvatarHTTP_UploadInvalidFormat(t *testing.T) {
	t.Parallel()
	h := NewAvatarHTTP(errSvc{e: domain.ErrInvalidFormat}, 1024, "")
	r := chi.NewRouter()
	r.Use(middleware.UserIDFromHeader)
	r.Post("/avatars", h.Upload)
	req := multipartUpload(t, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAvatarHTTP_UploadMultipart(t *testing.T) {
	t.Parallel()
	h := NewAvatarHTTP(&stubSvc{}, 1024, "")
	r := chi.NewRouter()
	r.Use(middleware.UserIDFromHeader)
	r.Post("/avatars", h.Upload)
	req := multipartUpload(t, "u1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestAvatarHTTP_Metadata(t *testing.T) {
	t.Parallel()
	h := NewAvatarHTTP(&stubSvc{}, 1024, "")
	r := chi.NewRouter()
	r.Get("/avatars/{avatarID}/metadata", h.Metadata)
	req := httptest.NewRequest(http.MethodGet, "/avatars/11111111-1111-1111-1111-111111111111/metadata", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestAvatarHTTP_Delete403(t *testing.T) {
	t.Parallel()
	h := NewAvatarHTTP(&stubSvc{}, 1024, "")
	r := chi.NewRouter()
	r.Use(middleware.UserIDFromHeader)
	r.Delete("/avatars/{avatarID}", h.DeleteAvatar)
	req := httptest.NewRequest(http.MethodDelete, "/avatars/11111111-1111-1111-1111-111111111111", nil)
	req.Header.Set("X-User-ID", "u")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAvatarHTTP_UserRoutes(t *testing.T) {
	t.Parallel()
	h := NewAvatarHTTP(&stubSvc{}, 1024, "")
	r := chi.NewRouter()
	r.Get("/users/{userID}/avatar", h.UserAvatar)
	r.Get("/users/{userID}/avatars", h.UserAvatars)
	req := httptest.NewRequest(http.MethodGet, "/users/u1/avatar", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/users/u1/avatars", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)
}
