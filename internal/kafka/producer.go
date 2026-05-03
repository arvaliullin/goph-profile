package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/arvaliullin/goph-profile/internal/pkg/retry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	log      *slog.Logger
}

// NewProducer создает синхронный producer Kafka.
func NewProducer(brokers []string, topicUpload, topicDelete string, maxMessageBytes int, log *slog.Logger) (*Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForLocal
	cfg.Producer.MaxMessageBytes = maxMessageBytes
	cfg.Version = sarama.V2_8_0_0
	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}
	return &Producer{p: p, topicUp: topicUpload, topicDel: topicDelete, log: log}, nil
}

// PublishUpload публикует событие загрузки с ключом avatar_id.
func (p *Producer) PublishUpload(ctx context.Context, e ports.AvatarUploadEvent) error {
	ctx, span := otel.Tracer("kafka-producer").Start(ctx, "kafka.publish.upload")
	span.SetAttributes(
		attribute.String("messaging.system", "kafka"),
		attribute.String("messaging.destination", p.topicUp),
		attribute.String("avatar.id", e.AvatarID),
		attribute.String("user.id", e.UserID),
	)
	defer span.End()

	b, err := MarshalUploadEvent(e)
	if err != nil {
		span.RecordError(err)
		observability.ObserveKafkaPublish("profiled", p.topicUp, "error")
		return err
	}
	msg := &sarama.ProducerMessage{
		Topic: p.topicUp,
		Key:   sarama.StringEncoder(e.AvatarID),
		Value: sarama.ByteEncoder(b),
	}
	carrier := newProducerHeaderCarrier(msg)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return producerPublishRetry.DoWithRetry(ctx, func(ctx context.Context) error {
		_, _, err := p.p.SendMessage(msg)
		if err != nil {
			span.RecordError(err)
			observability.ObserveKafkaPublish("profiled", p.topicUp, "error")
			return err
		}
		observability.LoggerWithTrace(ctx, p.log).InfoContext(ctx, "kafka publish upload",
			"topic", p.topicUp,
			"avatar_id", e.AvatarID,
		)
		observability.ObserveKafkaPublish("profiled", p.topicUp, "success")
		return nil
	})
}

// PublishDelete публикует событие удаления.
func (p *Producer) PublishDelete(ctx context.Context, e ports.AvatarDeleteEvent) error {
	ctx, span := otel.Tracer("kafka-producer").Start(ctx, "kafka.publish.delete")
	span.SetAttributes(
		attribute.String("messaging.system", "kafka"),
		attribute.String("messaging.destination", p.topicDel),
		attribute.String("avatar.id", e.AvatarID),
		attribute.Int("s3.keys", len(e.S3Keys)),
	)
	defer span.End()

	b, err := MarshalDeleteEvent(e)
	if err != nil {
		span.RecordError(err)
		observability.ObserveKafkaPublish("profiled", p.topicDel, "error")
		return err
	}
	msg := &sarama.ProducerMessage{
		Topic: p.topicDel,
		Key:   sarama.StringEncoder(e.AvatarID),
		Value: sarama.ByteEncoder(b),
	}
	carrier := newProducerHeaderCarrier(msg)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return producerPublishRetry.DoWithRetry(ctx, func(ctx context.Context) error {
		_, _, err := p.p.SendMessage(msg)
		if err != nil {
			span.RecordError(err)
			observability.ObserveKafkaPublish("profiled", p.topicDel, "error")
			return err
		}
		observability.LoggerWithTrace(ctx, p.log).InfoContext(ctx, "kafka publish delete",
			"topic", p.topicDel,
			"avatar_id", e.AvatarID,
			"s3_keys", len(e.S3Keys),
		)
		observability.ObserveKafkaPublish("profiled", p.topicDel, "success")
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
