package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/arvaliullin/goph-profile/internal/app"
	"github.com/rs/zerolog"

	_ "github.com/arvaliullin/goph-profile/docs" // swagger
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.NewProfiled(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("profiled init")
	}

	if err := a.Run(ctx); err != nil {
		a.Logger().Error().Err(err).Msg("profiled run")
		os.Exit(1)
	}
}
