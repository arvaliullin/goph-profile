package handlers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arvaliullin/goph-profile/internal/api/http/middleware"
	"github.com/arvaliullin/goph-profile/internal/core/ports/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAvatarHTTP_GetImage(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAvatarService(ctrl)
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	mock.EXPECT().GetImage(gomock.Any(), id, "", "").Return(
		io.NopCloser(bytes.NewReader([]byte("x"))), "image/jpeg", "etag", nil)
	h := NewAvatarHTTP(mock, 1024, "")
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
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAvatarService(ctrl)
	h := NewAvatarHTTP(mock, 1024, "")
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
