package breaker

import (
	"context"

	"github.com/arvaliullin/goph-profile/internal/core/ports"
)

// Publisher оборачивает EventPublisher circuit breaker.
type Publisher struct {
	inner ports.EventPublisher
	br    *Breaker
}

// WrapPublisher возвращает EventPublisher с circuit breaker.
func WrapPublisher(inner ports.EventPublisher, br *Breaker) ports.EventPublisher {
	if br == nil {
		return inner
	}
	return &Publisher{inner: inner, br: br}
}

// PublishUpload публикует событие загрузки.
func (p *Publisher) PublishUpload(ctx context.Context, e ports.AvatarUploadEvent) error {
	_, err := p.br.Execute(ctx, func() (any, error) {
		return nil, p.inner.PublishUpload(ctx, e)
	})
	return err
}

// PublishDelete публикует событие удаления.
func (p *Publisher) PublishDelete(ctx context.Context, e ports.AvatarDeleteEvent) error {
	_, err := p.br.Execute(ctx, func() (any, error) {
		return nil, p.inner.PublishDelete(ctx, e)
	})
	return err
}

// Close закрывает producer без circuit breaker.
func (p *Publisher) Close() error {
	return p.inner.Close()
}
