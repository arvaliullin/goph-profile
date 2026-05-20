package breaker

import (
	"context"
	"errors"
	"time"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/sony/gobreaker"
)

const (
	defaultMaxRequests      = 3
	defaultInterval         = 60 * time.Second
	defaultTimeout          = 30 * time.Second
	defaultConsecutiveFails = 5

	gaugeStateClosed   = 0
	gaugeStateOpen     = 1
	gaugeStateHalfOpen = 2
)

// Breaker обёртка circuit breaker для внешней зависимости.
type Breaker struct {
	cb *gobreaker.CircuitBreaker
}

// New создаёт breaker с именем dependency для метрик.
func New(name string) *Breaker {
	st := gobreaker.Settings{
		Name:        name,
		MaxRequests: defaultMaxRequests,
		Interval:    defaultInterval,
		Timeout:     defaultTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= defaultConsecutiveFails
		},
		OnStateChange: func(_ string, _ gobreaker.State, to gobreaker.State) {
			observability.SetCircuitBreakerState(name, stateToGauge(to))
		},
	}
	cb := gobreaker.NewCircuitBreaker(st)
	observability.SetCircuitBreakerState(name, stateToGauge(cb.State()))
	return &Breaker{cb: cb}
}

// ForPostgres возвращает breaker для PostgreSQL.
func ForPostgres() *Breaker { return New("postgres") }

// ForMinio возвращает breaker для MinIO.
func ForMinio() *Breaker { return New("minio") }

// ForKafka возвращает breaker для Kafka.
func ForKafka() *Breaker { return New("kafka") }

// Execute выполняет fn через circuit breaker с учётом ctx.
func (b *Breaker) Execute(ctx context.Context, fn func() (any, error)) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v, err := b.cb.Execute(func() (any, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return fn()
	})
	return v, mapError(err)
}

func stateToGauge(s gobreaker.State) float64 {
	switch s {
	case gobreaker.StateClosed:
		return gaugeStateClosed
	case gobreaker.StateOpen:
		return gaugeStateOpen
	case gobreaker.StateHalfOpen:
		return gaugeStateHalfOpen
	default:
		return gaugeStateClosed
	}
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return domain.ErrUnavailable
	}
	return err
}
