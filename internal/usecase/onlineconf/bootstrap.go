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

type adminRepository interface {
	CreateAdmin(context.Context, repository.CreateOnlineConfAdminParams) (domain.OnlineConfAdmin, error)
	ReadAdminByProfileID(context.Context, int64) (domain.OnlineConfAdmin, error)
}

// BootstrapResult содержит результат подготовки профиля и прав администратора
type BootstrapResult struct {
	Profile        domain.Profile
	ProfileCreated bool
	AdminCreated   bool
}

// UseCase содержит сценарии запуска OnlineConf
type UseCase struct {
	profiles profileRepository
	admins   adminRepository
}

// New создаёт usecase OnlineConf
func New(profiles profileRepository, admins adminRepository) *UseCase {
	return &UseCase{profiles: profiles, admins: admins}
}

// EnsureBootstrapAdmin создаёт профиль и выдаёт ему права администратора при первом запуске
func (u *UseCase) EnsureBootstrapAdmin(
	ctx context.Context,
	login string,
	password string,
) (BootstrapResult, error) {
	login = strings.TrimSpace(login)
	if !bootstrapNicknamePattern.MatchString(login) || len(login) < 4 || len(login) > 32 {
		return BootstrapResult{}, fmt.Errorf("ONLINECONF_ADMIN_LOGIN must be a Nexly nickname with 4 to 32 latin characters, digits, _ or .")
	}
	if len(password) < 8 || len(password) > 64 || !bootstrapPasswordPattern.MatchString(password) ||
		!bootstrapLetterPattern.MatchString(password) || !bootstrapDigitPattern.MatchString(password) {
		return BootstrapResult{}, errors.New("ONLINECONF_ADMIN_PASSWORD must contain 8 to 64 printable characters with latin letters and digits")
	}

	result := BootstrapResult{}
	profile, err := u.profiles.ReadByLoginAuth(ctx, login)
	switch {
	case err == nil:
		if bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(password)) != nil {
			return BootstrapResult{}, errors.New("ONLINECONF_ADMIN_PASSWORD does not match the existing bootstrap profile")
		}
		result.Profile = profile.Profile
	case errors.Is(err, repository.ErrProfileNotFound):
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return BootstrapResult{}, fmt.Errorf("hash bootstrap password: %w", hashErr)
		}
		created, createErr := u.profiles.Create(ctx, repository.CreateProfileParams{
			Nickname:     login,
			PasswordHash: string(hash),
		})
		if createErr == nil {
			result.Profile = created
			result.ProfileCreated = true
			break
		}
		if !errors.Is(createErr, repository.ErrProfileConflict) {
			return BootstrapResult{}, fmt.Errorf("create bootstrap profile: %w", createErr)
		}

		profile, readErr := u.profiles.ReadByLoginAuth(ctx, login)
		if readErr != nil {
			return BootstrapResult{}, fmt.Errorf("read concurrent bootstrap profile: %w", readErr)
		}
		if bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(password)) != nil {
			return BootstrapResult{}, errors.New("ONLINECONF_ADMIN_PASSWORD does not match the concurrent bootstrap profile")
		}
		result.Profile = profile.Profile
	default:
		return BootstrapResult{}, fmt.Errorf("read bootstrap profile: %w", err)
	}

	if result.Profile.ID <= 0 || result.Profile.UID == "" {
		return BootstrapResult{}, errors.New("bootstrap profile has invalid identity")
	}

	_, err = u.admins.ReadAdminByProfileID(ctx, result.Profile.ID)
	if err == nil {
		return result, nil
	}
	if !errors.Is(err, repository.ErrOnlineConfAdminNotFound) {
		return BootstrapResult{}, fmt.Errorf("read bootstrap admin: %w", err)
	}

	_, err = u.admins.CreateAdmin(ctx, repository.CreateOnlineConfAdminParams{
		ProfileID:  result.Profile.ID,
		ProfileUID: result.Profile.UID,
	})
	if err == nil {
		result.AdminCreated = true
		return result, nil
	}
	if !errors.Is(err, repository.ErrOnlineConfAdminConflict) {
		return BootstrapResult{}, fmt.Errorf("create bootstrap admin: %w", err)
	}

	if _, readErr := u.admins.ReadAdminByProfileID(ctx, result.Profile.ID); readErr != nil {
		return BootstrapResult{}, fmt.Errorf("read concurrent bootstrap admin: %w", readErr)
	}
	return result, nil
}
