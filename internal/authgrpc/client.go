package authgrpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	authv1 "kvmm/api/gen/auth/v1"
	"kvmm/internal/repository"
)

type Client struct {
	conn   *grpc.ClientConn
	client authv1.SessionServiceClient
}

func Dial(target string) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, client: authv1.NewSessionServiceClient(conn)}, nil
}
func (c *Client) Close() error { return c.conn.Close() }
func (c *Client) Read(token string) (int64, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := c.client.ReadSession(ctx, &authv1.ReadSessionRequest{Token: token})
	if err != nil {
		if status.Code(err) == codes.Unauthenticated {
			return 0, "", repository.ErrSessionNotFound
		}
		return 0, "", err
	}
	return response.GetUserId(), response.GetCsrfToken(), nil
}
func (c *Client) ValidateCSRF(token, csrf string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := c.client.ValidateCSRF(ctx, &authv1.ValidateCSRFRequest{Token: token, CsrfToken: csrf})
	if err != nil {
		return err
	}
	if !response.GetValid() {
		return repository.ErrInvalidCSRF
	}
	return nil
}
