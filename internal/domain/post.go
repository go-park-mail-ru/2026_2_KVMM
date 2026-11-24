package domain

import (
	"time"
)

type PostAuthorType string

const (
	AuthorProfile   PostAuthorType = "profile"
	AuthorCommunity PostAuthorType = "community"
)

// Post — публикация пользователя в ленте
type Post struct {
	ID                 int64      `json:"-"`
	UID                string     `json:"uid" example:"550e8400-e29b-41d4-a716-446655440000"`
	PostText           *string    `json:"post_text,omitempty" example:"Привет, мир!"`
	AuthorProfileID    *int64     `json:"-"`
	AuthorProfileUID   *string    `json:"author_profile_uid,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorCommunityID  *int64     `json:"-"`
	AuthorCommunityUID *string    `json:"author_community_uid,omitempty"`
	Author             Author     `json:"author"`
	CreatedAt          time.Time  `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
	CommentsCount      int64      `json:"comments_count" example:"0"`
	RepostsCount       int64      `json:"reposts_count" example:"0"`
	LikesCount         int64      `json:"likes_count" example:"0"`
	MediaCount         int64      `json:"media_count" example:"1"`
	MediaURLs          []string   `json:"media_urls,omitempty"`
}

// Post — публикация пользователя в ленте
type PostUpdateRequest struct {
	ID        int64    `json:"-"`
	UID       string   `json:"uid,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	PostText  *string  `json:"post_text,omitempty" example:"Привет, мир!"`
	MediaURLs []string `json:"media_urls,omitempty"`
}

type Author struct {
	// Name - для пользователей: surname + " " + profileName, если не добавлено Name - nickname;
	// Name - для сообществ: community_name
	Name      string         `json:"name" example:"Petrov Ivan"`
	AvatarURL string         `json:"avatar_urls,omitempty" example:"/api/media/avatar1.png"`
	Type      PostAuthorType `json:"type,omitempty" example:"profile"`
}

// CreatePostRequest содержит данные для создания публикации
type CreatePostRequest struct {
	PostText  string   `json:"post_text,omitempty" example:"Привет, мир!" validate:"omitempty,max=2048"`
	MediaURLs []string `json:"media_urls,omitempty"`
}

// UpdatePostRequest новый текст публикации.
type UpdatePostRequest struct {
	PostID   int64   `json:"-"`
	PostUID  string  `json:"post_uid" example:"550e8400-e29b-41d4-a716-446655440000" validate:"required"`
	PostText *string `json:"post_text" example:"Обновлённый текст" validate:"required,max=2048"`
}

// UserPostsResponse страница публикаций конкретного пользователя
type UserPostsResponse struct {
	Posts         []Post `json:"posts"`
	ProfileID     int64  `json:"-"`
	ProfileUID    string `json:"profile_uid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Cursor        int64  `json:"-"`
	CursorUID     string `json:"cursor,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Limit         int    `json:"limit" example:"10"`
	HasMore       bool   `json:"has_more" example:"true"`
	NextCursor    int64  `json:"-"`
	NextCursorUID string `json:"next_cursor,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}
