package auth

import (
	"errors"
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/mailer"
	"kvmm/internal/store"
)

// RequestRegisterCode отправляет код для регистрации нового пользователя
// RequestRegisterCode godoc
// @Summary Отправка кода регистрации
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterEmailCodeRequest true "Email и согласие на обработку данных"
// @Success 200 {object} domain.EmailCodeResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
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
	if store.EmailExists(req.Email) {
		helpers.WriteError(w, http.StatusConflict, "already_exists", "email is already registered")
		return
	}
	sendEmailCode(w, req.Email, store.EmailCodeRegister)
}

// RegisterByEmail подтверждает email-код и регистрирует пользователя
// RegisterByEmail godoc
// @Summary Регистрация по email-коду
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterByEmailRequest true "Данные регистрации и код"
// @Success 201 {object} domain.AuthResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
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
	if err := helpers.ValidateRegisterByEmailRequest(req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	profile, session, csrfToken, err := store.RegisterByEmail(req)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			helpers.WriteError(w, http.StatusConflict, "already_exists", "nickname, email or phone_number is already used")
			return
		}
		helpers.WriteError(w, http.StatusBadRequest, "invalid_code_or_request", "invalid or expired email code")
		return
	}
	setSessionCookie(w, session)
	helpers.WriteJSON(w, http.StatusCreated, domain.AuthResponse{Profile: profile, CSRFToken: csrfToken})
}

// RequestLoginCode отправляет код для входа по email
// RequestLoginCode godoc
// @Summary Отправка кода входа
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.EmailCodeRequest true "Email"
// @Success 200 {object} domain.EmailCodeResponse
// @Failure 400 {object} domain.ErrorResponse
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
	if !store.EmailExists(req.Email) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "email not already exists")
		return
	}
	sendEmailCode(w, req.Email, store.EmailCodeLogin)
}

// LoginByEmail авторизует пользователя одноразовым кодом из email
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
	profile, session, csrfToken, err := store.LoginByEmail(req)
	if err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or code")
		return
	}
	setSessionCookie(w, session)
	helpers.WriteJSON(w, http.StatusOK, domain.AuthResponse{Profile: profile, CSRFToken: csrfToken})
}

// sendEmailCode создаёт код, передаёт его mailer и возвращает единый ответ
func sendEmailCode(w http.ResponseWriter, email, purpose string) {
	if err := deliverEmailCode(email, purpose); err != nil {
		helpers.WriteError(w, http.StatusServiceUnavailable, "email_delivery_failed", err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusOK, domain.EmailCodeResponse{Message: "verification code sent"})
}

// deliverEmailCode создаёт код и передаёт его mailer
func deliverEmailCode(email, purpose string) error {
	code, err := store.CreateEmailCode(email, purpose)
	if err != nil {
		return err
	}
	if err := mailer.SendAuthCode(email, code); err != nil {
		store.CancelEmailCode(email, purpose)
		return err
	}
	return nil
}
