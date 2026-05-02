package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/arvaliullin/goph-profile/internal/config"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/kafka"
	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/arvaliullin/goph-profile/internal/repository/minio"
	"github.com/arvaliullin/goph-profile/internal/repository/postgres"
	"github.com/arvaliullin/goph-profile/internal/worker"
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
	cfg           *config.Worker
	log           *slog.Logger
	db            *postgres.DB
	proc          *worker.Processor
	traceShutdown func(context.Context) error
}

// NewAvatard собирает зависимости воркера.
func NewAvatard(ctx context.Context) (*Avatard, error) {
	cfg, err := config.LoadWorker()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	log := observability.NewLogger(cfg.ServiceName, cfg.LogLevel)
	traceShutdown, err := observability.InitTracing(ctx, observability.TelemetryConfig{
		ServiceName:  cfg.ServiceName,
		Environment:  cfg.Environment,
		OTLPEndpoint: cfg.OTLPEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}

	log.Info("avatard configuration loaded",
		"minio_endpoint", cfg.MinioEndpoint,
		"minio_bucket", cfg.MinioBucket,
		"minio_secure", cfg.MinioSecure,
		"kafka_brokers", cfg.KafkaBrokers,
		"kafka_topic_upload", cfg.KafkaTopicUp,
		"kafka_topic_delete", cfg.KafkaTopicDel,
		"kafka_group", cfg.KafkaGroup,
		"shutdown_timeout", cfg.ShutdownTimeout.String(),
		"metrics_addr", cfg.MetricsAddr,
	)

	db, err := postgres.New(ctx, cfg.DatabaseURI)
	if err != nil {
		if shutdownErr := traceShutdown(context.Background()); shutdownErr != nil {
			log.Error("trace shutdown after postgres failure", "error", shutdownErr)
		}
		return nil, fmt.Errorf("postgres: %w", err)
	}

	st, err := minio.New(ctx, cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioSecure, cfg.MinioBucket)
	if err != nil {
		db.Close()
		if shutdownErr := traceShutdown(context.Background()); shutdownErr != nil {
			log.Error("trace shutdown after minio failure", "error", shutdownErr)
		}
		return nil, fmt.Errorf("minio: %w", err)
	}

	repo := postgres.NewAvatarRepository(db.Pool)
	proc := worker.NewProcessor(repo, st, log)

	return &Avatard{
		cfg:           cfg,
		log:           log,
		db:            db,
		proc:          proc,
		traceShutdown: traceShutdown,
	}, nil
}

// Logger возвращает логгер приложения.
func (a *Avatard) Logger() *slog.Logger {
	return a.log
}

// Run блокируется на consumer group до отмены ctx, затем закрывает БД.
func (a *Avatard) Run(ctx context.Context) error {
	defer a.db.Close()
	defer func() {
		if a.traceShutdown == nil {
			return
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()
		if err := a.traceShutdown(shutdownCtx); err != nil {
			a.log.Error("trace shutdown", "error", err)
		}
	}()

	kcfg := kafka.Config{TopicUpload: a.cfg.KafkaTopicUp, TopicDelete: a.cfg.KafkaTopicDel}
	brokers := config.KafkaBrokerList(a.cfg.KafkaBrokers)
	br := &kafkaBridge{p: a.proc}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := a.db.Pool.Stat()
				observability.ObserveDBPool(a.cfg.ServiceName, stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns())
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runMetricsServer(ctx, a.cfg.MetricsAddr, a.log); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Error("metrics server", "error", err)
		}
	}()
	a.log.Info("avatard starting consumer", "brokers", brokers)

	err := kafka.RunConsumerGroup(ctx, brokers, a.cfg.KafkaGroup, kcfg, br, a.log)
	wg.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("consumer: %w", err)
	}
	stats := a.db.Pool.Stat()
	observability.ObserveDBPool(a.cfg.ServiceName, stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns())

	return nil
}

func runMetricsServer(ctx context.Context, addr string, log *slog.Logger) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", observability.MetricsHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("metrics server shutdown", "error", err)
		}
	}()
	log.Info("metrics server listen", "addr", addr)
	return srv.ListenAndServe()
}
