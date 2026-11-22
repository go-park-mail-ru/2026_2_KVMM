package usergrpc

import (
	"context"
	"time"

	userv1 "kvmm/api/gen/user/v1"
	"kvmm/internal/domain"
	"kvmm/internal/repository"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	conn   *grpc.ClientConn
	client userv1.UserServiceClient
}

func Dial(target string) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, client: userv1.NewUserServiceClient(conn)}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) GetProfileIDByUID(ctx context.Context, uid string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	profile, err := c.client.GetProfileByUID(ctx, &userv1.GetProfileByUIDRequest{Uid: uid})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return 0, repository.ErrProfileNotFound
		}
		return 0, err
	}
	return profile.GetProfileId(), nil
}

func (c *Client) GetProfilesByIDs(ctx context.Context, ids []int64) ([]domain.Profile, error) {
	if len(ids) == 0 {
		return []domain.Profile{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	response, err := c.client.GetProfilesByID(ctx, &userv1.GetProfilesByIDRequest{ProfileIds: ids})
	if err != nil {
		return nil, err
	}
	profiles := make([]domain.Profile, 0, len(response.GetProfiles()))
	for _, item := range response.GetProfiles() {
		profile := domain.Profile{ID: item.GetProfileId(), UID: item.GetUid(), Nickname: item.GetNickname()}
		if item.GetProfileName() != "" {
			profile.ProfileName = stringPointer(item.GetProfileName())
		}
		if item.GetSurname() != "" {
			profile.Surname = stringPointer(item.GetSurname())
		}
		if item.GetPatronymic() != "" {
			profile.Patronymic = stringPointer(item.GetPatronymic())
		}
		if item.GetAvatarUrl() != "" {
			profile.AvatarURLs = []string{item.GetAvatarUrl()}
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func stringPointer(value string) *string { return &value }
