package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httpserver "github.com/arvaliullin/goph-profile/internal/api/http"
	"github.com/arvaliullin/goph-profile/internal/api/http/handlers"
	"github.com/arvaliullin/goph-profile/internal/config"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/core/services/avatar"
	"github.com/arvaliullin/goph-profile/internal/kafka"
	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/arvaliullin/goph-profile/internal/pkg/breaker"
	"github.com/arvaliullin/goph-profile/internal/repository/minio"
	"github.com/arvaliullin/goph-profile/internal/repository/postgres"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	httpReadHeaderTimeout         = 10 * time.Second
	httpReadWriteTimeout          = 60 * time.Second
	profiledDBPoolObserveInterval = 10 * time.Second
)

// Profiled HTTP-сервис profiled (REST API и health).
type Profiled struct {
	cfg           *config.Server
	log           *slog.Logger
	db            *postgres.DB
	prod          *kafka.Producer
	srv           *http.Server
	traceShutdown func(context.Context) error
}

// NewProfiled собирает зависимости и HTTP-сервер.
func NewProfiled(ctx context.Context) (*Profiled, error) {
	cfg, err := config.LoadServer()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	log := observability.NewLogger(cfg.Telemetry.ServiceName, cfg.LogLevel)
	traceShutdown, err := observability.InitTracing(ctx, observability.TelemetryConfig{
		ServiceName:  cfg.Telemetry.ServiceName,
		Environment:  cfg.Telemetry.Environment,
		OTLPEndpoint: cfg.Telemetry.OTLPEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}

	log.Info("profiled configuration loaded",
		"http_addr", cfg.HTTPAddr,
		"minio_endpoint", cfg.Minio.Endpoint,
		"minio_bucket", cfg.Minio.Bucket,
		"minio_secure", cfg.Minio.Secure,
		"kafka_brokers", cfg.Kafka.Brokers,
		"kafka_topic_upload", cfg.Kafka.TopicUpload,
		"kafka_topic_delete", cfg.Kafka.TopicDelete,
		"kafka_max_message_bytes", cfg.Kafka.MaxMessageBytes,
		"max_upload_bytes", cfg.MaxUploadBytes,
		"shutdown_timeout", cfg.ShutdownTimeout.String(),
		"public_base_url", cfg.PublicBaseURL,
	)

	if err = postgres.RunMigrations(ctx, cfg.DatabaseURI); err != nil {
		if shutdownErr := traceShutdown(context.Background()); shutdownErr != nil {
			log.Error("trace shutdown after migrations failure", "error", shutdownErr)
		}
		return nil, fmt.Errorf("postgres migrations: %w", err)
	}
	db, err := postgres.New(ctx, cfg.DatabaseURI)
	if err != nil {
		if shutdownErr := traceShutdown(context.Background()); shutdownErr != nil {
			log.Error("trace shutdown after postgres failure", "error", shutdownErr)
		}
		return nil, fmt.Errorf("postgres: %w", err)
	}

	st, err := minio.New(ctx, cfg.Minio.Endpoint, cfg.Minio.AccessKey, cfg.Minio.SecretKey, cfg.Minio.Secure, cfg.Minio.Bucket)
	if err != nil {
		db.Close()
		if shutdownErr := traceShutdown(context.Background()); shutdownErr != nil {
			log.Error("trace shutdown after minio failure", "error", shutdownErr)
		}
		return nil, fmt.Errorf("minio: %w", err)
	}

	prod, err := kafka.NewProducer(
		config.KafkaBrokerList(cfg.Kafka.Brokers),
		cfg.Kafka.TopicUpload,
		cfg.Kafka.TopicDelete,
		cfg.Kafka.MaxMessageBytes,
		log,
	)
	if err != nil {
		db.Close()
		if shutdownErr := traceShutdown(context.Background()); shutdownErr != nil {
			log.Error("trace shutdown after kafka producer failure", "error", shutdownErr)
		}
		return nil, fmt.Errorf("kafka producer: %w", err)
	}

	var repo ports.AvatarRepository = postgres.NewAvatarRepository(db.Pool)
	storage := ports.ObjectStorage(st)
	pub := ports.EventPublisher(prod)
	if cfg.CircuitBreakerEnabled {
		repo = breaker.WrapRepository(repo, breaker.ForPostgres())
		storage = breaker.WrapStorage(storage, breaker.ForMinio())
		pub = breaker.WrapPublisher(pub, breaker.ForKafka())
	}
	svc := avatar.New(repo, storage, pub, nil, cfg.MaxUploadBytes)

	avh := handlers.NewAvatarHTTP(svc, cfg.MaxUploadBytes, cfg.PublicBaseURL)
	health := &handlers.Health{
		DB:    db.Pool,
		Minio: ports.Pinger(st),
		KafkaPing: func() error {
			return kafka.Ping(config.KafkaBrokerList(cfg.Kafka.Brokers))
		},
	}
	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: otelhttp.NewHandler(httpserver.NewRouter(httpserver.Deps{
			Log:               log,
			Service:           cfg.Telemetry.ServiceName,
			Avatar:            avh,
			Health:            health,
			RateLimitRequests: cfg.RateLimitRequests,
			RateLimitWindow:   cfg.RateLimitWindow,
		}), "http.server"),
		ReadHeaderTimeout: httpReadHeaderTimeout,
		ReadTimeout:       httpReadWriteTimeout,
		WriteTimeout:      httpReadWriteTimeout,
	}

	return &Profiled{
		cfg:           cfg,
		log:           log,
		db:            db,
		prod:          prod,
		srv:           srv,
		traceShutdown: traceShutdown,
	}, nil
}

// Logger возвращает логгер приложения.
func (a *Profiled) Logger() *slog.Logger {
	return a.log
}

// Run слушает сигналы, останавливает HTTP и закрывает ресурсы.
func (a *Profiled) Run(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(profiledDBPoolObserveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := a.db.Pool.Stat()
				observability.ObserveDBPool(a.cfg.Telemetry.ServiceName, stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns())
			}
		}
	}()

	go func() {
		a.log.Info("listen", "addr", a.srv.Addr)
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.log.Error("http", "error", err)
		}
	}()

	<-ctx.Done()
	a.log.Info("shutting down profiled")
	stats := a.db.Pool.Stat()
	observability.ObserveDBPool(a.cfg.Telemetry.ServiceName, stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()
	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		a.log.Error("http shutdown", "error", err)
	}

	if err := a.prod.Close(); err != nil {
		a.log.Error("kafka producer close", "error", err)
	}
	if a.traceShutdown != nil {
		if err := a.traceShutdown(shutdownCtx); err != nil {
			a.log.Error("trace shutdown", "error", err)
		}
	}

	a.db.Close()
	return nil
}
