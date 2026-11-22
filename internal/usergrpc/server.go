package usergrpc

import (
	"context"
	"errors"

	userv1 "kvmm/api/gen/user/v1"
	"kvmm/internal/domain"
	"kvmm/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type profileReader interface {
	SearchByUUID(context.Context, string) (domain.Profile, error)
	ReadProfilesByIDs(context.Context, []int64) ([]domain.Profile, error)
}

type Server struct {
	userv1.UnimplementedUserServiceServer
	profiles profileReader
}

func NewServer(profiles profileReader) *Server { return &Server{profiles: profiles} }

func (s *Server) GetProfileByUID(ctx context.Context, req *userv1.GetProfileByUIDRequest) (*userv1.GetProfileByUIDResponse, error) {
	profile, err := s.profiles.SearchByUUID(ctx, req.GetUid())
	if err != nil {
		if errors.Is(err, repository.ErrProfileNotFound) {
			return nil, status.Error(codes.NotFound, "profile not found")
		}
		return nil, status.Error(codes.Internal, "profile lookup failed")
	}
	return &userv1.GetProfileByUIDResponse{
		ProfileId:   profile.ID,
		Uid:         profile.UID,
		Nickname:    profile.Nickname,
		ProfileName: value(profile.ProfileName),
		Surname:     value(profile.Surname),
		Patronymic:  value(profile.Patronymic),
	}, nil
}

func (s *Server) GetProfilesByID(ctx context.Context, req *userv1.GetProfilesByIDRequest) (*userv1.GetProfilesByIDResponse, error) {
	profiles, err := s.profiles.ReadProfilesByIDs(ctx, req.GetProfileIds())
	if err != nil {
		return nil, status.Error(codes.Internal, "profile lookup failed")
	}
	response := &userv1.GetProfilesByIDResponse{Profiles: make([]*userv1.Profile, 0, len(profiles))}
	for _, profile := range profiles {
		avatarURL := ""
		if len(profile.AvatarURLs) > 0 {
			avatarURL = profile.AvatarURLs[0]
		}
		response.Profiles = append(response.Profiles, &userv1.Profile{
			ProfileId:   profile.ID,
			Uid:         profile.UID,
			Nickname:    profile.Nickname,
			ProfileName: value(profile.ProfileName),
			Surname:     value(profile.Surname),
			Patronymic:  value(profile.Patronymic),
			AvatarUrl:   avatarURL,
		})
	}
	return response, nil
}

func value(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
