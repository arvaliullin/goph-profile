package kafka

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewProducerFails(t *testing.T) {
	t.Parallel()
	_, err := NewProducer([]string{}, "a", "b", 1<<20, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	require.Error(t, err)
}
