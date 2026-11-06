package chat

import (
	"errors"
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
	switch {
	case errors.Is(err, chusecase.ErrUnauthorized):
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case errors.Is(err, chusecase.ErrInvalidCSRF):
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
	case errors.Is(err, chusecase.ErrChatNotFound):
		helpers.WriteError(w, http.StatusNotFound, "chat_not_found", "chat not found")
	case errors.Is(err, chusecase.ErrNotParticipant):
		helpers.WriteError(w, http.StatusForbidden, "not_chat_participant", "user is not a chat participant")
	default:
		helpers.WriteInternalError(w, "chat", err)
	}
}
