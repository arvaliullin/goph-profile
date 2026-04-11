package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_InvalidDSN(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := New(ctx, "postgres://invalid:invalid@127.0.0.1:1/nope?sslmode=disable")
	require.Error(t, err)
}
