package imageutil

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeContentType(t *testing.T) {
	t.Parallel()
	m, err := NormalizeContentType("image/png", nil)
	require.NoError(t, err)
	require.Equal(t, "image/png", m)

	m, err = NormalizeContentType("", []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	require.NoError(t, err)
	require.Equal(t, "image/png", m)
}

func TestFormatFromQuery(t *testing.T) {
	t.Parallel()
	m, err := FormatFromQuery("webp")
	require.NoError(t, err)
	require.Equal(t, "image/webp", m)
	_, err = FormatFromQuery("gif")
	require.Error(t, err)
}

func TestErrBodies(t *testing.T) {
	t.Parallel()
	d := ErrInvalidFormatDetail()
	require.Equal(t, "Invalid file format", d["error"])
	fl := ErrFileTooLarge(10)
	require.EqualValues(t, 10, fl["max_size"])
}

func TestReadSniffPrefix(t *testing.T) {
	t.Parallel()
	prefix, r, err := ReadSniffPrefix(strings.NewReader("abc"), 2)
	require.NoError(t, err)
	require.Len(t, prefix, 2)
	all, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "abc", string(all))
}
