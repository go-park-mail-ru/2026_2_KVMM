package domain

import "time"

type RelationshipStatus string

const (
	RelationshipFriendship    RelationshipStatus = "friendship"
	RelationshipFriendRequest RelationshipStatus = "friend_request"
)

// ProfileRelationship — направленная связь между профилями
type ProfileRelationship struct {
	ProfileIDMaster int64              `json:"profile_id_master"`
	ProfileIDSlave  int64              `json:"profile_id_slave"`
	Status          RelationshipStatus `json:"status"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       *time.Time         `json:"updated_at,omitempty"`
	DeletedAt       *time.Time         `json:"deleted_at,omitempty"`
}

// AddFriendRequest — отправка запроса в друзья
type AddFriendRequest struct {
	ProfileID int64 `json:"profile_id" validate:"required,gt=0"`
}

// ListFriendsRequest — фильтрация списка отношений
type ListFriendsRequest struct {
	FriendsRequest RelationshipStatus `json:"friends_request,omitempty"`
	Cursor         int                `json:"cursor" example:"10"`
	Limit          int                `json:"limit" example:"10"`
	HasMore        bool               `json:"has_more" example:"true"`
	NextCursor     int                `json:"next_cursor,omitempty" example:"20"`
}

// DeleteFriendRequest — удаление друга или отмена запроса
type DeleteFriendRequest struct {
	ProfileID int64 `json:"profile_id" validate:"required,gt=0"`
}

// AcceptFriendRequest — принятие входящего запроса
type AcceptFriendRequest struct {
	ProfileID int64 `json:"profile_id" validate:"required,gt=0"`
}

// RelationshipsResponse — результат получения списка отношений
type RelationshipsResponse struct {
	Friends    []ProfileRelationship `json:"friends"`
	Cursor     int                   `json:"cursor" example:"10"`
	Limit      int                   `json:"limit" example:"10"`
	HasMore    bool                  `json:"has_more" example:"true"`
	NextCursor int                   `json:"next_cursor,omitempty" example:"20"`
}

// FriendSuggestion — профиль пользователя, которого можно добавить в друзья
type FriendSuggestion struct {
	ProfileID   int64   `json:"profile_id"`
	Nickname    string  `json:"nickname"`
	ProfileName *string `json:"profile_name,omitempty" example:"Иван"`
	Surname     *string `json:"surname,omitempty" example:"Иванов"`
	AvatarURL   string  `json:"avatar_url,omitempty"`
}

// FriendSuggestionsResponse — ответ со списком возможных друзей
type FriendSuggestionsResponse struct {
	Profiles   []FriendSuggestion `json:"profiles"`
	Cursor     int                `json:"cursor" example:"10"`
	Limit      int                `json:"limit" example:"10"`
	HasMore    bool               `json:"has_more" example:"true"`
	NextCursor int                `json:"next_cursor,omitempty" example:"20"`
}
