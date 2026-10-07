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
	ProfileName string     `json:"profile_name" example:"Иван"`
	Surname     string     `json:"surname" example:"Иванов"`
	Patronymic  *string    `json:"patronymic,omitempty" example:"Иванович"`
	Gender      Gender     `json:"gender" example:"male" enums:"male,female,other"`
	Birthday    *time.Time `json:"birthday,omitempty" example:"2000-05-14T00:00:00Z"`
	Bio         *string    `json:"bio,omitempty" example:"hi!"`
	ImageURL    string     `json:"image_url,omitempty" example:"/api/media/avatar1.png"`
	AvatarURLs  []string   `json:"avatar_urls,omitempty"`
	CreatedAt   time.Time  `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" example:"2026-01-02T12:00:00Z"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// RegisterRequest - запрос на регистрацию
type RegisterRequest struct {
	Nickname    string  `json:"nickname" example:"ivan67" validate:"required,min=4,max=32"`
	Email       *string `json:"email,omitempty" example:"ivan@example.com" validate:"omitempty,max=254"`
	PhoneNumber *string `json:"phone_number,omitempty" example:"+79999999999" validate:"omitempty"`

	Password        string `json:"password" example:"sjkdfhgs4536njbjjkh" validate:"required,min=8,max=64"`
	ConfirmPassword string `json:"confirm_password" example:"sjkdfhgs4536njbjjkh" validate:"required,eqfield=Password"`

	ProfileName string  `json:"profile_name" example:"Иван" validate:"required,min=2,max=32"`
	Surname     string  `json:"surname" example:"Иванов" validate:"required,min=2,max=32"`
	Patronymic  *string `json:"patronymic,omitempty" example:"Иванович" validate:"omitempty,min=2,max=32"`

	Gender   Gender  `json:"gender" example:"male" validate:"required,oneof=male female"`
	Birthday *string `json:"birthday,omitempty" example:"2000-05-14" validate:"required"`
	Bio      *string `json:"bio,omitempty" example:"hi!" validate:"omitempty,max=256"`
}

// LoginRequest — Login может быть nickname, email или phone_number
type LoginRequest struct {
	Login    string `json:"login" example:"ivan_petrov" validate:"required,max=254"`
	Password string `json:"password" example:"strongpassword123" validate:"required,min=8,max=64"`
}
