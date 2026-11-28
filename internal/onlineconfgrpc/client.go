package onlineconfgrpc

import (
	"context"
	"encoding/json"
	"time"

	onlineconfv1 "kvmm/api/gen/onlineconf/v1"
	"kvmm/internal/domain"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client получает снимки конфигурации из сервиса OnlineConf.
type Client struct {
	conn   *grpc.ClientConn
	client onlineconfv1.OnlineConfServiceClient
	token  string
}

// Dial создаёт gRPC-клиент OnlineConf.
func Dial(target, token string) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, client: onlineconfv1.NewOnlineConfServiceClient(conn), token: token}, nil
}

// Close закрывает gRPC-соединение.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Snapshot получает данные scope для локального кэширования продуктовым сервисом.
func (c *Client) Snapshot(ctx context.Context, scope domain.OnlineConfScope) (domain.OnlineConfSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, internalTokenHeader, c.token)
	response, err := c.client.GetSnapshot(ctx, &onlineconfv1.GetSnapshotRequest{Scope: string(scope)})
	if err != nil {
		return domain.OnlineConfSnapshot{}, err
	}
	snapshot := domain.OnlineConfSnapshot{
		Scope:   domain.OnlineConfScope(response.GetScope()),
		Configs: make(map[string]json.RawMessage, len(response.GetConfigs())),
		Flags:   make(map[string]domain.OnlineConfFlagSnapshot, len(response.GetFlags())),
	}
	for _, config := range response.GetConfigs() {
		snapshot.Configs[config.GetName()] = append(json.RawMessage(nil), config.GetValueJson()...)
	}
	for _, flag := range response.GetFlags() {
		snapshot.Flags[flag.GetName()] = domain.OnlineConfFlagSnapshot{
			Rollout:    int(flag.GetRollout()),
			IncludeIDs: append([]int64(nil), flag.GetIncludeIds()...),
			ExcludeIDs: append([]int64(nil), flag.GetExcludeIds()...),
		}
	}
	return snapshot, nil
}
