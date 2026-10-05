package domain

// AuthResponse — ответ после успешной регистрации/авторизации
type AuthResponse struct {
	Profile   Profile `json:"profile"`
	CSRFToken string  `json:"csrf_token" example:"a1b2c3d4e5f6"`
}

// MeResponse — GET /api/auth/me
type MeResponse struct {
	Profile   Profile `json:"profile"`
	CSRFToken string  `json:"csrf_token" example:"a1b2c3d4e5f6"`
}

// FeedResponse — ответ ленты публикаций
type FeedResponse struct {
	Posts      []Post `json:"posts"`
	Offset     int    `json:"offset" example:"10"`
	Limit      int    `json:"limit" example:"10"`
	HasMore    bool   `json:"has_more" example:"true"`
	NextOffset int    `json:"next_offset,omitempty" example:"20"`
}
