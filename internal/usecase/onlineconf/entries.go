package onlineconf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

var (
	ErrForbidden     = errors.New("onlineconf admin access required")
	ErrInvalidEntry  = errors.New("invalid onlineconf entry")
	ErrEntryNotFound = repository.ErrOnlineConfEntryNotFound
	ErrEntryConflict = repository.ErrOnlineConfEntryConflict
)

var onlineConfNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)

type entryRepository interface {
	CreateEntry(context.Context, repository.CreateOnlineConfEntryParams) (domain.OnlineConfEntry, error)
	UpdateEntry(context.Context, string, repository.UpdateOnlineConfEntryParams) (domain.OnlineConfEntry, error)
	ReadEntry(context.Context, string) (domain.OnlineConfEntry, error)
	ListEntries(context.Context, domain.OnlineConfScope, string) ([]domain.OnlineConfEntry, error)
	Snapshot(context.Context, domain.OnlineConfScope) (domain.OnlineConfSnapshot, error)
}

type profileResolver interface {
	GetProfileIDByUID(context.Context, string) (int64, error)
}

// AdminUseCase содержит сценарии управления конфигурациями и feature flags из админки.
type AdminUseCase struct {
	entries  entryRepository
	sessions *sessionusecase.UseCase
	profiles profileResolver
	adminID  int64
}

// NewAdminUseCase создаёт usecase административного API OnlineConf.
func NewAdminUseCase(
	entries entryRepository,
	sessions *sessionusecase.UseCase,
	profiles profileResolver,
	adminID int64,
) *AdminUseCase {
	return &AdminUseCase{
		entries:  entries,
		sessions: sessions,
		profiles: profiles,
		adminID:  adminID,
	}
}

// CheckAccess проверяет, что пользователь текущей Nexly-сессии имеет доступ к админке.
func (u *AdminUseCase) CheckAccess(ctx context.Context, sessionToken string) error {
	_, err := u.authorizeRead(ctx, sessionToken)
	return err
}

// ListEntries возвращает конфигурации и feature flags с необязательной фильтрацией.
func (u *AdminUseCase) ListEntries(
	ctx context.Context,
	sessionToken string,
	scope domain.OnlineConfScope,
	query string,
) ([]domain.OnlineConfEntry, error) {
	if _, err := u.authorizeRead(ctx, sessionToken); err != nil {
		return nil, err
	}
	if scope != "" && !isOnlineConfScope(scope) {
		return nil, fmt.Errorf("%w: unknown scope", ErrInvalidEntry)
	}
	return u.entries.ListEntries(ctx, scope, query)
}

// CreateEntry создаёт новую конфигурацию или feature flag.
func (u *AdminUseCase) CreateEntry(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.CreateOnlineConfEntryRequest,
) (domain.OnlineConfEntry, error) {
	adminID, err := u.authorizeWrite(ctx, sessionToken, csrfToken)
	if err != nil {
		return domain.OnlineConfEntry{}, err
	}
	if err := validateNewEntry(req); err != nil {
		return domain.OnlineConfEntry{}, err
	}
	targets, err := u.resolveTargets(ctx, req.Include, req.Exclude)
	if err != nil {
		return domain.OnlineConfEntry{}, err
	}
	return u.entries.CreateEntry(ctx, repository.CreateOnlineConfEntryParams{
		Scope:              req.Scope,
		Name:               strings.TrimSpace(req.Name),
		Kind:               req.Kind,
		Value:              req.Value,
		Rollout:            req.Rollout,
		Targets:            targets,
		CreatedByProfileID: adminID,
	})
}

// UpdateEntry изменяет значение конфигурации, rollout или персональные правила feature flag.
func (u *AdminUseCase) UpdateEntry(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	entryUID string,
	req domain.UpdateOnlineConfEntryRequest,
) (domain.OnlineConfEntry, error) {
	adminID, err := u.authorizeWrite(ctx, sessionToken, csrfToken)
	if err != nil {
		return domain.OnlineConfEntry{}, err
	}
	entry, err := u.entries.ReadEntry(ctx, entryUID)
	if err != nil {
		return domain.OnlineConfEntry{}, err
	}

	params := repository.UpdateOnlineConfEntryParams{
		Value:              req.Value,
		Rollout:            req.Rollout,
		UpdatedByProfileID: adminID,
	}
	switch entry.Kind {
	case domain.OnlineConfEntryConfig:
		if req.Value == nil || req.Rollout != nil || req.Include != nil || req.Exclude != nil {
			return domain.OnlineConfEntry{}, fmt.Errorf("%w: config can update only value", ErrInvalidEntry)
		}
		if !json.Valid(*req.Value) {
			return domain.OnlineConfEntry{}, fmt.Errorf("%w: value must be valid JSON", ErrInvalidEntry)
		}
		if err := validateProductConfig(entry.Scope, entry.Name, *req.Value); err != nil {
			return domain.OnlineConfEntry{}, err
		}
	case domain.OnlineConfEntryFlag:
		if req.Value != nil || (req.Rollout == nil && req.Include == nil && req.Exclude == nil) {
			return domain.OnlineConfEntry{}, fmt.Errorf("%w: flag requires rollout or target changes", ErrInvalidEntry)
		}
		if req.Rollout != nil && (*req.Rollout < 0 || *req.Rollout > 100) {
			return domain.OnlineConfEntry{}, fmt.Errorf("%w: rollout must be between 0 and 100", ErrInvalidEntry)
		}
		if req.Include != nil || req.Exclude != nil {
			include, exclude := entry.Include, entry.Exclude
			if req.Include != nil {
				include = *req.Include
			}
			if req.Exclude != nil {
				exclude = *req.Exclude
			}
			targets, resolveErr := u.resolveTargets(ctx, include, exclude)
			if resolveErr != nil {
				return domain.OnlineConfEntry{}, resolveErr
			}
			params.ReplaceTargets = true
			params.Targets = targets
		}
	default:
		return domain.OnlineConfEntry{}, fmt.Errorf("%w: unknown entry kind", ErrInvalidEntry)
	}
	return u.entries.UpdateEntry(ctx, entryUID, params)
}

// Snapshot возвращает снимок scope для gRPC-контракта продуктовых сервисов.
func (u *AdminUseCase) Snapshot(ctx context.Context, scope domain.OnlineConfScope) (domain.OnlineConfSnapshot, error) {
	if !isOnlineConfScope(scope) {
		return domain.OnlineConfSnapshot{}, fmt.Errorf("%w: unknown scope", ErrInvalidEntry)
	}
	return u.entries.Snapshot(ctx, scope)
}

func (u *AdminUseCase) authorizeRead(ctx context.Context, sessionToken string) (int64, error) {
	profileID, _, err := u.sessions.Read(sessionToken)
	if err != nil {
		return 0, err
	}
	if profileID != u.adminID {
		return 0, ErrForbidden
	}
	return profileID, nil
}

func (u *AdminUseCase) authorizeWrite(ctx context.Context, sessionToken, csrfToken string) (int64, error) {
	profileID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return 0, err
	}
	if profileID != u.adminID {
		return 0, ErrForbidden
	}
	return profileID, nil
}

func validateNewEntry(req domain.CreateOnlineConfEntryRequest) error {
	if !isOnlineConfScope(req.Scope) {
		return fmt.Errorf("%w: unknown scope", ErrInvalidEntry)
	}
	name := strings.TrimSpace(req.Name)
	if !onlineConfNamePattern.MatchString(name) {
		return fmt.Errorf("%w: name must contain lowercase latin letters, digits and _", ErrInvalidEntry)
	}
	switch req.Kind {
	case domain.OnlineConfEntryConfig:
		if len(req.Value) == 0 || !json.Valid(req.Value) || req.Rollout != nil || len(req.Include) > 0 || len(req.Exclude) > 0 {
			return fmt.Errorf("%w: config requires only valid JSON value", ErrInvalidEntry)
		}
		if req.Scope == domain.OnlineConfScopePosts && name == "likes_enabled" {
			return fmt.Errorf("%w: posts.likes_enabled must have kind flag", ErrInvalidEntry)
		}
		return validateProductConfig(req.Scope, name, req.Value)
	case domain.OnlineConfEntryFlag:
		if len(req.Value) > 0 || req.Rollout == nil || *req.Rollout < 0 || *req.Rollout > 100 {
			return fmt.Errorf("%w: flag requires rollout between 0 and 100", ErrInvalidEntry)
		}
		if req.Scope == domain.OnlineConfScopePosts && name == "likes_enabled" {
			return nil
		}
		return nil
	default:
		return fmt.Errorf("%w: kind must be config or flag", ErrInvalidEntry)
	}
}

func validateProductConfig(scope domain.OnlineConfScope, name string, value json.RawMessage) error {
	if scope != domain.OnlineConfScopePosts || name != "max_text_length" {
		return nil
	}
	var length int
	if err := json.Unmarshal(value, &length); err != nil || length < 1 || length > 2048 {
		return fmt.Errorf("%w: posts.max_text_length must be an integer between 1 and 2048", ErrInvalidEntry)
	}
	return nil
}

func (u *AdminUseCase) resolveTargets(
	ctx context.Context,
	include []string,
	exclude []string,
) ([]domain.OnlineConfFlagTarget, error) {
	targets := make([]domain.OnlineConfFlagTarget, 0, len(include)+len(exclude))
	seen := make(map[string]struct{}, len(include)+len(exclude))
	appendTargets := func(values []string, mode domain.OnlineConfFlagTargetMode) error {
		for _, value := range values {
			uid := strings.TrimSpace(value)
			if uid == "" {
				return fmt.Errorf("%w: profile UUID is required", ErrInvalidEntry)
			}
			if _, exists := seen[uid]; exists {
				return fmt.Errorf("%w: profile UUID cannot be repeated in include or exclude", ErrInvalidEntry)
			}
			profileID, err := u.profiles.GetProfileIDByUID(ctx, uid)
			if err != nil {
				if errors.Is(err, repository.ErrProfileNotFound) {
					return fmt.Errorf("%w: profile %s does not exist", ErrInvalidEntry, uid)
				}
				return err
			}
			seen[uid] = struct{}{}
			targets = append(targets, domain.OnlineConfFlagTarget{ProfileID: profileID, ProfileUID: uid, Mode: mode})
		}
		return nil
	}
	if err := appendTargets(include, domain.OnlineConfTargetInclude); err != nil {
		return nil, err
	}
	if err := appendTargets(exclude, domain.OnlineConfTargetExclude); err != nil {
		return nil, err
	}
	return targets, nil
}

func isOnlineConfScope(scope domain.OnlineConfScope) bool {
	switch scope {
	case domain.OnlineConfScopeFrontend,
		domain.OnlineConfScopeAuth,
		domain.OnlineConfScopeUser,
		domain.OnlineConfScopePosts,
		domain.OnlineConfScopeCommunity,
		domain.OnlineConfScopeBackend:
		return true
	default:
		return false
	}
}
