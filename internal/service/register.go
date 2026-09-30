package service

import (
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"kvmm/internal/domain"
	"kvmm/internal/store"
)

// Register регистрирует пользователя, создаёт сессию и возвращает профиль.
func (s *AuthService) Register(req domain.RegisterRequest) (domain.Profile, string, string, error) {
	password, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.Profile{}, "", "", err
	}
	now := time.Now().UTC()
	profile := domain.Profile{
		Nickname:    strings.TrimSpace(req.Nickname),
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		ProfileName: strings.TrimSpace(req.ProfileName),
		Surname:     strings.TrimSpace(req.Surname),
		Patronymic:  req.Patronymic,
		Gender:      req.Gender,
		Bio:         req.Bio,
		CreatedAt:   now,
	}
	if req.Birthday != nil && *req.Birthday != "" {
		birthday, parseErr := time.Parse("2006-01-02", *req.Birthday)
		if parseErr != nil {
			return domain.Profile{}, "", "", parseErr
		}
		profile.Birthday = &birthday
	}
	created, err := s.store.CreateUser(store.User{Profile: profile, PasswordHash: password})
	if err != nil {
		return domain.Profile{}, "", "", err
	}
	session, csrfToken := s.newSession(created.ID)
	return created, session, csrfToken, nil
}
