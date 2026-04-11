package kafka

import (
	"context"
	"errors"
	"time"

	"github.com/IBM/sarama"
	"github.com/arvaliullin/goph-profile/internal/pkg/retry"
)

// ErrUnknownTopic возвращается из dispatchMessage для неизвестного топика (повторять бессмысленно).
var ErrUnknownTopic = errors.New("kafka: unknown topic")

var consumerHandlerRetry = retry.NewStrategy(
	retry.ExponentialBackoffDelays(200*time.Millisecond, 15*time.Second, 6),
	isConsumerHandlerRetryable,
)

// GroupHandler обрабатывает полезную нагрузку сообщений.
type GroupHandler interface {
	OnUpload(ctx context.Context, payload []byte) error
	OnDelete(ctx context.Context, payload []byte) error
}

type claimHandler struct {
	cfg     Config
	handler GroupHandler
}

// Config имена топиков для маршрутизации.
type Config struct {
	TopicUpload string
	TopicDelete string
}

func (h *claimHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *claimHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *claimHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		ctx := sess.Context()
		err := consumerHandlerRetry.DoWithRetry(ctx, func(ctx context.Context) error {
			return dispatchMessage(msg.Topic, h.cfg, h.handler, ctx, msg.Value)
		})
		if err != nil {
			return err
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}

func dispatchMessage(topic string, cfg Config, gh GroupHandler, ctx context.Context, value []byte) error {
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
func RunConsumerGroup(ctx context.Context, brokers []string, group string, cfg Config, gh GroupHandler) error {
	sconfig := sarama.NewConfig()
	sconfig.Version = sarama.V2_8_0_0
	sconfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	sconfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	g, err := sarama.NewConsumerGroup(brokers, group, sconfig)
	if err != nil {
		return err
	}
	defer func() { _ = g.Close() }()

	topics := []string{cfg.TopicUpload, cfg.TopicDelete}
	h := &claimHandler{cfg: cfg, handler: gh}

	for {
		if err := g.Consume(ctx, topics, h); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}
