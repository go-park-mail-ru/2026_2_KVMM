package chat

import (
	"net/http"

	"kvmm/internal/handler/helpers"
	chusecase "kvmm/internal/usecase/chat"
	sessionusecase "kvmm/internal/usecase/session"
)

// ChatHandler обрабатывает запросы к чатам и сообщениям
type ChatHandler struct {
	useCase  *chusecase.UseCase
	sessions *sessionusecase.UseCase
	hub      *chatHub
}

// NewChatHandler создаёт обработчик чатов
func NewChatHandler(useCase *chusecase.UseCase, sessions *sessionusecase.UseCase) *ChatHandler {
	return &ChatHandler{useCase: useCase, sessions: sessions, hub: newChatHub()}
}

func (h *ChatHandler) writeError(w http.ResponseWriter, err error) {
	switch err {
	case chusecase.ErrUnauthorized:
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case chusecase.ErrInvalidCSRF:
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
	default:
		helpers.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
	}
}
