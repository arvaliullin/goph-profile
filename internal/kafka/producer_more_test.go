package kafka

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewProducerFails(t *testing.T) {
	t.Parallel()
	_, err := NewProducer([]string{}, "a", "b")
	require.Error(t, err)
}
