package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSentinels(t *testing.T) {
	t.Parallel()
	require.True(t, errors.Is(ErrNotFound, ErrNotFound))
}
