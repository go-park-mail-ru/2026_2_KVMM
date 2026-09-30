package domain

// AuthResponse — ответ после успешной регистрации/авторизации
type AuthResponse struct {
	Profile   Profile `json:"profile"`
	CSRFToken string  `json:"csrf_token" examples:"a1b2c3d4e5f6"`
}

// MeResponse — GET /api/auth/me
type MeResponse struct {
	Profile   Profile `json:"profile"`
	CSRFToken string  `json:"csrf_token" examples:"a1b2c3d4e5f6"`
}

// FeedResponse — ответ ленты публикаций
type FeedResponse struct {
	Posts      []Post `json:"posts"`
	Offset     int    `json:"offset" examples:"10"`
	Limit      int    `json:"limit" examples:"10"`
	HasMore    bool   `json:"has_more" examples:"true"`
	NextOffset int    `json:"next_offset,omitempty" examples:"20"`
}
