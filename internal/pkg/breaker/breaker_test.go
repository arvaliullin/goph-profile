package breaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/stretchr/testify/require"
)

func TestBreakerOpensAfterConsecutiveFailures(t *testing.T) {
	t.Parallel()
	br := New("test")
	ctx := context.Background()
	errFail := errors.New("upstream down")

	for range defaultConsecutiveFails {
		_, err := br.Execute(ctx, func() (any, error) {
			return nil, errFail
		})
		require.ErrorIs(t, err, errFail)
	}

	_, err := br.Execute(ctx, func() (any, error) {
		return "ok", nil
	})
	require.ErrorIs(t, err, domain.ErrUnavailable)
}

func TestBreakerRecoversAfterSuccessInHalfOpen(t *testing.T) {
	t.Parallel()
	st := New("test-recover")
	ctx := context.Background()
	errFail := errors.New("fail")

	for range defaultConsecutiveFails {
		st.Execute(ctx, func() (any, error) { return nil, errFail })
	}
	_, err := st.Execute(ctx, func() (any, error) { return nil, nil })
	require.ErrorIs(t, err, domain.ErrUnavailable)

	time.Sleep(defaultTimeout + 50*time.Millisecond)

	v, err := st.Execute(ctx, func() (any, error) {
		return "ok", nil
	})
	require.NoError(t, err)
	require.Equal(t, "ok", v)

	v2, err := st.Execute(ctx, func() (any, error) {
		return "again", nil
	})
	require.NoError(t, err)
	require.Equal(t, "again", v2)
}
