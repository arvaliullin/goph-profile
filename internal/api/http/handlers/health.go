package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/arvaliullin/goph-profile/internal/api/http/dto"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

const healthCheckTimeout = 3 * time.Second

// Health сводная проверка зависимостей.
type Health struct {
	DB        *pgxpool.Pool
	Minio     ports.Pinger
	KafkaPing func() error
}

// Handle сводная проверка зависимостей (PostgreSQL, MinIO, Kafka).
// @Summary Health check
// @Description Статус подключения к postgres, minio и Kafka.
// @Tags health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Router /health [get]
func (h *Health) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
	defer cancel()
	out := dto.HealthResponse{
		Postgres: dto.ComponentStatus{OK: true},
		Minio:    dto.ComponentStatus{OK: true},
		Kafka:    dto.ComponentStatus{OK: true},
	}
	if h.DB != nil {
		if err := h.DB.Ping(ctx); err != nil {
			out.Postgres = dto.ComponentStatus{OK: false, Error: err.Error()}
		}
	}
	if h.Minio != nil {
		if err := h.Minio.Ping(ctx); err != nil {
			out.Minio = dto.ComponentStatus{OK: false, Error: err.Error()}
		}
	}
	if h.KafkaPing != nil {
		if err := h.KafkaPing(); err != nil {
			out.Kafka = dto.ComponentStatus{OK: false, Error: err.Error()}
		}
	}
	writeJSON(w, http.StatusOK, out)
}
