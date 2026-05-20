package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimitByUserID(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := UserIDFromHeader(RateLimit("test", 1, time.Minute)(inner))

	do := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/avatars", nil)
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	require.Equal(t, http.StatusOK, do().Code)
	second := do()
	require.Equal(t, http.StatusTooManyRequests, second.Code)
	require.Contains(t, second.Body.String(), "rate limit exceeded")
}
