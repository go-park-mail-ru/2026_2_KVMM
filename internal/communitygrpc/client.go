package communitygrpc

import (
	"context"
	"time"

	communityv1 "kvmm/api/gen/community/v1"
	"kvmm/internal/domain"
	"kvmm/internal/repository"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	conn   *grpc.ClientConn
	client communityv1.CommunityServiceClient
}

func Dial(target string) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, client: communityv1.NewCommunityServiceClient(conn)}, nil
}
func (c *Client) Close() error { return c.conn.Close() }
func (c *Client) CommunityIDByUID(ctx context.Context, uid string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	response, err := c.client.GetCommunityIDByUID(ctx, &communityv1.GetCommunityIDByUIDRequest{CommunityUid: uid})
	if err != nil {
		return 0, normalizeGRPCError(err)
	}
	return response.GetCommunityId(), nil
}
func (c *Client) AuthorizePostWrite(ctx context.Context, profileID int64, uid string) (int64, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	response, err := c.client.AuthorizePostWrite(ctx, &communityv1.AuthorizePostWriteRequest{ProfileId: profileID, CommunityUid: uid})
	if err != nil {
		return 0, "", normalizeGRPCError(err)
	}
	return response.GetCommunityId(), response.GetCommunityName(), nil
}

func normalizeGRPCError(err error) error {
	switch status.Code(err) {
	case codes.NotFound:
		return repository.ErrCommunityNotFound
	case codes.PermissionDenied:
		return repository.ErrCommunityForbidden
	default:
		return err
	}
}
func (c *Client) GetCommunitiesByIDs(ctx context.Context, ids []int64) ([]domain.Community, error) {
	if len(ids) == 0 {
		return []domain.Community{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	response, err := c.client.GetCommunitiesByID(ctx, &communityv1.GetCommunitiesByIDRequest{CommunityIds: ids})
	if err != nil {
		return nil, err
	}
	items := make([]domain.Community, 0, len(response.GetCommunities()))
	for _, item := range response.GetCommunities() {
		community := domain.Community{ID: item.GetCommunityId(), UID: item.GetUid(), Name: item.GetName()}
		if item.GetAvatarUrl() != "" {
			avatarURL := item.GetAvatarUrl()
			community.AvatarURL = &avatarURL
		}
		items = append(items, community)
	}
	return items, nil
}
