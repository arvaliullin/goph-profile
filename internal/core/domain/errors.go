package domain

import "errors"

// Ошибки доменного слоя; текст сообщений используется в HTTP API.
var (
	// ErrNotFound сущность не найдена.
	ErrNotFound = errors.New("not found")
	// ErrForbidden операция запрещена для текущего пользователя.
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidFormat недопустимый формат файла.
	ErrInvalidFormat = errors.New("invalid file format")
	// ErrFileTooLarge размер файла превышает лимит.
	ErrFileTooLarge = errors.New("file too large")
	// ErrMissingUserID не задан идентификатор пользователя.
	ErrMissingUserID = errors.New("missing user id")
	// ErrMissingFile файл не передан.
	ErrMissingFile = errors.New("missing file")
)
