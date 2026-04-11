package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestRequestLogger(t *testing.T) {
	t.Parallel()
	log := zerolog.Nop()
	h := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/z", nil)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTeapot, rec.Code)
}
