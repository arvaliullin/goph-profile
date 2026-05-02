package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/arvaliullin/goph-profile/internal/app"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "avatard")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.NewAvatard(ctx)
	if err != nil {
		log.Error("avatard init", "error", err)
		os.Exit(1)
	}

	if err := a.Run(ctx); err != nil {
		a.Logger().Error("consumer", "error", err)
		os.Exit(1)
	}
}
