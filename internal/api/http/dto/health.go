package dto

// ComponentStatus состояние одной зависимости в health check.
type ComponentStatus struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// HealthResponse ответ GET /health.
type HealthResponse struct {
	Postgres ComponentStatus `json:"postgres"`
	Minio    ComponentStatus `json:"minio"`
	Kafka    ComponentStatus `json:"kafka"`
}
