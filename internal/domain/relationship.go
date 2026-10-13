package domain

import "time"

type RelationshipStatus string

const (
	RelationshipRequest  RelationshipStatus = "request"
	RelationshipAccept   RelationshipStatus = "accepted"
	RelationshipCanceled RelationshipStatus = "canceled"
)

// ProfileRelationship — направленная связь между профилями
type ProfileRelationship struct {
	ProfileIDMaster     int64              `json:"profile_id_master" example:"1"`
	ProfileIDSlave      int64              `json:"profile_id_slave" example:"2"`
	RelationshipsStatus RelationshipStatus `json:"relationships_status" example:"request" enums:"request,approved,canceled"`
	RelationshipsType   string             `json:"relationships_type" example:"friendship"`
	CreatedAt           time.Time          `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt           *time.Time         `json:"updated_at,omitempty" example:"2026-01-02T12:00:00Z"`
	DeletedAt           *time.Time         `json:"deleted_at,omitempty"`
}

// CreateRelationshipRequest — отправка запроса на произвольный вид связи
type CreateRelationshipRequest struct {
	ProfileIDSlave    int64  `json:"profile_id_slave" example:"2" validate:"required,gt=0"`
	RelationshipsType string `json:"relationships_type" example:"friendship" validate:"required"`
}

// ListRelationshipsRequest — параметры фильтрации и пагинации списка связей
type ListRelationshipsRequest struct {
	RelationshipsStatus RelationshipStatus `json:"relationships_status,omitempty" example:"approved"`
	RelationshipsType   string             `json:"relationships_type,omitempty" example:"friendship"`
	Cursor              int                `json:"cursor" example:"0"`
	Limit               int                `json:"limit" example:"10"`
}

// CancleRelationshipRequest — удаление связи или отмена исходящего запроса
type CancleRelationshipRequest struct {
	ProfileID         int64  `json:"profile_id" validate:"required,gt=0"`
	RelationshipsType string `json:"relationships_type" example:"friendship" validate:"required"`
}

// AcceptRelationshipRequest — подтверждение входящего запроса на связь
type AcceptRelationshipRequest struct {
	ProfileID         int64  `json:"profile_id" validate:"required,gt=0"`
	RelationshipsType string `json:"relationships_type" example:"friendship" validate:"required"`
}

// RelationshipsResponse — результат получения списка связей
type RelationshipsResponse struct {
	Relationships []ProfileRelationship `json:"relationships"`
	Cursor        int                   `json:"cursor" example:"0"`
	Limit         int                   `json:"limit" example:"10"`
	HasMore       bool                  `json:"has_more" example:"true"`
	NextCursor    int                   `json:"next_cursor,omitempty" example:"10"`
}

// RelationshipSuggestion — профиль пользователя, которому можно отправить запрос на связь
type RelationshipSuggestion struct {
	ProfileID int64  `json:"profile_id"`
	Nickname  string `json:"nickname"`
}

// RelationshipSuggestionsResponse — ответ со списком возможных получателей связи
type RelationshipSuggestionsResponse struct {
	Profiles   []RelationshipSuggestion `json:"profiles"`
	Cursor     int                      `json:"cursor" example:"0"`
	Limit      int                      `json:"limit" example:"10"`
	HasMore    bool                     `json:"has_more" example:"true"`
	NextCursor int                      `json:"next_cursor,omitempty" example:"10"`
}

// RelationshipTypes — типы отношений пользователя
type RelationshipTypes struct {
	RelationshipsTypes []RelationshipSuggestion `json:"relationships_types" validate:"required"`
}
