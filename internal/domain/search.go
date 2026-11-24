package domain

// SearchPostsResponse описывает страницу результатов поиска публикаций
type SearchPostsResponse struct {
	Query      string `json:"query"`
	Posts      []Post `json:"posts"`
	Cursor     string `json:"cursor,omitempty"`
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// SearchProfilesResponse описывает страницу результатов поиска пользователей
type SearchProfilesResponse struct {
	Query      string         `json:"query"`
	Profiles   []ShortProfile `json:"profiles"`
	Cursor     string         `json:"cursor,omitempty"`
	Limit      int            `json:"limit"`
	HasMore    bool           `json:"has_more"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// SearchCommunitiesResponse описывает страницу результатов поиска сообществ
type SearchCommunitiesResponse struct {
	Query       string      `json:"query"`
	Communities []Community `json:"communities"`
	Cursor      string      `json:"cursor,omitempty"`
	Limit       int         `json:"limit"`
	HasMore     bool        `json:"has_more"`
	NextCursor  string      `json:"next_cursor,omitempty"`
}

// SearchMessagesResponse описывает страницу сообщений из чатов текущего пользователя
type SearchMessagesResponse struct {
	Query      string    `json:"query"`
	Messages   []Message `json:"messages"`
	Cursor     string    `json:"cursor,omitempty"`
	Limit      int       `json:"limit"`
	HasMore    bool      `json:"has_more"`
	NextCursor string    `json:"next_cursor,omitempty"`
}
