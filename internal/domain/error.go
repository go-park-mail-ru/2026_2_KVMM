package domain

// ErrorResponse — единый формат ошибки API.
type ErrorResponse struct {
	Code    string `json:"code" example:"invalid_credentials"`
	Message string `json:"message" example:"Invalid credentials"`
}
