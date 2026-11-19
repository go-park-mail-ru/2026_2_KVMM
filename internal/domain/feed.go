package domain

// FeedResponse — ответ ленты публикаций
type FeedResponse struct {
	Posts         []Post `json:"posts"`
	Cursor        int64  `json:"-"`
	CursorUID     string `json:"cursor,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Limit         int    `json:"limit" example:"10"`
	HasMore       bool   `json:"has_more" example:"true"`
	NextCursor    int64  `json:"-"`
	NextCursorUID string `json:"next_cursor,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}
