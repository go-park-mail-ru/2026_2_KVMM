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
	Profile             ShortProfile       `json:"profile"`
	RelationshipsStatus RelationshipStatus `json:"relationships_status" example:"request" enums:"request,accepted,canceled"`
	RelationshipsType   string             `json:"relationships_type" example:"friendship"`
	CreatedAt           time.Time          `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt           *time.Time         `json:"updated_at,omitempty" example:"2026-01-02T12:00:00Z"`
	DeletedAt           *time.Time         `json:"deleted_at,omitempty"`
}

// CreateRelationshipRequest — отправка запроса на произвольный вид связи
type CreateRelationshipRequest struct {
	ProfileIDSlave    int64  `json:"profile_id_slave" example:"2" validate:"required,gt=0"`
	RelationshipsType string `json:"relationships_type" example:"friendship" validate:"required,max=32"`
}

// ListRelationshipsRequest — параметры фильтрации и пагинации списка связей
type ListRelationshipsRequest struct {
	RelationshipsStatus RelationshipStatus `json:"relationships_status,omitempty" example:"accepted"`
	RelationshipsType   string             `json:"relationships_type,omitempty" example:"friendship"`
	Cursor              int64              `json:"cursor" example:"0"`
	Limit               int                `json:"limit" example:"10"`
}

// CancleRelationshipRequest — удаление связи или отмена исходящего запроса
type CancleRelationshipRequest struct {
	ProfileID         int64  `json:"profile_id" validate:"required,gt=0"`
	RelationshipsType string `json:"relationships_type" example:"friendship" validate:"required,max=32"`
}

// AcceptRelationshipRequest — подтверждение входящего запроса на связь
type AcceptRelationshipRequest struct {
	ProfileID         int64  `json:"profile_id" validate:"required,gt=0"`
	RelationshipsType string `json:"relationships_type" example:"friendship" validate:"required,max=32"`
}

// RelationshipsResponse — результат получения списка связей
type RelationshipsResponse struct {
	Relationships []ProfileRelationship `json:"relationships"`
	Cursor        int64                 `json:"cursor" example:"0"`
	Limit         int                   `json:"limit" example:"10"`
	HasMore       bool                  `json:"has_more" example:"true"`
	NextCursor    int64                 `json:"next_cursor,omitempty" example:"10"`
}

// RelationshipSuggestion — профиль пользователя, которому можно отправить запрос на связь
type RelationshipSuggestion struct {
	Profile     ShortProfile `json:"profile"`
	CommonCount int64        `json:"common_count" example:"5"`
}

// RelationshipSuggestionsResponse — ответ со списком возможных получателей связи
type RelationshipSuggestionsResponse struct {
	Profiles   []RelationshipSuggestion `json:"profiles"`
	Cursor     int64                    `json:"cursor" example:"0"`
	Limit      int                      `json:"limit" example:"10"`
	HasMore    bool                     `json:"has_more" example:"true"`
	NextCursor int64                    `json:"next_cursor,omitempty" example:"10"`
}

// RelationshipTypes — типы отношений пользователя
type RelationshipTypes struct {
	RelationshipsTypes []string `json:"relationships_types" validate:"required" example:"friendship,family"`
}

type ShortProfile struct {
	ID          int64   `json:"id" example:"1"`
	Nickname    string  `json:"nickname" example:"ivan_petrov"`
	ProfileName *string `json:"profile_name,omitempty" example:"Иван"`
	Surname     *string `json:"surname,omitempty" example:"Иванов"`
	Patronymic  *string `json:"patronymic,omitempty" example:"Иванович"`
	AvatarURL   *string `json:"avatar_urls,omitempty" example:"/api/media/avatar1.png"`
}
