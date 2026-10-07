package domain

// FeedResponse — ответ ленты публикаций
type FeedResponse struct {
	Posts      []Post `json:"posts"`
	Cursor     int    `json:"cursor" example:"10"`
	Limit      int    `json:"limit" example:"10"`
	HasMore    bool   `json:"has_more" example:"true"`
	NextCursor int    `json:"next_cursor,omitempty" example:"20"`
}
