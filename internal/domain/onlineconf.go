package domain

import (
	"encoding/json"
	"time"
)

// OnlineConfScope определяет область применения конфигурации или feature flag
type OnlineConfScope string

const (
	OnlineConfScopeFrontend  OnlineConfScope = "frontend"
	OnlineConfScopeAuth      OnlineConfScope = "auth"
	OnlineConfScopeUser      OnlineConfScope = "user"
	OnlineConfScopePosts     OnlineConfScope = "posts"
	OnlineConfScopeCommunity OnlineConfScope = "community"
	OnlineConfScopeBackend   OnlineConfScope = "backend"
)

// OnlineConfEntryKind определяет тип записи OnlineConf
type OnlineConfEntryKind string

const (
	OnlineConfEntryConfig OnlineConfEntryKind = "config"
	OnlineConfEntryFlag   OnlineConfEntryKind = "flag"
)

// OnlineConfFlagTargetMode определяет принудительное правило раскатки feature flag
type OnlineConfFlagTargetMode string

const (
	OnlineConfTargetInclude OnlineConfFlagTargetMode = "include"
	OnlineConfTargetExclude OnlineConfFlagTargetMode = "exclude"
)

// OnlineConfEntry описывает конфигурационный ключ или feature flag
type OnlineConfEntry struct {
	ID                 int64               `json:"-"`
	UID                string              `json:"uid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Scope              OnlineConfScope     `json:"scope" example:"posts"`
	Name               string              `json:"name" example:"max_text_length"`
	Kind               OnlineConfEntryKind `json:"kind" example:"config"`
	Value              json.RawMessage     `json:"value,omitempty" swaggertype:"object"`
	Rollout            *int                `json:"rollout,omitempty" example:"20"`
	Include            []string            `json:"include,omitempty"`
	Exclude            []string            `json:"exclude,omitempty"`
	CreatedByProfileID int64               `json:"-"`
	UpdatedByProfileID int64               `json:"-"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          *time.Time          `json:"updated_at,omitempty"`
}

// OnlineConfEntriesResponse содержит список конфигураций и feature flags для административного API.
type OnlineConfEntriesResponse struct {
	Entries []OnlineConfEntry `json:"entries"`
}

// CreateOnlineConfEntryRequest содержит данные новой конфигурации или feature flag.
type CreateOnlineConfEntryRequest struct {
	Scope   OnlineConfScope     `json:"scope" example:"posts"`
	Name    string              `json:"name" example:"max_text_length"`
	Kind    OnlineConfEntryKind `json:"kind" example:"config"`
	Value   json.RawMessage     `json:"value,omitempty" swaggertype:"object"`
	Rollout *int                `json:"rollout,omitempty" example:"20"`
	Include []string            `json:"include,omitempty"`
	Exclude []string            `json:"exclude,omitempty"`
}

// UpdateOnlineConfEntryRequest содержит изменяемые поля существующей записи.
type UpdateOnlineConfEntryRequest struct {
	Value   *json.RawMessage `json:"value,omitempty" swaggertype:"object"`
	Rollout *int             `json:"rollout,omitempty" example:"20"`
	Include *[]string        `json:"include,omitempty"`
	Exclude *[]string        `json:"exclude,omitempty"`
}

// OnlineConfFlagTarget содержит внутренние и публичные данные пользователя для раскатки флага
type OnlineConfFlagTarget struct {
	ProfileID  int64                    `json:"-"`
	ProfileUID string                   `json:"profile_uid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Mode       OnlineConfFlagTargetMode `json:"mode" example:"include"`
}

// OnlineConfFlagSnapshot содержит данные feature flag для продуктового сервиса
type OnlineConfFlagSnapshot struct {
	Rollout    int     `json:"rollout" example:"20"`
	IncludeIDs []int64 `json:"include_ids"`
	ExcludeIDs []int64 `json:"exclude_ids"`
}

// OnlineConfSnapshot содержит конфигурации и флаги одного scope для кеширования сервисом
type OnlineConfSnapshot struct {
	Scope   OnlineConfScope                   `json:"scope" example:"posts"`
	Configs map[string]json.RawMessage        `json:"configs"`
	Flags   map[string]OnlineConfFlagSnapshot `json:"flags"`
}
