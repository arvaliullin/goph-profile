package retry

import (
	"context"
	"fmt"
	"time"
)

var (
	// ErrAttemptFuncNil возвращается, когда action не передан.
	ErrAttemptFuncNil = fmt.Errorf("retry action is nil")
	// ErrStrategyNil возвращается, когда стратегия повторов не инициализирована.
	ErrStrategyNil = fmt.Errorf("retry strategy is nil")
)

// DefaultDelays задаёт интервалы между попытками по умолчанию.
var DefaultDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

// AttemptFunc описывает операцию, которую нужно выполнить с повторами.
type AttemptFunc func(ctx context.Context) error

// ShouldRetryFunc определяет, нужно ли повторять операцию после ошибки.
type ShouldRetryFunc func(err error) bool

// Strategy описывает конфигурацию повторных попыток.
type Strategy struct {
	delays      []time.Duration
	shouldRetry ShouldRetryFunc
}

// NewStrategy создаёт новую стратегию повторов.
func NewStrategy(delays []time.Duration, shouldRetry ShouldRetryFunc) *Strategy {
	delayCopy := make([]time.Duration, len(delays))
	copy(delayCopy, delays)

	if len(delayCopy) == 0 {
		delayCopy = append(delayCopy, DefaultDelays...)
	}

	predicate := shouldRetry
	if predicate == nil {
		predicate = func(error) bool {
			return true
		}
	}

	for i := range delayCopy {
		if delayCopy[i] < 0 {
			delayCopy[i] = 0
		}
	}

	return &Strategy{
		delays:      delayCopy,
		shouldRetry: predicate,
	}
}

// DoWithRetry выполняет операцию с ограниченным числом повторов.
func (s *Strategy) DoWithRetry(ctx context.Context, attempt AttemptFunc) error {
	if s == nil {
		return ErrStrategyNil
	}
	if attempt == nil {
		return ErrAttemptFuncNil
	}

	var lastErr error

	for idx := 0; idx <= len(s.delays); idx++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := attempt(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		if !s.shouldRetry(err) || idx == len(s.delays) {
			return err
		}

		delay := s.delays[idx]
		if delay <= 0 {
			continue
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return lastErr
}

// ExponentialBackoffDelays возвращает steps задержек между попытками: base, затем удвоение до потолка maxDelay.
// Первая задержка применяется после первой неудачной попытки (всего до steps+1 вызовов attempt в DoWithRetry).
func ExponentialBackoffDelays(base, maxDelay time.Duration, steps int) []time.Duration {
	if steps < 1 {
		steps = 1
	}
	if base <= 0 {
		base = time.Millisecond
	}
	if maxDelay < base {
		maxDelay = base
	}
	out := make([]time.Duration, steps)
	d := base
	for i := range steps {
		if d > maxDelay {
			d = maxDelay
		}
		out[i] = d
		if d >= maxDelay {
			d = maxDelay
			continue
		}
		next := d * 2
		if next < d || next > maxDelay {
			d = maxDelay
		} else {
			d = next
		}
	}
	return out
}
