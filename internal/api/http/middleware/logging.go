package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

// RequestLogger логирует метод, путь, статус и длительность.
func RequestLogger(log *slog.Logger, service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrap := &respWrap{ResponseWriter: w, status: http.StatusOK}
			span := trace.SpanContextFromContext(r.Context())
			if span.IsValid() {
				w.Header().Set("X-Trace-ID", span.TraceID().String())
			}
			next.ServeHTTP(wrap, r)
			route := routePattern(r)
			dur := time.Since(start)
			observability.ObserveHTTPRequest(service, r.Method, route, wrap.status, dur)

			logger := observability.LoggerWithTrace(r.Context(), log)
			logger.InfoContext(r.Context(), "http",
				"method", r.Method,
				"path", r.URL.Path,
				"route", route,
				"status", wrap.status,
				"duration_ms", dur.Milliseconds(),
			)
		})
	}
}

func routePattern(r *http.Request) string {
	pattern := r.URL.Path
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return pattern
	}
	route := rctx.RoutePattern()
	if route == "" {
		return pattern
	}
	return route
}

type respWrap struct {
	http.ResponseWriter
	status int
}

func (w *respWrap) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
