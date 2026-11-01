package relationship

import (
	"errors"
	"net/http"

	"kvmm/internal/handler/helpers"
	relusecase "kvmm/internal/usecase/relationship"
)

type FriendsHandler struct {
	useCase *relusecase.UseCase
}

// NewRelationshipHandler создаёт обработчик связей
func NewRelationshipHandler(useCase *relusecase.UseCase) *FriendsHandler {
	return &FriendsHandler{useCase: useCase}
}

func (h *FriendsHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, relusecase.ErrUnauthorized):
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case errors.Is(err, relusecase.ErrInvalidCSRF):
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
	case errors.Is(err, relusecase.ErrConflict):
		helpers.WriteError(w, http.StatusConflict, "relationship_conflict", "profiles already have an active relationship")
	case errors.Is(err, relusecase.ErrNotFound):
		helpers.WriteError(w, http.StatusNotFound, "not_found", "relationship not found")
	case errors.Is(err, relusecase.ErrInvalid):
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "invalid relationship data")
	default:
		helpers.WriteInternalError(w, "relationship", err)
	}
}
