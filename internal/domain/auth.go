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

// RegisterRequest - запрос на регистрацию
type RegisterRequest struct {
	Nickname             string `json:"nickname" example:"ivan67" validate:"required,min=4,max=32"`
	Password             string `json:"password" example:"sjkdfhgs4536njbjjkh" validate:"required,min=8,max=64"`
	ConfirmPassword      string `json:"confirm_password" example:"sjkdfhgs4536njbjjkh" validate:"required,eqfield=Password"`
	PrivacyPolicyAllowed bool   `json:"privacy_policy_allowed" example:"true" validate:"required,eq=true"`
}

// LoginRequest — Login может быть nickname, email или phone_number
type LoginRequest struct {
	Login    string `json:"login" example:"ivan_petrov" validate:"required,max=254"`
	Password string `json:"password" example:"strongpassword123" validate:"required,min=8,max=64"`
}
