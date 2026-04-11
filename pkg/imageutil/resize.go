package imageutil

import (
	"bytes"
	"fmt"
	"image"
	"io"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/disintegration/imaging"
)

// DecodeAndResize декодирует изображение и возвращает закодированные байты для заданного размера.
func DecodeAndResize(r io.Reader, sizeLabel string, outFormat imaging.Format) ([]byte, string, error) {
	img, err := imaging.Decode(r)
	if err != nil {
		return nil, "", err
	}
	var side int
	switch sizeLabel {
	case domain.Thumbnail100:
		side = 100
	case domain.Thumbnail300:
		side = 300
	default:
		return nil, "", fmt.Errorf("unknown size %q", sizeLabel)
	}
	resized := imaging.Fill(img, side, side, imaging.Center, imaging.Lanczos)
	var buf bytes.Buffer
	var mime string
	switch outFormat {
	case imaging.PNG:
		mime = "image/png"
		err = imaging.Encode(&buf, resized, imaging.PNG)
	default:
		mime = "image/jpeg"
		err = imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(85))
	}
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), mime, nil
}

// Dimensions возвращает ширину и высоту декодированного изображения.
func Dimensions(r io.Reader) (int, int, error) {
	img, err := imaging.Decode(r)
	if err != nil {
		return 0, 0, err
	}
	b := img.Bounds()
	return b.Dx(), b.Dy(), nil
}

// FormatFromMIME выбирает формат imaging для превью (WebP кодируется как JPEG).
func FormatFromMIME(mime string) imaging.Format {
	if mime == "image/png" {
		return imaging.PNG
	}
	return imaging.JPEG
}

// EncodeToBytes кодирует изображение в байты.
func EncodeToBytes(img image.Image, mime string) ([]byte, error) {
	f := FormatFromMIME(mime)
	var buf bytes.Buffer
	var err error
	if f == imaging.PNG {
		err = imaging.Encode(&buf, img, imaging.PNG)
	} else {
		err = imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(90))
	}
	return buf.Bytes(), err
}
