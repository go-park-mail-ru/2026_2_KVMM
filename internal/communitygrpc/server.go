package communitygrpc

import (
	"context"
	"errors"

	communityv1 "kvmm/api/gen/community/v1"
	"kvmm/internal/domain"
	"kvmm/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type reader interface {
	CommunityIDByUID(context.Context, string) (int64, error)
	AuthorizePostWrite(context.Context, int64, string) (int64, string, error)
	ReadByIDs(context.Context, []int64) ([]domain.Community, error)
}

type Server struct {
	communityv1.UnimplementedCommunityServiceServer
	repository reader
}

func NewServer(repo reader) *Server { return &Server{repository: repo} }

func (s *Server) GetCommunityIDByUID(ctx context.Context, req *communityv1.GetCommunityIDByUIDRequest) (*communityv1.GetCommunityIDByUIDResponse, error) {
	id, err := s.repository.CommunityIDByUID(ctx, req.GetCommunityUid())
	if err != nil {
		return nil, grpcError(err)
	}
	return &communityv1.GetCommunityIDByUIDResponse{CommunityId: id}, nil
}

func (s *Server) AuthorizePostWrite(ctx context.Context, req *communityv1.AuthorizePostWriteRequest) (*communityv1.AuthorizePostWriteResponse, error) {
	id, name, err := s.repository.AuthorizePostWrite(ctx, req.GetProfileId(), req.GetCommunityUid())
	if err != nil {
		return nil, grpcError(err)
	}
	return &communityv1.AuthorizePostWriteResponse{CommunityId: id, CommunityName: name}, nil
}

func (s *Server) GetCommunitiesByID(ctx context.Context, req *communityv1.GetCommunitiesByIDRequest) (*communityv1.GetCommunitiesByIDResponse, error) {
	items, err := s.repository.ReadByIDs(ctx, req.GetCommunityIds())
	if err != nil {
		return nil, grpcError(err)
	}
	response := &communityv1.GetCommunitiesByIDResponse{Communities: make([]*communityv1.Community, 0, len(items))}
	for _, item := range items {
		avatarURL := ""
		if item.AvatarURL != nil {
			avatarURL = *item.AvatarURL
		}
		response.Communities = append(response.Communities, &communityv1.Community{
			CommunityId: item.ID,
			Uid:         item.UID,
			Name:        item.Name,
			AvatarUrl:   avatarURL,
		})
	}
	return response, nil
}

func grpcError(err error) error {
	switch {
	case errors.Is(err, repository.ErrCommunityNotFound):
		return status.Error(codes.NotFound, "community not found")
	case errors.Is(err, repository.ErrCommunityForbidden):
		return status.Error(codes.PermissionDenied, "community post permission denied")
	default:
		return status.Error(codes.Internal, "community service failure")
	}
}
