package domain

import "time"

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
)

// Profile — публичные данные пользователя
type Profile struct {
	ID          int64      `json:"id" example:"1"`
	Nickname    string     `json:"nickname" example:"ivan_petrov"`
	Email       *string    `json:"email,omitempty" example:"ivan@example.com"`
	PhoneNumber *string    `json:"phone_number,omitempty" example:"+79999999999"`
	ProfileName *string    `json:"profile_name,omitempty" example:"Иван"`
	Surname     *string    `json:"surname,omitempty" example:"Иванов"`
	Patronymic  *string    `json:"patronymic,omitempty" example:"Иванович"`
	Gender      *Gender    `json:"gender,omitempty" example:"male" enums:"male,female,other"`
	Birthday    *time.Time `json:"birthday,omitempty" example:"2000-05-14T00:00:00Z"`
	Bio         *string    `json:"bio,omitempty" example:"hi!"`
	AvatarURLs  []string   `json:"avatar_urls,omitempty" example:"/api/media/avatar1.png"`
	CreatedAt   time.Time  `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" example:"2026-01-02T12:00:00Z"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// UpdateProfileRequest — поля, которые можно изменить в профиле
type UpdateProfileRequest struct {
	Nickname    *string `json:"nickname,omitempty" example:"ivan67" validate:"omitempty,min=4,max=32"`
	Email       *string `json:"email,omitempty" example:"ivan@example.com" validate:"omitempty,max=254"`
	PhoneNumber *string `json:"phone_number,omitempty" example:"+79999999999" validate:"omitempty"`

	Password        *string `json:"password,omitempty" example:"sjkdfhgs4536njbjjkh" validate:"omitempty,min=8,max=64"`
	ConfirmPassword *string `json:"confirm_password,omitempty" example:"sjkdfhgs4536njbjjkh" validate:"omitempty"`

	ProfileName *string `json:"profile_name,omitempty" example:"Иван" validate:"omitempty,min=2,max=32"`
	Surname     *string `json:"surname,omitempty" example:"Иванов" validate:"omitempty,min=2,max=32"`
	Patronymic  *string `json:"patronymic,omitempty" example:"Иванович" validate:"omitempty,min=2,max=32"`

	Gender   *Gender `json:"gender,omitempty" example:"male" validate:"omitempty,oneof=male female"`
	Birthday *string `json:"birthday,omitempty" example:"2000-05-14" validate:"omitempty"`
	Bio      *string `json:"bio,omitempty" example:"hi!" validate:"omitempty,max=256"`
}
