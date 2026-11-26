package community

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"kvmm/internal/domain"
	"kvmm/internal/logging"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

var (
	ErrUnauthorized            = sessionusecase.ErrUnauthorized
	ErrInvalidCSRF             = sessionusecase.ErrInvalidCSRF
	ErrCommunityNotFound       = repository.ErrCommunityNotFound
	ErrCommunityForbidden      = repository.ErrCommunityForbidden
	ErrCommunityMemberNotFound = repository.ErrCommunityMemberNotFound
	ErrCommunityConflict       = repository.ErrCommunityConflict
	ErrProfileNotFound         = repository.ErrProfileNotFound
	ErrInvalidCommunity        = errors.New("invalid community data")
	ErrInvalidAvatar           = repository.ErrInvalidAvatar
)

type communityRepository interface {
	Create(context.Context, int64, domain.CreateCommunityRequest) (domain.Community, error)
	Read(context.Context, string) (domain.Community, error)
	IsSubscribed(context.Context, string, int64) (bool, error)
	List(context.Context, int, int) (domain.CommunityPage, error)
	ListSubscriptions(context.Context, int64, int, int) (domain.CommunityPage, error)
	Update(context.Context, int64, string, domain.UpdateCommunityRequest) (domain.Community, error)
	Delete(context.Context, int64, string) error
	Subscribe(context.Context, int64, string) error
	Unsubscribe(context.Context, int64, string) error
	AssignAdmin(context.Context, int64, int64, string) error
	CreateAvatar(context.Context, int64, string, domain.UploadAvatarRequest) (domain.Avatar, error)
}

type profileIDReader interface {
	GetProfileIDByUID(context.Context, string) (int64, error)
}

type CommunityUseCase struct {
	communities communityRepository
	sessions    *sessionusecase.UseCase
	profiles    profileIDReader
}

func NewCommunityUseCase(communities communityRepository, sessions *sessionusecase.UseCase, profiles profileIDReader) *CommunityUseCase {
	return &CommunityUseCase{communities: communities, sessions: sessions, profiles: profiles}
}

func (u *CommunityUseCase) Create(ctx context.Context, session, csrf string, req domain.CreateCommunityRequest) (domain.Community, error) {
	userID, err := u.sessions.Authorize(session, csrf)
	if err != nil {
		return domain.Community{}, err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Bio = strings.TrimSpace(req.Bio)
	if utf8.RuneCountInString(req.Name) < 2 || utf8.RuneCountInString(req.Name) > 128 || utf8.RuneCountInString(req.Bio) > 256 {
		return domain.Community{}, ErrInvalidCommunity
	}
	community, err := u.communities.Create(ctx, userID, req)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "create", err)
	}
	return community, err
}

func (u *CommunityUseCase) Read(ctx context.Context, uid, session string) (domain.Community, error) {
	community, err := u.communities.Read(ctx, uid)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "read", err)
		return community, err
	}
	if session != "" {
		profileID, _, readErr := u.sessions.Read(session)
		if readErr != nil {
			return domain.Community{}, readErr
		}
		community.IsSubscribed, err = u.communities.IsSubscribed(ctx, uid, profileID)
		if err != nil {
			return domain.Community{}, err
		}
	}
	return community, err
}

func (u *CommunityUseCase) List(ctx context.Context, offset, limit int) (domain.CommunityPage, error) {
	page, err := u.communities.List(ctx, offset, limit)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "list", err)
	}
	return page, err
}

func (u *CommunityUseCase) Update(ctx context.Context, session, csrf, uid string, req domain.UpdateCommunityRequest) (domain.Community, error) {
	userID, err := u.sessions.Authorize(session, csrf)
	if err != nil {
		return domain.Community{}, err
	}
	if req.Name == nil && req.Bio == nil {
		return domain.Community{}, ErrInvalidCommunity
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 128 {
			return domain.Community{}, ErrInvalidCommunity
		}
		req.Name = &name
	}
	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if utf8.RuneCountInString(bio) > 256 {
			return domain.Community{}, ErrInvalidCommunity
		}
		req.Bio = &bio
	}
	community, err := u.communities.Update(ctx, userID, uid, req)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "update", err)
	}
	return community, err
}

func (u *CommunityUseCase) Delete(ctx context.Context, session, csrf, uid string) error {
	userID, err := u.sessions.Authorize(session, csrf)
	if err != nil {
		return err
	}
	if err = u.communities.Delete(ctx, userID, uid); err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "delete", err)
	}
	return err
}

func (u *CommunityUseCase) Subscribe(ctx context.Context, session, csrf, uid string) error {
	userID, err := u.sessions.Authorize(session, csrf)
	if err != nil {
		return err
	}
	if err = u.communities.Subscribe(ctx, userID, uid); err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "subscribe", err)
	}
	return err
}

func (u *CommunityUseCase) Unsubscribe(ctx context.Context, session, csrf, uid string) error {
	userID, err := u.sessions.Authorize(session, csrf)
	if err != nil {
		return err
	}
	if err = u.communities.Unsubscribe(ctx, userID, uid); err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "unsubscribe", err)
	}
	return err
}

func (u *CommunityUseCase) AssignAdmin(ctx context.Context, session, csrf, communityUID, profileUID string) error {
	ownerID, err := u.sessions.Authorize(session, csrf)
	if err != nil {
		return err
	}
	profileID, err := u.profiles.GetProfileIDByUID(ctx, profileUID)
	if err != nil {
		return err
	}
	if err = u.communities.AssignAdmin(ctx, ownerID, profileID, communityUID); err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "assign admin", err)
	}
	return err
}

// UploadAvatar загружает изображение сообщества от имени владельца или администратора.
func (u *CommunityUseCase) UploadAvatar(ctx context.Context, session, csrf, communityUID string, req domain.UploadAvatarRequest) (domain.Avatar, error) {
	profileID, err := u.sessions.Authorize(session, csrf)
	if err != nil {
		return domain.Avatar{}, err
	}
	avatar, err := u.communities.CreateAvatar(ctx, profileID, communityUID, req)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "upload avatar", err)
	}
	return avatar, err
}

func (u *CommunityUseCase) Subscriptions(ctx context.Context, session string, offset, limit int) (domain.CommunityPage, error) {
	profileID, _, err := u.sessions.Read(session)
	if err != nil {
		return domain.CommunityPage{}, err
	}
	page, err := u.communities.ListSubscriptions(ctx, profileID, offset, limit)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "community", "list subscriptions", err)
	}
	return page, err
}
