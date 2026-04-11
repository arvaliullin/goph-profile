package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/arvaliullin/goph-profile/internal/repository/minio"
	"github.com/jackc/pgx/v5/pgxpool"
)

const healthCheckTimeout = 3 * time.Second

// Health сводная проверка зависимостей.
type Health struct {
	DB        *pgxpool.Pool
	Minio     *minio.Storage
	KafkaPing func() error
}

type status struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// healthResponse ответ GET /health.
type healthResponse struct {
	Postgres status `json:"postgres"`
	Minio    status `json:"minio"`
	Kafka    status `json:"kafka"`
}

// Handle сводная проверка зависимостей (PostgreSQL, MinIO, Kafka).
// @Summary Health check
// @Description Статус подключения к postgres, minio и Kafka.
// @Tags health
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health [get]
func (h *Health) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
	defer cancel()
	out := healthResponse{
		Postgres: status{OK: true},
		Minio:    status{OK: true},
		Kafka:    status{OK: true},
	}
	if h.DB != nil {
		if err := h.DB.Ping(ctx); err != nil {
			out.Postgres = status{OK: false, Error: err.Error()}
		}
	}
	if h.Minio != nil {
		if err := h.Minio.Ping(ctx); err != nil {
			out.Minio = status{OK: false, Error: err.Error()}
		}
	}
	if h.KafkaPing != nil {
		if err := h.KafkaPing(); err != nil {
			out.Kafka = status{OK: false, Error: err.Error()}
		}
	}
	writeJSON(w, http.StatusOK, out)
}
