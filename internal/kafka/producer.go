package kafka

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/pkg/retry"
	"github.com/rs/zerolog"
)

const (
	producerRetryInitialDelay = 50 * time.Millisecond
	producerRetryMaxDelay     = 8 * time.Second
	producerRetrySteps        = 8
)

var producerPublishRetry = retry.NewStrategy(
	retry.ExponentialBackoffDelays(producerRetryInitialDelay, producerRetryMaxDelay, producerRetrySteps),
	IsProducerKafkaRetryable,
)

// Producer обертка синхронного producer с именами топиков.
type Producer struct {
	p        sarama.SyncProducer
	topicUp  string
	topicDel string
	log      zerolog.Logger
}

// NewProducer создает синхронный producer Kafka.
func NewProducer(brokers []string, topicUpload, topicDelete string, log zerolog.Logger) (*Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForLocal
	cfg.Version = sarama.V2_8_0_0
	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}
	return &Producer{p: p, topicUp: topicUpload, topicDel: topicDelete, log: log}, nil
}

// PublishUpload публикует событие загрузки с ключом avatar_id.
func (p *Producer) PublishUpload(ctx context.Context, e ports.AvatarUploadEvent) error {
	b, err := MarshalUploadEvent(e)
	if err != nil {
		return err
	}
	msg := &sarama.ProducerMessage{
		Topic: p.topicUp,
		Key:   sarama.StringEncoder(e.AvatarID),
		Value: sarama.ByteEncoder(b),
	}
	return producerPublishRetry.DoWithRetry(ctx, func(ctx context.Context) error {
		_, _, err := p.p.SendMessage(msg)
		if err != nil {
			return err
		}
		p.log.Info().
			Str("topic", p.topicUp).
			Str("avatar_id", e.AvatarID).
			Msg("kafka publish upload")
		return nil
	})
}

// PublishDelete публикует событие удаления.
func (p *Producer) PublishDelete(ctx context.Context, e ports.AvatarDeleteEvent) error {
	b, err := MarshalDeleteEvent(e)
	if err != nil {
		return err
	}
	msg := &sarama.ProducerMessage{
		Topic: p.topicDel,
		Key:   sarama.StringEncoder(e.AvatarID),
		Value: sarama.ByteEncoder(b),
	}
	return producerPublishRetry.DoWithRetry(ctx, func(ctx context.Context) error {
		_, _, err := p.p.SendMessage(msg)
		if err != nil {
			return err
		}
		p.log.Info().
			Str("topic", p.topicDel).
			Str("avatar_id", e.AvatarID).
			Int("s3_keys", len(e.S3Keys)).
			Msg("kafka publish delete")
		return nil
	})
}

// Close завершает работу producer.
func (p *Producer) Close() error {
	return p.p.Close()
}

// Ping проверяет доступность брокера через метаданные клиента.
func Ping(brokers []string) error {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_0_0
	cl, err := sarama.NewClient(brokers, cfg)
	if err != nil {
		return err
	}
	return cl.Close()
}
