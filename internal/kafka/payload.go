package kafka

import (
	"encoding/json"

	"github.com/arvaliullin/goph-profile/internal/core/ports"
)

// MarshalUploadEvent сериализует событие загрузки аватара в JSON тела сообщения Kafka.
func MarshalUploadEvent(e ports.AvatarUploadEvent) ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalUploadEvent восстанавливает событие загрузки из JSON тела сообщения Kafka.
func UnmarshalUploadEvent(b []byte) (ports.AvatarUploadEvent, error) {
	var e ports.AvatarUploadEvent
	err := json.Unmarshal(b, &e)
	return e, err
}

// MarshalDeleteEvent сериализует событие удаления аватара в JSON тела сообщения Kafka.
func MarshalDeleteEvent(e ports.AvatarDeleteEvent) ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalDeleteEvent восстанавливает событие удаления из JSON тела сообщения Kafka.
func UnmarshalDeleteEvent(b []byte) (ports.AvatarDeleteEvent, error) {
	var e ports.AvatarDeleteEvent
	err := json.Unmarshal(b, &e)
	return e, err
}
