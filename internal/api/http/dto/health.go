package dto

// ComponentStatus состояние одной зависимости в сводной проверке готовности.
type ComponentStatus struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// HealthResponse ответ GET /health.
type HealthResponse struct {
	Postgres *ComponentStatus `json:"postgres,omitempty"`
	Minio    *ComponentStatus `json:"minio,omitempty"`
	Kafka    *ComponentStatus `json:"kafka,omitempty"`
}
