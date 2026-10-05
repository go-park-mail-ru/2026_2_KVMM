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
