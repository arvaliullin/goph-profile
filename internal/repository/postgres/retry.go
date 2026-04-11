package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var connectionErrorCodes = map[string]struct{}{
	pgerrcode.ConnectionException:                           {},
	pgerrcode.ConnectionDoesNotExist:                        {},
	pgerrcode.ConnectionFailure:                             {},
	pgerrcode.SQLClientUnableToEstablishSQLConnection:       {},
	pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection: {},
	pgerrcode.TransactionResolutionUnknown:                  {},
	pgerrcode.ProtocolViolation:                             {},
}

// IsConnectionRetryable определяет, относится ли ошибка к классу сбоев соединения PostgreSQL.
func IsConnectionRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	pgErr, pgOK := errors.AsType[*pgconn.PgError](err)
	if pgOK {
		_, inMap := connectionErrorCodes[pgErr.Code]
		return inMap
	}

	_, connectOK := errors.AsType[*pgconn.ConnectError](err)
	return connectOK
}
