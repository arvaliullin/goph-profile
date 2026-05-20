package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/arvaliullin/goph-profile/internal/repository/postgres"
)

const migrationTimeout = 60 * time.Second

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "migrate")

	dsn := os.Getenv("DATABASE_URI")
	if dsn == "" {
		log.Error("empty DATABASE_URI")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	if err := postgres.RunMigrations(ctx, dsn); err != nil {
		log.Error("run migrations", "error", err)
		os.Exit(1)
	}

	log.Info("migrations applied")
}
