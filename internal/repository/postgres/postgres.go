package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/arvaliullin/goph-profile/internal/pkg/retry"
	_ "github.com/arvaliullin/goph-profile/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// DB обертка над pgx pool.
type DB struct {
	Pool *pgxpool.Pool
}

// RunMigrations применяет миграции goose; каталог `migrations` ищется относительно рабочей директории процесса.
func RunMigrations(ctx context.Context, dsn string) error {
	strategy := retry.NewStrategy(nil, IsConnectionRetryable)
	return strategy.DoWithRetry(ctx, func(ctx context.Context) error {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		if err := goose.SetDialect("postgres"); err != nil {
			return err
		}
		return goose.UpContext(ctx, db, "migrations")
	})
}

// New создаёт пул соединений pgx и проверяет доступность базы через Ping.
func New(ctx context.Context, dsn string) (*DB, error) {
	strategy := retry.NewStrategy(nil, IsConnectionRetryable)
	var pool *pgxpool.Pool
	err := strategy.DoWithRetry(ctx, func(ctx context.Context) error {
		p, e := pgxpool.New(ctx, dsn)
		if e != nil {
			return fmt.Errorf("pool: %w", e)
		}
		if e = p.Ping(ctx); e != nil {
			p.Close()
			return fmt.Errorf("ping: %w", e)
		}
		pool = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &DB{Pool: pool}, nil
}

// Close освобождает pool.
func (d *DB) Close() {
	d.Pool.Close()
}
