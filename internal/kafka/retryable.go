package kafka

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/IBM/sarama"
)

// IsProducerKafkaRetryable возвращает true для ошибок брокера и сети, при которых имеет смысл повторить SendMessage.
// Для ошибок без KError в цепочке (например обрыв TCP) возвращает true как для временных сбоев.
func IsProducerKafkaRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.EOF) {
		return false
	}
	ke, ok := errors.AsType[sarama.KError](err)
	if ok {
		switch ke {
		case sarama.ErrLeaderNotAvailable,
			sarama.ErrNotLeaderForPartition,
			sarama.ErrRequestTimedOut,
			sarama.ErrBrokerNotAvailable,
			sarama.ErrReplicaNotAvailable,
			sarama.ErrNetworkException,
			sarama.ErrNotEnoughReplicas,
			sarama.ErrNotEnoughReplicasAfterAppend,
			sarama.ErrUnknownTopicOrPartition,
			sarama.ErrOffsetsLoadInProgress,
			sarama.ErrNotCoordinatorForConsumer,
			sarama.ErrRebalanceInProgress,
			sarama.ErrUnknown:
			return true
		default:
			return false
		}
	}
	netErr, netOK := errors.AsType[net.Error](err)
	if netOK && netErr.Timeout() {
		return true
	}
	return false
}

// isConsumerHandlerRetryable задаёт, повторять ли вызов обработчика после ошибки.
func isConsumerHandlerRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, ErrUnknownTopic) {
		return false
	}
	return true
}
