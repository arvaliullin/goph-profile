package ports

import "context"

// Pinger описывает зависимость, доступность которой можно проверить.
type Pinger interface {
	Ping(ctx context.Context) error
}
