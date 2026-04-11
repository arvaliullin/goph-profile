package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsConnectionRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, false},
		{"generic error", errors.New("something"), false},
		{
			"connection exception",
			&pgconn.PgError{Code: pgerrcode.ConnectionException},
			true,
		},
		{
			"connection failure",
			&pgconn.PgError{Code: pgerrcode.ConnectionFailure},
			true,
		},
		{
			"sql client unable",
			&pgconn.PgError{Code: pgerrcode.SQLClientUnableToEstablishSQLConnection},
			true,
		},
		{
			"protocol violation",
			&pgconn.PgError{Code: pgerrcode.ProtocolViolation},
			true,
		},
		{
			"transaction resolution unknown",
			&pgconn.PgError{Code: pgerrcode.TransactionResolutionUnknown},
			true,
		},
		{
			"non-retryable pg error",
			&pgconn.PgError{Code: pgerrcode.UniqueViolation},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsConnectionRetryable(tt.err))
		})
	}
}
