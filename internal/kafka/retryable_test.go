package kafka

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
)

func TestIsProducerKafkaRetryable(t *testing.T) {
	t.Parallel()
	assert.False(t, IsProducerKafkaRetryable(nil))
	assert.False(t, IsProducerKafkaRetryable(context.Canceled))
	assert.True(t, IsProducerKafkaRetryable(sarama.ErrLeaderNotAvailable))
	assert.True(t, IsProducerKafkaRetryable(sarama.ErrRequestTimedOut))
	assert.False(t, IsProducerKafkaRetryable(sarama.ErrInvalidMessage))
	assert.False(t, IsProducerKafkaRetryable(errors.New("logical")))
	assert.False(t, IsProducerKafkaRetryable(io.EOF))
}

func TestIsProducerKafkaRetryable_netTimeout(t *testing.T) {
	t.Parallel()
	err := &net.OpError{Err: &timeoutError{}}
	assert.True(t, IsProducerKafkaRetryable(err))
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return false }

func TestIsConsumerHandlerRetryable(t *testing.T) {
	t.Parallel()
	assert.False(t, isConsumerHandlerRetryable(nil))
	assert.False(t, isConsumerHandlerRetryable(context.Canceled))
	assert.False(t, isConsumerHandlerRetryable(ErrUnknownTopic))
	assert.True(t, isConsumerHandlerRetryable(errors.New("transient")))
}
