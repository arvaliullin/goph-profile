package ports

import "context"

//go:generate mockgen -source=publisher.go -destination=mocks/publisher_mock.go -package=mocks

// EventPublisher публикует события предметной области в Kafka.
type EventPublisher interface {
	PublishUpload(ctx context.Context, e AvatarUploadEvent) error
	PublishDelete(ctx context.Context, e AvatarDeleteEvent) error
	Close() error
}
