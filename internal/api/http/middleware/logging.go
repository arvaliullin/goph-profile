package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// RequestLogger логирует метод, путь, статус и длительность.
func RequestLogger(log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrap := &respWrap{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrap, r)
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrap.status).
				Dur("dur", time.Since(start)).
				Msg("http")
		})
	}
}

type respWrap struct {
	http.ResponseWriter
	status int
}

func (w *respWrap) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
