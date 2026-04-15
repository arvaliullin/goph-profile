package kafka

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPingInvalidBroker(t *testing.T) {
	t.Parallel()
	err := Ping([]string{"127.0.0.1:1"})
	require.Error(t, err)
}
