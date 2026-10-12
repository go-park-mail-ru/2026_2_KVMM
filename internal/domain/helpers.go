package domain

// PaginationSettings содержит параметры пагинации
type PaginationSettings struct {
	ChatID int64 `json:"chat_id" example:"1" validate:"required,gt=0"`
	Offset int   `json:"offset" example:"0"`
	Limit  int   `json:"limit" example:"10"`
}
