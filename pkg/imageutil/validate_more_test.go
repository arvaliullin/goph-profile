package imageutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeContentTypeInvalid(t *testing.T) {
	t.Parallel()
	_, err := NormalizeContentType("image/gif", nil)
	require.Error(t, err)
}

func TestFormatFromQueryJPEGAliases(t *testing.T) {
	t.Parallel()
	m, err := FormatFromQuery("jpg")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", m)
}
