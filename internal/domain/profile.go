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
	ID          int64      `json:"id" examples:"1"`
	Nickname    string     `json:"nickname" examples:"ivan_petrov"`
	Email       *string    `json:"email,omitempty" examples:"ivan@example.com"`
	PhoneNumber *string    `json:"phone_number,omitempty" examples:"+79999999999"`
	ProfileName string     `json:"profile_name" examples:"Иван"`
	Surname     string     `json:"surname" examples:"Иванов"`
	Patronymic  *string    `json:"patronymic,omitempty" examples:"Иванович"`
	Gender      Gender     `json:"gender" examples:"male" enums:"male,female,other"`
	Birthday    *time.Time `json:"birthday,omitempty" examples:"2000-05-14T00:00:00Z"`
	Bio         *string    `json:"bio,omitempty" examples:"hi!"`
	ImageURL    string     `json:"image_url,omitempty" examples:"/api/media/avatar1.png"`
	AvatarURLs  []string   `json:"avatar_urls,omitempty"`
	CreatedAt   time.Time  `json:"created_at" examples:"2026-01-01T12:00:00Z"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" examples:"2026-01-02T12:00:00Z"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// RegisterRequest - запрос на регистрацию
type RegisterRequest struct {
	Nickname    string  `json:"nickname" examples:"ivan67" validate:"required,min=4,max=32"`
	Email       *string `json:"email,omitempty" examples:"ivan@example.com" validate:"omitempty"`
	PhoneNumber *string `json:"phone_number,omitempty" examples:"+79999999999" validate:"omitempty"`

	Password        string `json:"password" examples:"sjkdfhgs4536njbjjkh" validate:"required,min=8,max=72"`
	ConfirmPassword string `json:"confirm_password" examples:"sjkdfhgs4536njbjjkh" validate:"required,eqfield=Password"`

	ProfileName string  `json:"profile_name" examples:"Иван" validate:"required,min=2,max=32"`
	Surname     string  `json:"surname" examples:"Иванов" validate:"required,min=2,max=32"`
	Patronymic  *string `json:"patronymic,omitempty" examples:"Иванович" validate:"omitempty,min=2,max=32"`

	Gender   Gender  `json:"gender" examples:"male" validate:"required,oneof=male female other"`
	Birthday *string `json:"birthday,omitempty" examples:"2000-05-14"`
	Bio      *string `json:"bio,omitempty" examples:"hi!" validate:"omitempty,max=256"`
}

// LoginRequest — Login может быть nickname, email или phone_number
type LoginRequest struct {
	Login    string `json:"login" examples:"ivan_petrov" validate:"required"`
	Password string `json:"password" examples:"strongpassword123" validate:"required"`
}
