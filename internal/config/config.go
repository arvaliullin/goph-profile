package config

import (
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

const (
	defaultMaxUpload            = 10 << 20 // 10 МиБ
	defaultKafkaMaxMessageBytes = 1 << 20
)

type minioConfig struct {
	Endpoint  string `env:"MINIO_ENDPOINT,required"`
	AccessKey string `env:"MINIO_ACCESS_KEY,required"`
	SecretKey string `env:"MINIO_SECRET_KEY,required"`
	Secure    bool   `env:"MINIO_SECURE" envDefault:"false"`
	Bucket    string `env:"MINIO_BUCKET" envDefault:"avatars"`
}

type kafkaConfig struct {
	Brokers         string `env:"KAFKA_BROKERS,required"`
	TopicUpload     string `env:"KAFKA_TOPIC_UPLOAD" envDefault:"avatars.upload"`
	TopicDelete     string `env:"KAFKA_TOPIC_DELETE" envDefault:"avatars.delete"`
	MaxMessageBytes int    `env:"KAFKA_MAX_MESSAGE_BYTES" envDefault:"1048576"`
}

type workerKafkaConfig struct {
	Brokers         string `env:"KAFKA_BROKERS,required"`
	TopicUpload     string `env:"KAFKA_TOPIC_UPLOAD" envDefault:"avatars.upload"`
	TopicDelete     string `env:"KAFKA_TOPIC_DELETE" envDefault:"avatars.delete"`
	Group           string `env:"KAFKA_GROUP" envDefault:"avatars-worker"`
	MaxMessageBytes int    `env:"KAFKA_MAX_MESSAGE_BYTES" envDefault:"1048576"`
}

type telemetryConfig struct {
	OTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:""`
	Environment  string `env:"OTEL_ENVIRONMENT" envDefault:"local"`
	ServiceName  string `env:"OTEL_SERVICE_NAME" envDefault:""`
}

// Server настройки HTTP и интеграций для profiled.
type Server struct {
	HTTPAddr              string        `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURI           string        `env:"DATABASE_URI,required"`
	MaxUploadBytes        int64         `env:"MAX_UPLOAD_BYTES" envDefault:"10485760"`
	ShutdownTimeout       time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	PublicBaseURL         string        `env:"PUBLIC_BASE_URL" envDefault:""`
	LogLevel              string        `env:"LOG_LEVEL" envDefault:"info"`
	CircuitBreakerEnabled bool          `env:"CIRCUIT_BREAKER_ENABLED" envDefault:"true"`
	RateLimitRequests     int           `env:"RATE_LIMIT_REQUESTS" envDefault:"60"`
	RateLimitWindow       time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
	Minio                 minioConfig
	Kafka                 kafkaConfig
	Telemetry             telemetryConfig
}

// Worker настройки для avatard.
type Worker struct {
	DatabaseURI           string        `env:"DATABASE_URI,required"`
	ShutdownTimeout       time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`
	MetricsAddr           string        `env:"METRICS_ADDR" envDefault:":9091"`
	LogLevel              string        `env:"LOG_LEVEL" envDefault:"info"`
	CircuitBreakerEnabled bool          `env:"CIRCUIT_BREAKER_ENABLED" envDefault:"true"`
	Minio                 minioConfig
	Kafka                 workerKafkaConfig
	Telemetry             telemetryConfig
}

// LoadServer читает переменные окружения в Server.
func LoadServer() (*Server, error) {
	var c Server
	if err := env.Parse(&c); err != nil {
		return nil, err
	}
	if c.MaxUploadBytes <= 0 {
		c.MaxUploadBytes = defaultMaxUpload
	}
	if c.Kafka.MaxMessageBytes <= 0 {
		c.Kafka.MaxMessageBytes = defaultKafkaMaxMessageBytes
	}
	if c.Telemetry.ServiceName == "" {
		c.Telemetry.ServiceName = "profiled"
	}
	return &c, nil
}

// LoadWorker читает переменные окружения в Worker.
func LoadWorker() (*Worker, error) {
	var c Worker
	if err := env.Parse(&c); err != nil {
		return nil, err
	}
	if c.Kafka.MaxMessageBytes <= 0 {
		c.Kafka.MaxMessageBytes = defaultKafkaMaxMessageBytes
	}
	if c.Telemetry.ServiceName == "" {
		c.Telemetry.ServiceName = "avatard"
	}
	return &c, nil
}

// KafkaBrokerList разбивает строку брокеров по запятым.
func KafkaBrokerList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
