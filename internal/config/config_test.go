package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKafkaBrokerList(t *testing.T) {
	t.Parallel()
	l := KafkaBrokerList("a:9092, b:9093 , ")
	require.Equal(t, []string{"a:9092", "b:9093"}, l)
}

func TestLoadWorker_MinimalEnv(t *testing.T) {
	t.Setenv("DATABASE_URI", "postgres://x/x")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "k")
	t.Setenv("MINIO_SECRET_KEY", "s")
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	w, err := LoadWorker()
	require.NoError(t, err)
	require.Equal(t, "avatars.upload", w.Kafka.TopicUpload)
	require.Equal(t, "avatard", w.Telemetry.ServiceName)
	require.Equal(t, 1<<20, w.Kafka.MaxMessageBytes)
}
