package domain

import "time"

// Community описывает сообщество
type Community struct {
	ID               int64      `json:"-"`
	UID              string     `json:"uid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name             string     `json:"community_name" example:"Go developers"`
	Bio              string     `json:"bio" example:"Сообщество разработчиков Go"`
	AvatarURL        *string    `json:"avatar_url,omitempty" example:"/api/media/community_avatars/avatar.png"`
	SubscribersCount int64      `json:"subscribers_count" example:"42"`
	IsSubscribed     bool       `json:"is_subscribed" example:"true"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
}

// CreateCommunityRequest содержит данные нового сообщества
type CreateCommunityRequest struct {
	Name string `json:"community_name" validate:"required,min=2,max=128"`
	Bio  string `json:"bio" validate:"max=256"`
}

// UpdateCommunityRequest содержит изменяемые поля сообщества
type UpdateCommunityRequest struct {
	Name *string `json:"community_name,omitempty" validate:"omitempty,min=2,max=128"`
	Bio  *string `json:"bio,omitempty" validate:"omitempty,max=256"`
}

// CommunityPage содержит страницу сообществ
type CommunityPage struct {
	Communities []Community `json:"communities"`
	Offset      int         `json:"offset"`
	Limit       int         `json:"limit"`
	HasMore     bool        `json:"has_more"`
}

// AssignCommunityAdminRequest указывает UUID участника, которому назначают администратора
type AssignCommunityAdminRequest struct {
	ProfileUID string `json:"profile_uid" validate:"required"`
}

// CommunityPostsResponse содержит страницу публикаций сообщества
type CommunityPostsResponse struct {
	Posts        []Post `json:"posts"`
	CommunityUID string `json:"community_uid"`
	Cursor       string `json:"cursor,omitempty"`
	Limit        int    `json:"limit"`
	HasMore      bool   `json:"has_more"`
	NextCursor   string `json:"next_cursor,omitempty"`
}

// CommunityPostRequest содержит данные публикации сообщества
type CommunityPostRequest struct {
	CommunityUID string  `json:"community_uid" validate:"required"`
	PostText     *string `json:"post_text,omitempty"`
}

// UpdateCommunityPostRequest содержит UUID публикации и её новый текст
type UpdateCommunityPostRequest struct {
	CommunityUID string  `json:"community_uid" validate:"required"`
	PostUID      string  `json:"post_uid" validate:"required"`
	PostText     *string `json:"post_text" validate:"required"`
}
