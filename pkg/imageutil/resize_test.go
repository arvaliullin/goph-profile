package imageutil

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/require"
)

func TestDimensions(t *testing.T) {
	t.Parallel()
	img := imaging.New(4, 4, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	require.NoError(t, imaging.Encode(&buf, img, imaging.PNG))
	w, h, err := Dimensions(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Equal(t, 4, w)
	require.Equal(t, 4, h)
}

func TestDecodeAndResize(t *testing.T) {
	t.Parallel()
	img := imaging.New(80, 40, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, imaging.Encode(&buf, img, imaging.JPEG))
	out, mime, err := DecodeAndResize(bytes.NewReader(buf.Bytes()), domain.Thumbnail100, imaging.JPEG)
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", mime)
	require.Greater(t, len(out), 10)
}

func TestFormatFromMIME(t *testing.T) {
	t.Parallel()
	require.Equal(t, imaging.PNG, FormatFromMIME("image/png"))
	require.Equal(t, imaging.JPEG, FormatFromMIME("image/webp"))
}

func TestEncodeToBytes(t *testing.T) {
	t.Parallel()
	img := imaging.New(2, 2, color.RGBA{G: 255, A: 255})
	b, err := EncodeToBytes(img, "image/png")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(b), "\x89PNG"))
}
