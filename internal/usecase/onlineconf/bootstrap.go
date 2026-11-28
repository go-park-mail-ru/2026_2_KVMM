package onlineconf

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	bootstrapNicknamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.]*$`)
	bootstrapPasswordPattern = regexp.MustCompile(`^[\x21-\x7E]+$`)
	bootstrapLetterPattern   = regexp.MustCompile(`[A-Za-z]`)
	bootstrapDigitPattern    = regexp.MustCompile(`[0-9]`)
)

type profileRepository interface {
	Create(context.Context, repository.CreateProfileParams) (domain.Profile, error)
	ReadByLoginAuth(context.Context, string) (repository.AuthProfile, error)
}

// BootstrapResult содержит профиль единственного bootstrap-администратора.
type BootstrapResult struct {
	Profile ProfileIdentity
	Created bool
}

// ProfileIdentity содержит внутренний ID и nickname bootstrap-профиля.
type ProfileIdentity struct {
	ID       int64
	Nickname string
}

// Bootstrapper создаёт и находит обычный Nexly-профиль администратора OnlineConf.
type Bootstrapper struct {
	profiles profileRepository
}

// NewBootstrapper создаёт usecase первоначального администратора OnlineConf.
func NewBootstrapper(profiles profileRepository) *Bootstrapper {
	return &Bootstrapper{profiles: profiles}
}

// EnsureProfile создаёт обычный Nexly-профиль при первом запуске и возвращает его при повторных.
func (u *Bootstrapper) EnsureProfile(ctx context.Context, login, password string) (BootstrapResult, error) {
	login = strings.TrimSpace(login)
	if !bootstrapNicknamePattern.MatchString(login) || len(login) < 4 || len(login) > 32 {
		return BootstrapResult{}, fmt.Errorf("ONLINECONF_ADMIN_LOGIN must be a Nexly nickname with 4 to 32 latin characters, digits, _ or .")
	}
	if len(password) < 8 || len(password) > 64 || !bootstrapPasswordPattern.MatchString(password) ||
		!bootstrapLetterPattern.MatchString(password) || !bootstrapDigitPattern.MatchString(password) {
		return BootstrapResult{}, errors.New("ONLINECONF_ADMIN_PASSWORD must contain 8 to 64 printable characters with latin letters and digits")
	}

	profile, err := u.profiles.ReadByLoginAuth(ctx, login)
	if err == nil {
		if bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(password)) != nil {
			return BootstrapResult{}, errors.New("ONLINECONF_ADMIN_PASSWORD does not match the existing bootstrap profile")
		}
		return BootstrapResult{Profile: ProfileIdentity{ID: profile.Profile.ID, Nickname: profile.Profile.Nickname}}, nil
	}
	if !errors.Is(err, repository.ErrProfileNotFound) {
		return BootstrapResult{}, fmt.Errorf("read bootstrap profile: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("hash bootstrap password: %w", err)
	}
	created, err := u.profiles.Create(ctx, repository.CreateProfileParams{Nickname: login, PasswordHash: string(hash)})
	if err == nil {
		return BootstrapResult{Profile: ProfileIdentity{ID: created.ID, Nickname: created.Nickname}, Created: true}, nil
	}
	if !errors.Is(err, repository.ErrProfileConflict) {
		return BootstrapResult{}, fmt.Errorf("create bootstrap profile: %w", err)
	}

	profile, err = u.profiles.ReadByLoginAuth(ctx, login)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("read concurrent bootstrap profile: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(password)) != nil {
		return BootstrapResult{}, errors.New("ONLINECONF_ADMIN_PASSWORD does not match the concurrent bootstrap profile")
	}
	return BootstrapResult{Profile: ProfileIdentity{ID: profile.Profile.ID, Nickname: profile.Profile.Nickname}}, nil
}
