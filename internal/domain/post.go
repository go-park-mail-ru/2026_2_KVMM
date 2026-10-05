package domain

import (
	"time"
)

// Post — публикация пользователя в ленте
type Post struct {
	ID                int64      `json:"id" example:"1"`
	PostText          *string    `json:"post_text,omitempty" example:"Привет, мир!"`
	AuthorProfileID   *int64     `json:"author_profile_id,omitempty" example:"1"`
	AuthorCommunityID *int64     `json:"author_community_id,omitempty"`
	Author            Profile    `json:"author"`
	CreatedAt         time.Time  `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	CommentsCount     int64      `json:"comments_count" example:"0"`
	RepostsCount      int64      `json:"reposts_count" example:"0"`
	LikesCount        int64      `json:"likes_count" example:"0"`
	MediaCount        int64      `json:"media_count" example:"1"`
	MediaURLs         []string   `json:"media_urls,omitempty"`
}
