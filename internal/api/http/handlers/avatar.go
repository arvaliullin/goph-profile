package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/arvaliullin/goph-profile/internal/api/http/middleware"
	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/pkg/imageutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// multipartOverheadBytes запас байт сверх лимита тела запроса под служебные данные multipart.
const multipartOverheadBytes = 1024

// AvatarHTTP REST-обработчики аватаров.
type AvatarHTTP struct {
	svc       ports.AvatarService
	maxBytes  int64
	publicURL string
}

// NewAvatarHTTP создает набор обработчиков.
func NewAvatarHTTP(svc ports.AvatarService, maxBytes int64, publicURL string) *AvatarHTTP {
	return &AvatarHTTP{svc: svc, maxBytes: maxBytes, publicURL: strings.TrimRight(publicURL, "/")}
}

// avatarUploadResponse ответ POST /api/v1/avatars.
type avatarUploadResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// Upload загружает файл аватара.
// @Summary Загрузить аватар
// @Description Принимает multipart/form-data с полем file. Требуется заголовок X-User-ID.
// @Tags avatars
// @Accept multipart/form-data
// @Produce json
// @Param X-User-ID header string true "Идентификатор пользователя"
// @Param file formData file true "Изображение"
// @Success 201 {object} avatarUploadResponse
// @Failure 400 {object} map[string]string "Неверный запрос или отсутствует X-User-ID"
// @Failure 413 {object} map[string]interface{} "Файл слишком большой"
// @Failure 500 {object} map[string]string "Внутренняя ошибка"
// @Security UserIDAuth
// @Router /api/v1/avatars [post]
func (h *AvatarHTTP) Upload(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok || uid == "" {
		writeError(w, http.StatusBadRequest, map[string]string{"error": "X-User-ID required"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes+multipartOverheadBytes)
	if err := r.ParseMultipartForm(h.maxBytes + multipartOverheadBytes); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, imageutil.ErrFileTooLarge(h.maxBytes))
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, map[string]string{"error": "file required"})
		return
	}
	defer f.Close()
	a, err := h.svc.Upload(r.Context(), uid, hdr.Filename, hdr.Header.Get("Content-Type"), f, hdr.Size)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	meta, err := h.svc.Metadata(r.Context(), a.ID, h.publicURL)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	var url string
	if u, ok := meta["url"].(string); ok {
		url = u
	}
	writeJSON(w, http.StatusCreated, avatarUploadResponse{
		ID:        a.ID.String(),
		UserID:    a.UserID,
		URL:       url,
		Status:    "processing",
		CreatedAt: a.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *AvatarHTTP) mapErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidFormat):
		writeError(w, http.StatusBadRequest, imageutil.ErrInvalidFormatDetail())
	case errors.Is(err, domain.ErrFileTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, imageutil.ErrFileTooLarge(h.maxBytes))
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, map[string]string{"error": "Avatar not found"})
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, map[string]any{"error": "Forbidden", "details": "You can only delete your own avatars"})
	default:
		writeError(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

// GetImage отдаёт бинарное изображение аватара.
// @Summary Получить изображение аватара
// @Tags avatars
// @Produce octet-stream
// @Param avatarID path string true "UUID аватара"
// @Param size query string false "Размер превью"
// @Param format query string false "Формат выдачи (например webp)"
// @Success 200 {file} binary "Тело изображения"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/avatars/{avatarID} [get]
func (h *AvatarHTTP) GetImage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "avatarID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	size := r.URL.Query().Get("size")
	format := r.URL.Query().Get("format")
	rc, mime, etag, err := h.svc.GetImage(r.Context(), id, size, format)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "max-age=86400")
	w.Header().Set("ETag", `"`+etag+`"`)
	if _, err := io.Copy(w, rc); err != nil {
		return
	}
}

// Metadata возвращает JSON-метаданные аватара.
// @Summary Метаданные аватара
// @Tags avatars
// @Produce json
// @Param avatarID path string true "UUID аватара"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/avatars/{avatarID}/metadata [get]
func (h *AvatarHTTP) Metadata(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "avatarID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	m, err := h.svc.Metadata(r.Context(), id, h.publicURL)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// DeleteAvatar удаляет аватар по id (только владелец).
// @Summary Удалить аватар по ID
// @Tags avatars
// @Param X-User-ID header string true "Идентификатор пользователя"
// @Param avatarID path string true "UUID аватара"
// @Success 204 "Нет тела ответа"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security UserIDAuth
// @Router /api/v1/avatars/{avatarID} [delete]
func (h *AvatarHTTP) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok || uid == "" {
		writeError(w, http.StatusBadRequest, map[string]string{"error": "X-User-ID required"})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "avatarID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(r.Context(), id, uid); err != nil {
		h.mapErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UserAvatar отдаёт текущее изображение аватара пользователя.
// @Summary Аватар пользователя по user ID
// @Tags avatars
// @Produce octet-stream
// @Param userID path string true "Идентификатор пользователя"
// @Success 200 {file} binary "Тело изображения"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{userID}/avatar [get]
func (h *AvatarHTTP) UserAvatar(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	rc, mime, etag, err := h.svc.GetImageForUser(r.Context(), userID)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "max-age=86400")
	w.Header().Set("ETag", `"`+etag+`"`)
	if _, err := io.Copy(w, rc); err != nil {
		return
	}
}

// UserAvatars возвращает список метаданных аватаров пользователя.
// @Summary Список аватаров пользователя
// @Tags avatars
// @Produce json
// @Param userID path string true "Идентификатор пользователя"
// @Success 200 {array} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{userID}/avatars [get]
func (h *AvatarHTTP) UserAvatars(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	list, err := h.svc.ListMetadata(r.Context(), userID, h.publicURL)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// DeleteUserAvatar удаляет аватар пользователя (только свой или с правами).
// @Summary Удалить аватар пользователя
// @Tags avatars
// @Param X-User-ID header string true "Идентификатор пользователя (инициатор)"
// @Param userID path string true "Идентификатор пользователя-владельца"
// @Success 204 "Нет тела ответа"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security UserIDAuth
// @Router /api/v1/users/{userID}/avatar [delete]
func (h *AvatarHTTP) DeleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok || uid == "" {
		writeError(w, http.StatusBadRequest, map[string]string{"error": "X-User-ID required"})
		return
	}
	userID := chi.URLParam(r, "userID")
	if err := h.svc.DeleteForUser(r.Context(), userID, uid); err != nil {
		h.mapErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
