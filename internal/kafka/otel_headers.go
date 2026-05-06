package kafka

import (
	"github.com/IBM/sarama"
)

type producerHeaderCarrier struct {
	msg *sarama.ProducerMessage
}

func newProducerHeaderCarrier(msg *sarama.ProducerMessage) producerHeaderCarrier {
	return producerHeaderCarrier{msg: msg}
}

func (c producerHeaderCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c producerHeaderCarrier) Set(key, value string) {
	for i := range c.msg.Headers {
		if string(c.msg.Headers[i].Key) == key {
			c.msg.Headers[i].Value = []byte(value)
			return
		}
	}
	c.msg.Headers = append(c.msg.Headers, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	})
}

func (c producerHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.msg.Headers))
	for _, h := range c.msg.Headers {
		keys = append(keys, string(h.Key))
	}
	return keys
}

type consumerHeaderCarrier struct {
	msg *sarama.ConsumerMessage
}

func newConsumerHeaderCarrier(msg *sarama.ConsumerMessage) consumerHeaderCarrier {
	return consumerHeaderCarrier{msg: msg}
}

func (c consumerHeaderCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c consumerHeaderCarrier) Set(_, _ string) {}

func (c consumerHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.msg.Headers))
	for _, h := range c.msg.Headers {
		keys = append(keys, string(h.Key))
	}
	return keys
}
