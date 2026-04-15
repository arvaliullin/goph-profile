package handlers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arvaliullin/goph-profile/internal/api/http/middleware"
	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAvatarService(ctrl)
	mock.EXPECT().Upload(gomock.Any(), "u1", "a.jpg", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, domain.ErrInvalidFormat)
	h := NewAvatarHTTP(mock, 1024, "")
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
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAvatarService(ctrl)
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	av := &domain.Avatar{ID: id, UserID: "u1", CreatedAt: time.Unix(0, 0).UTC()}
	mock.EXPECT().Upload(gomock.Any(), "u1", "a.jpg", gomock.Any(), gomock.Any(), gomock.Any()).Return(av, nil)
	h := NewAvatarHTTP(mock, 1024, "")
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
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAvatarService(ctrl)
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	mock.EXPECT().Metadata(gomock.Any(), id).Return(&domain.Avatar{
		ID: id, UserID: "u", FileName: "a.jpg", MimeType: "image/jpeg",
		ThumbnailS3Keys: map[string]string{},
		UploadStatus:    domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusPending,
		CreatedAt: time.Unix(0, 0).UTC(), UpdatedAt: time.Unix(0, 0).UTC(),
	}, nil)
	h := NewAvatarHTTP(mock, 1024, "")
	r := chi.NewRouter()
	r.Get("/avatars/{avatarID}/metadata", h.Metadata)
	req := httptest.NewRequest(http.MethodGet, "/avatars/11111111-1111-1111-1111-111111111111/metadata", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestAvatarHTTP_Delete403(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAvatarService(ctrl)
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	mock.EXPECT().Delete(gomock.Any(), id, "u").Return(domain.ErrForbidden)
	h := NewAvatarHTTP(mock, 1024, "")
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
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAvatarService(ctrl)
	gomock.InOrder(
		mock.EXPECT().GetImageForUser(gomock.Any(), "u1").Return(
			io.NopCloser(bytes.NewReader([]byte("x"))), "image/jpeg", "e", nil),
		mock.EXPECT().ListMetadata(gomock.Any(), "u1").Return([]domain.Avatar{}, nil),
	)
	h := NewAvatarHTTP(mock, 1024, "")
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
