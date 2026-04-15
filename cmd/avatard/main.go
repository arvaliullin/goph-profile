package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/arvaliullin/goph-profile/internal/app"
	"github.com/rs/zerolog"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.NewAvatard(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("avatard init")
	}

	if err := a.Run(ctx); err != nil {
		a.Logger().Fatal().Err(err).Msg("consumer")
	}
}
