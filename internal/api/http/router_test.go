package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arvaliullin/goph-profile/internal/api/http/handlers"
	"github.com/arvaliullin/goph-profile/internal/core/ports/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	_ "github.com/arvaliullin/goph-profile/docs"
)

func testAvatarHandler(t *testing.T) *handlers.AvatarHTTP {
	t.Helper()
	ctrl := gomock.NewController(t)
	return handlers.NewAvatarHTTP(mocks.NewMockAvatarService(ctrl), 1024, "")
}

func TestNewRouterHealth(t *testing.T) {
	t.Parallel()
	log := zerolog.Nop()
	h := &handlers.Health{}
	av := testAvatarHandler(t)
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
	av := testAvatarHandler(t)
	r := NewRouter(Deps{Log: log, Health: h, Avatar: av})
	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
