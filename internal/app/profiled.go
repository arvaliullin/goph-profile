package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	httpserver "github.com/arvaliullin/goph-profile/internal/api/http"
	"github.com/arvaliullin/goph-profile/internal/api/http/handlers"
	"github.com/arvaliullin/goph-profile/internal/config"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/core/services/avatar"
	"github.com/arvaliullin/goph-profile/internal/kafka"
	"github.com/arvaliullin/goph-profile/internal/repository/minio"
	"github.com/arvaliullin/goph-profile/internal/repository/postgres"
	"github.com/rs/zerolog"
)

const (
	httpReadHeaderTimeout = 10 * time.Second
	httpReadWriteTimeout  = 60 * time.Second
)

// Profiled HTTP-сервис profiled (REST API и health).
type Profiled struct {
	cfg  *config.Server
	log  zerolog.Logger
	db   *postgres.DB
	prod *kafka.Producer
	srv  *http.Server
}

// NewProfiled собирает зависимости и HTTP-сервер.
func NewProfiled(ctx context.Context) (*Profiled, error) {
	cfg, err := config.LoadServer()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	log := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel)

	log.Info().
		Str("http_addr", cfg.HTTPAddr).
		Str("minio_endpoint", cfg.MinioEndpoint).
		Str("minio_bucket", cfg.MinioBucket).
		Bool("minio_secure", cfg.MinioSecure).
		Str("kafka_brokers", cfg.KafkaBrokers).
		Str("kafka_topic_upload", cfg.KafkaTopicUp).
		Str("kafka_topic_delete", cfg.KafkaTopicDel).
		Int64("max_upload_bytes", cfg.MaxUploadBytes).
		Dur("shutdown_timeout", cfg.ShutdownTimeout).
		Str("public_base_url", cfg.PublicBaseURL).
		Msg("profiled configuration loaded")

	if err = postgres.RunMigrations(ctx, cfg.DatabaseURI); err != nil {
		return nil, fmt.Errorf("postgres migrations: %w", err)
	}
	db, err := postgres.New(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	st, err := minio.New(ctx, cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioSecure, cfg.MinioBucket)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("minio: %w", err)
	}

	prod, err := kafka.NewProducer(config.KafkaBrokerList(cfg.KafkaBrokers), cfg.KafkaTopicUp, cfg.KafkaTopicDel, log)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("kafka producer: %w", err)
	}

	repo := postgres.NewAvatarRepository(db.Pool)
	svc := avatar.New(repo, st, prod, nil, cfg.MaxUploadBytes)

	avh := handlers.NewAvatarHTTP(svc, cfg.MaxUploadBytes, cfg.PublicBaseURL)
	health := &handlers.Health{
		DB:    db.Pool,
		Minio: ports.Pinger(st),
		KafkaPing: func() error {
			return kafka.Ping(config.KafkaBrokerList(cfg.KafkaBrokers))
		},
	}
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.NewRouter(httpserver.Deps{Log: log, Avatar: avh, Health: health}),
		ReadHeaderTimeout: httpReadHeaderTimeout,
		ReadTimeout:       httpReadWriteTimeout,
		WriteTimeout:      httpReadWriteTimeout,
	}

	return &Profiled{
		cfg:  cfg,
		log:  log,
		db:   db,
		prod: prod,
		srv:  srv,
	}, nil
}

// Logger возвращает логгер приложения.
func (a *Profiled) Logger() *zerolog.Logger {
	return &a.log
}

// Run слушает сигналы, останавливает HTTP и закрывает ресурсы.
func (a *Profiled) Run(ctx context.Context) error {
	go func() {
		a.log.Info().Str("addr", a.srv.Addr).Msg("listen")
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.log.Fatal().Err(err).Msg("http")
		}
	}()

	<-ctx.Done()
	a.log.Info().Msg("shutting down profiled")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()
	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		a.log.Error().Err(err).Msg("http shutdown")
	}

	if err := a.prod.Close(); err != nil {
		a.log.Error().Err(err).Msg("kafka producer close")
	}

	a.db.Close()
	return nil
}
