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

// EmailCodeRequest — запрос на отправку кода на email
type EmailCodeRequest struct {
	Email string `json:"email" example:"ivan@example.com" validate:"required,max=254"`
}

// EmailCodeResponse — ответ после отправки кода на email
type EmailCodeResponse struct {
	Message string `json:"message" example:"verification code sent"`
}

// RegisterEmailCodeRequest — запрос на отправку кода для регистрации
type RegisterEmailCodeRequest struct {
	Email                string `json:"email" example:"ivan@example.com" validate:"required,max=254"`
	PrivacyPolicyAllowed bool   `json:"privacy_policy_allowed" example:"true" validate:"required,eq=true"`
}

// RegisterRequest - запрос на регистрацию
type RegisterRequest struct {
	Nickname             string `json:"nickname" example:"ivan67" validate:"required,min=4,max=32"`
	Password             string `json:"password" example:"sjkdfhgs4536njbjjkh" validate:"required,min=8,max=64"`
	ConfirmPassword      string `json:"confirm_password" example:"sjkdfhgs4536njbjjkh" validate:"required,eqfield=Password"`
	PrivacyPolicyAllowed bool   `json:"privacy_policy_allowed" example:"true" validate:"required,eq=true"`
}

// RegisterByEmailRequest — запрос на завершение регистрации по email-коду
type RegisterByEmailRequest struct {
	Email string `json:"email" example:"ivan@example.com" validate:"required,max=254"`
	Code  string `json:"code" example:"123456" validate:"required,len=6,numeric"`
}

// LoginByEmailRequest — запрос на авторизацию по email-коду
type LoginByEmailRequest struct {
	Email string `json:"email" example:"ivan@example.com" validate:"required,max=254"`
	Code  string `json:"code" example:"123456" validate:"required,len=6,numeric"`
}

// LoginRequest — Login может быть nickname, email или phone_number
type LoginRequest struct {
	Login    string `json:"login" example:"ivan_petrov" validate:"required,max=254"`
	Password string `json:"password" example:"strongpassword123" validate:"required,min=8,max=64"`
}
