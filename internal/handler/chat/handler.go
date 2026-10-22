package chat

import (
	"net/http"

	"kvmm/internal/handler/helpers"
	chusecase "kvmm/internal/usecase/chat"
)

// ChatHandler обрабатывает запросы к чатам и сообщениям
type ChatHandler struct {
	useCase *chusecase.UseCase
}

// NewChatHandler создаёт обработчик чатов
func NewChatHandler(useCase *chusecase.UseCase) *ChatHandler {
	return &ChatHandler{useCase: useCase}
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
