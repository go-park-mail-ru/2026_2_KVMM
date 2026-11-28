// Package onlineconf содержит HTTP-доставку административного API OnlineConf.
package onlineconf

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	onlineconfusecase "kvmm/internal/usecase/onlineconf"
	sessionusecase "kvmm/internal/usecase/session"

	"github.com/gorilla/mux"
)

// adminUseCase описывает сценарии административного API OnlineConf.
type adminUseCase interface {
	CheckAccess(ctx context.Context, sessionToken string) error
	ListEntries(ctx context.Context, sessionToken string, scope domain.OnlineConfScope, query string) ([]domain.OnlineConfEntry, error)
	CreateEntry(ctx context.Context, sessionToken, csrfToken string, req domain.CreateOnlineConfEntryRequest) (domain.OnlineConfEntry, error)
	UpdateEntry(ctx context.Context, sessionToken, csrfToken string, entryUID string, req domain.UpdateOnlineConfEntryRequest) (domain.OnlineConfEntry, error)
	Snapshot(ctx context.Context, scope domain.OnlineConfScope) (domain.OnlineConfSnapshot, error)
}

// Handler обрабатывает HTTP-запросы административной панели OnlineConf.
type Handler struct {
	useCase       adminUseCase
	internalToken string
}

// NewHandler создаёт HTTP-обработчик административного API OnlineConf.
func NewHandler(useCase adminUseCase, internalToken string) *Handler {
	return &Handler{useCase: useCase, internalToken: internalToken}
}

// NewRouter создаёт маршруты HTTP API OnlineConf.
func NewRouter(handler *Handler) http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/health", Health).Methods(http.MethodGet)
	router.HandleFunc("/internal/onlineconf/admin/access", handler.Access).Methods(http.MethodGet)
	router.HandleFunc("/internal/onlineconf/snapshot", handler.Snapshot).Methods(http.MethodGet)
	router.HandleFunc("/api/onlineconf/admin/entries", handler.ListEntries).Methods(http.MethodGet)
	router.HandleFunc("/api/onlineconf/admin/entries", handler.CreateEntry).Methods(http.MethodPost)
	router.HandleFunc("/api/onlineconf/admin/entries/{uid}", handler.UpdateEntry).Methods(http.MethodPatch)
	return router
}

// Health возвращает состояние работоспособности сервиса.
func Health(w http.ResponseWriter, _ *http.Request) {
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Access проверяет доступ текущего пользователя к административной панели.
// @Summary Проверить доступ к OnlineConf
// @Tags onlineconf
// @Security SessionCookie
// @Produce json
// @Success 204
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Router /internal/onlineconf/admin/access [get]
func (h *Handler) Access(w http.ResponseWriter, r *http.Request) {
	if err := h.useCase.CheckAccess(r.Context(), helpers.SessionCookie(r)); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Snapshot возвращает снимок конфигураций одного scope для продуктового сервиса.
// @Summary Получить внутренний снимок OnlineConf
// @Tags onlineconf
// @Produce json
// @Param scope query string true "Scope: frontend, auth, user, posts, community или backend"
// @Param X-OnlineConf-Token header string true "Внутренний токен сервиса"
// @Success 200 {object} domain.OnlineConfSnapshot
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /internal/onlineconf/snapshot [get]
func (h *Handler) Snapshot(w http.ResponseWriter, r *http.Request) {
	if h.internalToken == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-OnlineConf-Token")), []byte(h.internalToken)) != 1 {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "valid internal token is required")
		return
	}
	snapshot, err := h.useCase.Snapshot(r.Context(), domain.OnlineConfScope(strings.TrimSpace(r.URL.Query().Get("scope"))))
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, snapshot)
}

// ListEntries возвращает конфигурации и feature flags с фильтрацией по scope и названию.
// @Summary Получить конфигурации и feature flags
// @Tags onlineconf
// @Security SessionCookie
// @Produce json
// @Param scope query string false "Scope: frontend, auth, user, posts, community или backend"
// @Param q query string false "Подстрока имени"
// @Success 200 {object} domain.OnlineConfEntriesResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/onlineconf/admin/entries [get]
func (h *Handler) ListEntries(w http.ResponseWriter, r *http.Request) {
	entries, err := h.useCase.ListEntries(
		r.Context(),
		helpers.SessionCookie(r),
		domain.OnlineConfScope(strings.TrimSpace(r.URL.Query().Get("scope"))),
		r.URL.Query().Get("q"),
	)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, domain.OnlineConfEntriesResponse{Entries: entries})
}

// CreateEntry создаёт конфигурационный ключ или feature flag.
// @Summary Создать конфигурацию или feature flag
// @Tags onlineconf
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.CreateOnlineConfEntryRequest true "Новая запись"
// @Success 201 {object} domain.OnlineConfEntry
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/onlineconf/admin/entries [post]
func (h *Handler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateOnlineConfEntryRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	entry, err := h.useCase.CreateEntry(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, entry)
}

// UpdateEntry изменяет значение конфигурации, rollout или исключения feature flag.
// @Summary Изменить конфигурацию или feature flag
// @Tags onlineconf
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param uid path string true "UUID записи"
// @Param request body domain.UpdateOnlineConfEntryRequest true "Изменяемые поля"
// @Success 200 {object} domain.OnlineConfEntry
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/onlineconf/admin/entries/{uid} [patch]
func (h *Handler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	entryUID := strings.ToLower(strings.TrimSpace(mux.Vars(r)["uid"]))
	if !helpers.IsUUID(entryUID) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "uid must be a valid UUID")
		return
	}
	var req domain.UpdateOnlineConfEntryRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	entry, err := h.useCase.UpdateEntry(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), entryUID, req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, entry)
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sessionusecase.ErrUnauthorized):
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case errors.Is(err, sessionusecase.ErrInvalidCSRF):
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
	case errors.Is(err, onlineconfusecase.ErrForbidden):
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "onlineconf admin access required")
	case errors.Is(err, onlineconfusecase.ErrInvalidEntry):
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, onlineconfusecase.ErrEntryNotFound):
		helpers.WriteError(w, http.StatusNotFound, "not_found", "onlineconf entry not found")
	case errors.Is(err, onlineconfusecase.ErrEntryConflict):
		helpers.WriteError(w, http.StatusConflict, "conflict", "onlineconf entry already exists")
	default:
		helpers.WriteInternalError(w, "onlineconf", err)
	}
}
