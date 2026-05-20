package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/go-chi/httprate"
)

// RateLimit ограничивает число запросов.
func RateLimit(service string, requests int, window time.Duration) func(http.Handler) http.Handler {
	if requests <= 0 || window <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return httprate.Limit(
		requests,
		window,
		httprate.WithKeyFuncs(rateLimitKey),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			observability.ObserveRateLimitRejected(service)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
		}),
	)
}

func rateLimitKey(r *http.Request) (string, error) {
	if uid, ok := UserID(r.Context()); ok && uid != "" {
		return "user:" + uid, nil
	}
	return httprate.KeyByIP(r)
}
