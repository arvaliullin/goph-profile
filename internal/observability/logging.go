package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// NewLogger создает JSON slog-логгер с выбранным уровнем.
func NewLogger(service, level string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(level),
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler).With("service", service)
}

// LoggerWithTrace добавляет trace_id и span_id в логгер, если есть активный span.
func LoggerWithTrace(ctx context.Context, log *slog.Logger) *slog.Logger {
	if log == nil {
		return slog.Default()
	}
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return log
	}
	return log.With(
		"trace_id", spanCtx.TraceID().String(),
		"span_id", spanCtx.SpanID().String(),
	)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
