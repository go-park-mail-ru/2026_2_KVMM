package posts

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

func (h *PostHandler) Runtime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	values, err := h.useCase.Runtime(helpers.SessionCookie(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, values)
}
