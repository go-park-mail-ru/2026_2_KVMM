package posts

import (
	"net/http"
	"strconv"
	"strings"

	"kvmm/internal/handler/helpers"
)

// ListCommunity возвращает страницу постов сообщества
// @Summary Получить посты сообщества
// @Tags community-posts
// @Produce json
// @Param community_uid query string true "UUID сообщества"
// @Param cursor query string false "UUID последнего поста"
// @Param limit query int false "Размер страницы" default(10) maximum(100)
// @Success 200 {object} domain.CommunityPostsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/community-posts/list [get]
func (h *PostHandler) ListCommunity(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	communityUID := strings.TrimSpace(query.Get("community_uid"))
	cursorUID := strings.TrimSpace(query.Get("cursor"))
	if !helpers.IsUUID(communityUID) || (cursorUID != "" && !helpers.IsUUID(cursorUID)) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "community_uid and cursor must be valid UUIDs")
		return
	}
	limit := 10
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			helpers.WriteError(w, http.StatusBadRequest, "validation_error", "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	response, err := h.useCase.CommunityPosts(r.Context(), communityUID, cursorUID, limit, helpers.SessionCookie(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
