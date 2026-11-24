package community

import (
	"errors"
	"net/http"
	"strconv"

	"kvmm/internal/handler/helpers"
	communityusecase "kvmm/internal/usecase/community"
)

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, communityusecase.ErrUnauthorized):
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case errors.Is(err, communityusecase.ErrInvalidCSRF):
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
	case errors.Is(err, communityusecase.ErrCommunityForbidden):
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "insufficient community permissions")
	case errors.Is(err, communityusecase.ErrCommunityNotFound), errors.Is(err, communityusecase.ErrCommunityMemberNotFound):
		helpers.WriteError(w, http.StatusNotFound, "not_found", "community or membership not found")
	case errors.Is(err, communityusecase.ErrCommunityConflict):
		helpers.WriteError(w, http.StatusConflict, "conflict", "community name is already in use")
	case errors.Is(err, communityusecase.ErrProfileNotFound):
		helpers.WriteError(w, http.StatusNotFound, "profile_not_found", "profile not found")
	case errors.Is(err, communityusecase.ErrInvalidCommunity):
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		helpers.WriteInternalError(w, "community", err)
	}
}

func pagination(r *http.Request) (int, int, error) {
	offset, limit := 0, 10
	var err error
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return 0, 0, errors.New("offset must be a non-negative integer")
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 50 {
			return 0, 0, errors.New("limit must be between 1 and 50")
		}
	}
	return offset, limit, nil
}
