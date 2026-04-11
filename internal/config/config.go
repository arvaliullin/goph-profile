package config

import (
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

const defaultMaxUpload = 10 << 20 // 10 МиБ

// Server настройки HTTP и интеграций для profiled.
type Server struct {
	HTTPAddr        string        `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURI     string        `env:"DATABASE_URI,required"`
	MinioEndpoint   string        `env:"MINIO_ENDPOINT,required"`
	MinioAccessKey  string        `env:"MINIO_ACCESS_KEY,required"`
	MinioSecretKey  string        `env:"MINIO_SECRET_KEY,required"`
	MinioSecure     bool          `env:"MINIO_SECURE" envDefault:"false"`
	MinioBucket     string        `env:"MINIO_BUCKET" envDefault:"avatars"`
	KafkaBrokers    string        `env:"KAFKA_BROKERS,required"`
	KafkaTopicUp    string        `env:"KAFKA_TOPIC_UPLOAD" envDefault:"avatars.upload"`
	KafkaTopicDel   string        `env:"KAFKA_TOPIC_DELETE" envDefault:"avatars.delete"`
	MaxUploadBytes  int64         `env:"MAX_UPLOAD_BYTES" envDefault:"10485760"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	PublicBaseURL   string        `env:"PUBLIC_BASE_URL" envDefault:""`
}

// Worker настройки для avatard.
type Worker struct {
	DatabaseURI     string        `env:"DATABASE_URI,required"`
	MinioEndpoint   string        `env:"MINIO_ENDPOINT,required"`
	MinioAccessKey  string        `env:"MINIO_ACCESS_KEY,required"`
	MinioSecretKey  string        `env:"MINIO_SECRET_KEY,required"`
	MinioSecure     bool          `env:"MINIO_SECURE" envDefault:"false"`
	MinioBucket     string        `env:"MINIO_BUCKET" envDefault:"avatars"`
	KafkaBrokers    string        `env:"KAFKA_BROKERS,required"`
	KafkaTopicUp    string        `env:"KAFKA_TOPIC_UPLOAD" envDefault:"avatars.upload"`
	KafkaTopicDel   string        `env:"KAFKA_TOPIC_DELETE" envDefault:"avatars.delete"`
	KafkaGroup      string        `env:"KAFKA_GROUP" envDefault:"avatars-worker"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`
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
	return &c, nil
}

// LoadWorker читает переменные окружения в Worker.
func LoadWorker() (*Worker, error) {
	var c Worker
	if err := env.Parse(&c); err != nil {
		return nil, err
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
