package kafka

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/arvaliullin/goph-profile/internal/pkg/retry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// ErrUnknownTopic возвращается из dispatchMessage для неизвестного топика.
var ErrUnknownTopic = errors.New("kafka: unknown topic")

const (
	consumerRetryInitialDelay = 200 * time.Millisecond
	consumerRetryMaxDelay     = 15 * time.Second
	consumerRetrySteps        = 6
)

var consumerHandlerRetry = retry.NewStrategy(
	retry.ExponentialBackoffDelays(consumerRetryInitialDelay, consumerRetryMaxDelay, consumerRetrySteps),
	isConsumerHandlerRetryable,
)

// claimHandler маршрутизирует сообщения consumer group по топикам.
type claimHandler struct {
	cfg     Config
	handler ports.GroupHandler
	log     *slog.Logger
}

// Config имена топиков для маршрутизации.
type Config struct {
	TopicUpload     string
	TopicDelete     string
	MaxMessageBytes int
}

func (h *claimHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *claimHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *claimHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		ctx := otel.GetTextMapPropagator().Extract(sess.Context(), newConsumerHeaderCarrier(msg))
		ctx, span := otel.Tracer("kafka-consumer").Start(ctx, "kafka.consume")
		span.SetAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", msg.Topic),
			attribute.Int64("messaging.kafka.offset", msg.Offset),
			attribute.Int64("messaging.kafka.partition", int64(msg.Partition)),
		)
		err := consumerHandlerRetry.DoWithRetry(ctx, func(ctx context.Context) error {
			return dispatchMessage(msg.Topic, h.cfg, h.handler, ctx, msg.Value)
		})
		if err != nil {
			span.RecordError(err)
			observability.ObserveKafkaConsume("avatard", msg.Topic, "error")
			span.End()
			return err
		}
		sess.MarkMessage(msg, "")
		observability.ObserveKafkaConsume("avatard", msg.Topic, "success")
		observability.LoggerWithTrace(ctx, h.log).InfoContext(ctx, "kafka consumer offset committed",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
		)
		span.End()
	}
	return nil
}

// dispatchMessage вызывает обработчик по имени топика.
func dispatchMessage(topic string, cfg Config, gh ports.GroupHandler, ctx context.Context, value []byte) error {
	switch topic {
	case cfg.TopicUpload:
		return gh.OnUpload(ctx, value)
	case cfg.TopicDelete:
		return gh.OnDelete(ctx, value)
	default:
		return ErrUnknownTopic
	}
}

// RunConsumerGroup запускает блокирующий цикл чтения до отмены ctx.
func RunConsumerGroup(ctx context.Context, brokers []string, group string, cfg Config, gh ports.GroupHandler, log *slog.Logger) error {
	sconfig := sarama.NewConfig()
	sconfig.Version = sarama.V2_8_0_0
	sconfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	sconfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	sconfig.Consumer.Fetch.Max = int32(cfg.MaxMessageBytes)

	g, err := sarama.NewConsumerGroup(brokers, group, sconfig)
	if err != nil {
		return err
	}
	defer g.Close()

	topics := []string{cfg.TopicUpload, cfg.TopicDelete}
	h := &claimHandler{cfg: cfg, handler: gh, log: log}

	for {
		if err := g.Consume(ctx, topics, h); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}
