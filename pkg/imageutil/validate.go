package imageutil

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
)

// allowed допустимые MIME-типы для загрузки.
var allowed = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// NormalizeContentType возвращает канонический MIME по заголовку или сигнатуре байтов.
func NormalizeContentType(header string, sniff []byte) (string, error) {
	h := strings.TrimSpace(strings.ToLower(header))
	if h != "" {
		if _, ok := allowed[h]; ok {
			return h, nil
		}
	}
	if len(sniff) >= 3 && sniff[0] == 0xFF && sniff[1] == 0xD8 {
		return "image/jpeg", nil
	}
	if len(sniff) >= 8 && sniff[0] == 0x89 && sniff[1] == 0x50 {
		return "image/png", nil
	}
	if len(sniff) >= 12 && sniff[0] == 'R' && sniff[1] == 'I' && sniff[2] == 'F' && sniff[3] == 'F' {
		return "image/webp", nil
	}
	return "", domain.ErrInvalidFormat
}

// ReadSniffPrefix читает первые n байт для определения формата.
func ReadSniffPrefix(r io.Reader, n int) ([]byte, io.Reader, error) {
	buf := make([]byte, n)
	m, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, nil, err
	}
	pr := io.MultiReader(bytes.NewReader(buf[:m]), r)
	return buf[:m], pr, nil
}

// ErrInvalidFormatDetail возвращает тело ответа API при ошибке 400.
func ErrInvalidFormatDetail() map[string]any {
	return map[string]any{
		"error":   "Invalid file format",
		"details": "Supported formats: jpeg, png, webp",
	}
}

// ErrFileTooLarge возвращает тело ответа API при ошибке 413.
func ErrFileTooLarge(max int64) map[string]any {
	return map[string]any{
		"error":    "File too large",
		"max_size": max,
	}
}

// FormatFromQuery сопоставляет строку формата с MIME.
func FormatFromQuery(f string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "", "jpeg", "jpg":
		return "image/jpeg", nil
	case "png":
		return "image/png", nil
	case "webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("unsupported format")
	}
}
