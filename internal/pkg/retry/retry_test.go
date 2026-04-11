package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStrategy_Defaults(t *testing.T) {
	strategy := NewStrategy(nil, nil)
	require.NotNil(t, strategy)

	expected := []time.Duration{
		time.Second,
		3 * time.Second,
		5 * time.Second,
	}
	assert.Equal(t, expected, strategy.delays)
	assert.True(t, strategy.shouldRetry(errors.New("any")))
}

func TestNewStrategy_NormalizesNegativeDelays(t *testing.T) {
	strategy := NewStrategy([]time.Duration{-time.Second, 0, time.Second}, nil)
	assert.Equal(t, time.Duration(0), strategy.delays[0])
	assert.Equal(t, time.Duration(0), strategy.delays[1])
	assert.Equal(t, time.Second, strategy.delays[2])
}

func TestDoWithRetry_RetriesUntilSuccess(t *testing.T) {
	strategy := NewStrategy([]time.Duration{0, 0, 0}, func(error) bool { return true })

	attempts := 0
	err := strategy.DoWithRetry(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("fail")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestDoWithRetry_StopsWhenPredicateFalse(t *testing.T) {
	strategy := NewStrategy([]time.Duration{0, 0, 0}, func(error) bool { return false })

	attempts := 0
	testErr := errors.New("fail")
	err := strategy.DoWithRetry(context.Background(), func(ctx context.Context) error {
		attempts++
		return testErr
	})

	assert.ErrorIs(t, err, testErr)
	assert.Equal(t, 1, attempts)
}

func TestDoWithRetry_NilAttemptFunc(t *testing.T) {
	strategy := NewStrategy(nil, nil)
	err := strategy.DoWithRetry(context.Background(), nil)
	assert.ErrorIs(t, err, ErrAttemptFuncNil)
}

func TestDoWithRetry_NilStrategy(t *testing.T) {
	var strategy *Strategy
	err := strategy.DoWithRetry(context.Background(), func(ctx context.Context) error {
		return nil
	})
	assert.ErrorIs(t, err, ErrStrategyNil)
}

func TestDoWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	strategy := NewStrategy([]time.Duration{time.Hour}, func(error) bool { return true })

	attempts := 0
	err := strategy.DoWithRetry(ctx, func(ctx context.Context) error {
		attempts++
		cancel()
		return errors.New("fail")
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, attempts)
}

func TestExponentialBackoffDelays(t *testing.T) {
	t.Parallel()
	got := ExponentialBackoffDelays(100*time.Millisecond, 500*time.Millisecond, 5)
	require.Len(t, got, 5)
	assert.Equal(t, 100*time.Millisecond, got[0])
	assert.Equal(t, 200*time.Millisecond, got[1])
	assert.Equal(t, 400*time.Millisecond, got[2])
	assert.Equal(t, 500*time.Millisecond, got[3])
	assert.Equal(t, 500*time.Millisecond, got[4])
}

func TestExponentialBackoffDelays_normalizesBaseAndSteps(t *testing.T) {
	t.Parallel()
	got := ExponentialBackoffDelays(0, time.Second, 1)
	require.Len(t, got, 1)
	assert.Equal(t, time.Millisecond, got[0])
}
