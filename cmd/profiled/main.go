package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/arvaliullin/goph-profile/internal/app"

	_ "github.com/arvaliullin/goph-profile/docs" // swagger
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "profiled")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.NewProfiled(ctx)
	if err != nil {
		log.Error("profiled init", "error", err)
		os.Exit(1)
	}

	if err := a.Run(ctx); err != nil {
		a.Logger().Error("profiled run", "error", err)
		os.Exit(1)
	}
}
