package auth

import (
	"errors"
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	authusecase "kvmm/internal/usecase/auth"
)

// RequestRegisterCode отправляет код для регистрации нового пользователя.
// RequestRegisterCode godoc
// @Summary Отправка кода регистрации
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterEmailCodeRequest true "Email и согласие на обработку данных"
// @Success 200 {object} domain.EmailCodeResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Failure 503 {object} domain.ErrorResponse
// @Router /api/auth/register/email/send/code [post]
func (h *AuthHandler) RequestRegisterCode(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.RegisterEmailCodeRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := helpers.ValidateEmailAddress(req.Email); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := h.useCase.RequestRegisterEmailCode(r.Context(), req.Email); err != nil {
		h.writeEmailError(w, err, http.StatusBadRequest)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, domain.EmailCodeResponse{Message: "verification code sent"})
}

// RegisterByEmail подтверждает email-код и регистрирует пользователя.
// RegisterByEmail godoc
// @Summary Регистрация по email-коду
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterByEmailRequest true "Email и код"
// @Success 201 {object} domain.AuthResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/auth/register/email/code [post]
func (h *AuthHandler) RegisterByEmail(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.RegisterByEmailRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := helpers.ValidateEmailAddress(req.Email); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	response, session, err := h.useCase.RegisterByEmail(r.Context(), req)
	if err != nil {
		h.writeEmailError(w, err, http.StatusBadRequest)
		return
	}
	helpers.SetSessionCookie(w, session)
	helpers.WriteJSON(w, http.StatusCreated, response)
}

// RequestLoginCode отправляет код для входа по email.
// RequestLoginCode godoc
// @Summary Отправка кода входа
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.EmailCodeRequest true "Email"
// @Success 200 {object} domain.EmailCodeResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 503 {object} domain.ErrorResponse
// @Router /api/auth/login/email/send/code [post]
func (h *AuthHandler) RequestLoginCode(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.EmailCodeRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := helpers.ValidateEmailAddress(req.Email); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := h.useCase.RequestLoginEmailCode(r.Context(), req.Email); err != nil {
		h.writeEmailError(w, err, http.StatusUnauthorized)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, domain.EmailCodeResponse{Message: "verification code sent"})
}

// LoginByEmail авторизует пользователя одноразовым кодом из email.
// LoginByEmail godoc
// @Summary Авторизация по email-коду
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.LoginByEmailRequest true "Email и код"
// @Success 200 {object} domain.AuthResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/auth/login/email/code [post]
func (h *AuthHandler) LoginByEmail(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.LoginByEmailRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := helpers.ValidateEmailAddress(req.Email); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	response, session, err := h.useCase.LoginByEmail(r.Context(), req)
	if err != nil {
		h.writeEmailError(w, err, http.StatusUnauthorized)
		return
	}
	helpers.SetSessionCookie(w, session)
	helpers.WriteJSON(w, http.StatusOK, response)
}

// RequestEmailChange отправляет код на новый email текущего пользователя.
// RequestEmailChange godoc
// @Summary Запрос на смену email
// @Tags auth
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.ChangeEmailRequest true "Новый email"
// @Success 200 {object} domain.EmailCodeResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Failure 503 {object} domain.ErrorResponse
// @Router /api/auth/email/change [post]
func (h *AuthHandler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.ChangeEmailRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := helpers.ValidateEmailAddress(req.Email); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := h.useCase.RequestEmailChange(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), req.Email); err != nil {
		h.writeEmailError(w, err, http.StatusBadRequest)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, domain.EmailCodeResponse{Message: "verification code sent"})
}

// ConfirmEmail подтверждает новый email текущего пользователя.
// ConfirmEmail godoc
// @Summary Подтверждение email
// @Tags auth
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.ConfirmEmailRequest true "Email и код подтверждения"
// @Success 200 {object} domain.Profile
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Router /api/auth/email/confirm [post]
func (h *AuthHandler) ConfirmEmail(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.ConfirmEmailRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := helpers.ValidateEmailAddress(req.Email); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	profile, err := h.useCase.ConfirmEmailChange(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), req)
	if err != nil {
		h.writeEmailError(w, err, http.StatusBadRequest)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, profile)
}

func (h *AuthHandler) writeEmailError(w http.ResponseWriter, err error, invalidStatus int) {
	switch {
	case errors.Is(err, authusecase.ErrConflict):
		helpers.WriteError(w, http.StatusConflict, "already_exists", "email is already registered")
	case errors.Is(err, authusecase.ErrInvalidEmailCode), errors.Is(err, authusecase.ErrInvalidCredentials):
		helpers.WriteError(w, invalidStatus, "invalid_code_or_request", "invalid or expired email code")
	case errors.Is(err, authusecase.ErrUnauthorized):
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case errors.Is(err, authusecase.ErrInvalidCSRF):
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
	case errors.Is(err, authusecase.ErrEmailDelivery):
		helpers.WriteError(w, http.StatusServiceUnavailable, "email_delivery_failed", "could not send verification email")
	default:
		helpers.WriteInternalError(w, "auth.email", err)
	}
}
