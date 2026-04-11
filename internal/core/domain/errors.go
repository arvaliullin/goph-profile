package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidFormat = errors.New("invalid file format")
	ErrFileTooLarge  = errors.New("file too large")
	ErrMissingUserID = errors.New("missing user id")
	ErrMissingFile   = errors.New("missing file")
)
