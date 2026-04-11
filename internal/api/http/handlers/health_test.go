package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
}

func TestHealth_WithDB(t *testing.T) {
	t.Parallel()
	h := &Health{DB: (*pgxpool.Pool)(nil), Minio: nil, KafkaPing: func() error { return nil }}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
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
