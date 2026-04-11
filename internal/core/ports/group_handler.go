package ports

import "context"

//go:generate mockgen -source=group_handler.go -destination=mocks/group_handler_mock.go -package=mocks

// GroupHandler обрабатывает полезную нагрузку сообщений consumer group.
type GroupHandler interface {
	OnUpload(ctx context.Context, payload []byte) error
	OnDelete(ctx context.Context, payload []byte) error
}
