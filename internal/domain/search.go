package domain

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
