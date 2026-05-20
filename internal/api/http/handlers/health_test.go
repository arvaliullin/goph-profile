package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arvaliullin/goph-profile/internal/api/http/dto"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestHealth_Handle(t *testing.T) {
	t.Parallel()
	h := &Health{
		KafkaPing: func() error { return nil },
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var body dto.HealthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Nil(t, body.Postgres)
	require.Nil(t, body.Minio)
	require.NotNil(t, body.Kafka)
	require.True(t, body.Kafka.OK)
}

func TestHealth_UnhealthyKafka(t *testing.T) {
	t.Parallel()
	h := &Health{
		KafkaPing: func() error { return context.DeadlineExceeded },
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestHealth_WithDB(t *testing.T) {
	t.Parallel()
	h := &Health{DB: (*pgxpool.Pool)(nil), Minio: nil, KafkaPing: func() error { return nil }}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHealth_UnconfiguredComponentsOmitted(t *testing.T) {
	t.Parallel()
	h := &Health{KafkaPing: func() error { return nil }}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var body dto.HealthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Nil(t, body.Postgres)
	require.Nil(t, body.Minio)
	require.NotNil(t, body.Kafka)
}

func TestHealth_ContextCancel(t *testing.T) {
	t.Parallel()
	h := &Health{KafkaPing: func() error { return nil }}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/health", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
