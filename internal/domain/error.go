package domain

// ErrorResponse — единый формат ошибки API
type ErrorResponse struct {
	Code    string `json:"code" examples:"invalid_credentials"`
	Message string `json:"message" examples:"Invalid credentials"`
}
