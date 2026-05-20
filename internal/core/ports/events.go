package ports

// AvatarUploadEvent событие после сохранения оригинала в объектном хранилище.
type AvatarUploadEvent struct {
	AvatarID string `json:"avatar_id"`
	UserID   string `json:"user_id"`
	S3Key    string `json:"s3_key"`
}

// AvatarDeleteEvent событие асинхронного удаления объектов из объектного хранилища.
type AvatarDeleteEvent struct {
	AvatarID string   `json:"avatar_id"`
	S3Keys   []string `json:"s3_keys"`
}
