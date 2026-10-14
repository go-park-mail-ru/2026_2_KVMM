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
	ID                int64      `json:"id" example:"1"`
	PostText          *string    `json:"post_text,omitempty" example:"Привет, мир!"`
	AuthorProfileID   *int64     `json:"author_profile_id,omitempty" example:"1"`
	AuthorCommunityID *int64     `json:"author_community_id,omitempty"`
	Author            Author     `json:"author"`
	CreatedAt         time.Time  `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	CommentsCount     int64      `json:"comments_count" example:"0"`
	RepostsCount      int64      `json:"reposts_count" example:"0"`
	LikesCount        int64      `json:"likes_count" example:"0"`
	MediaCount        int64      `json:"media_count" example:"1"`
	MediaURLs         []string   `json:"media_urls,omitempty"`
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
	PostText string `json:"post_text,omitempty" example:"Привет, мир!" validate:"omitempty,max=256"`
}

// UpdatePostRequest новый текст публикации.
type UpdatePostRequest struct {
	PostID   int64   `json:"post_id" example:"1" validate:"required,gt=0"`
	PostText *string `json:"post_text" example:"Обновлённый текст" validate:"required,max=256"`
}

// UserPostsResponse страница публикаций конкретного пользователя
type UserPostsResponse struct {
	Posts      []Post `json:"posts"`
	ProfileID  int64  `json:"profile_id" example:"1"`
	Cursor     int    `json:"cursor" example:"0"`
	Limit      int    `json:"limit" example:"10"`
	HasMore    bool   `json:"has_more" example:"true"`
	NextCursor int    `json:"next_cursor,omitempty" example:"10"`
}
