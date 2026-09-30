package domain

import (
	"time"
)

// Post — публикация пользователя в ленте
type Post struct {
	ID                int64      `json:"id" examples:"1"`
	PostText          *string    `json:"post_text,omitempty" examples:"Привет, мир!"`
	AuthorProfileID   *int64     `json:"author_profile_id,omitempty" examples:"1"`
	AuthorCommunityID *int64     `json:"author_community_id,omitempty"`
	Author            Profile    `json:"author"`
	CreatedAt         time.Time  `json:"created_at" examples:"2026-01-01T12:00:00Z"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	CommentsCount     int64      `json:"comments_count" examples:"0"`
	RepostsCount      int64      `json:"reposts_count" examples:"0"`
	LikesCount        int64      `json:"likes_count" examples:"0"`
	MediaCount        int64      `json:"media_count" examples:"1"`
	MediaURLs         []string   `json:"media_urls,omitempty"`
}
