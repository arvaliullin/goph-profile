package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/arvaliullin/goph-profile/internal/repository/minio"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}
