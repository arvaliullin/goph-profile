package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/arvaliullin/goph-profile/internal/config"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/kafka"
	"github.com/arvaliullin/goph-profile/internal/repository/minio"
	"github.com/arvaliullin/goph-profile/internal/repository/postgres"
	"github.com/arvaliullin/goph-profile/internal/worker"
	"github.com/rs/zerolog"
)

type kafkaBridge struct {
	p *worker.Processor
}

func (b *kafkaBridge) OnUpload(ctx context.Context, payload []byte) error {
	return b.p.HandleUpload(ctx, payload)
}

func (b *kafkaBridge) OnDelete(ctx context.Context, payload []byte) error {
	return b.p.HandleDelete(ctx, payload)
}

var _ ports.GroupHandler = (*kafkaBridge)(nil)

// Avatard воркер обработки Kafka (миниатюры, удаление в S3).
type Avatard struct {
	cfg  *config.Worker
	log  zerolog.Logger
	db   *postgres.DB
	proc *worker.Processor
}

// NewAvatard собирает зависимости воркера.
func NewAvatard(ctx context.Context) (*Avatard, error) {
	cfg, err := config.LoadWorker()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	log := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel)

	log.Info().
		Str("minio_endpoint", cfg.MinioEndpoint).
		Str("minio_bucket", cfg.MinioBucket).
		Bool("minio_secure", cfg.MinioSecure).
		Str("kafka_brokers", cfg.KafkaBrokers).
		Str("kafka_topic_upload", cfg.KafkaTopicUp).
		Str("kafka_topic_delete", cfg.KafkaTopicDel).
		Str("kafka_group", cfg.KafkaGroup).
		Dur("shutdown_timeout", cfg.ShutdownTimeout).
		Msg("avatard configuration loaded")

	db, err := postgres.New(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	st, err := minio.New(ctx, cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioSecure, cfg.MinioBucket)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("minio: %w", err)
	}

	repo := postgres.NewAvatarRepository(db.Pool)
	proc := worker.NewProcessor(repo, st, log)

	return &Avatard{
		cfg:  cfg,
		log:  log,
		db:   db,
		proc: proc,
	}, nil
}

// Logger возвращает логгер приложения.
func (a *Avatard) Logger() *zerolog.Logger {
	return &a.log
}

// Run блокируется на consumer group до отмены ctx, затем закрывает БД.
func (a *Avatard) Run(ctx context.Context) error {
	defer a.db.Close()

	kcfg := kafka.Config{TopicUpload: a.cfg.KafkaTopicUp, TopicDelete: a.cfg.KafkaTopicDel}
	brokers := config.KafkaBrokerList(a.cfg.KafkaBrokers)
	br := &kafkaBridge{p: a.proc}

	a.log.Info().Strs("brokers", brokers).Msg("avatard starting consumer")

	err := kafka.RunConsumerGroup(ctx, brokers, a.cfg.KafkaGroup, kcfg, br, a.log)
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("consumer: %w", err)
	}

	return nil
}
