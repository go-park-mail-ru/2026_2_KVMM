// Package onlineconfgrpc содержит gRPC-доставку и клиент сервиса OnlineConf.
package onlineconfgrpc

import (
	"context"
	"crypto/subtle"
	"errors"
	"sort"

	onlineconfv1 "kvmm/api/gen/onlineconf/v1"
	"kvmm/internal/domain"
	onlineconfusecase "kvmm/internal/usecase/onlineconf"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const internalTokenHeader = "x-onlineconf-token"

type snapshotReader interface {
	Snapshot(context.Context, domain.OnlineConfScope) (domain.OnlineConfSnapshot, error)
}

// Server предоставляет продуктовым микросервисам снимки OnlineConf по gRPC.
type Server struct {
	onlineconfv1.UnimplementedOnlineConfServiceServer
	reader snapshotReader
	token  string
}

// NewServer создаёт gRPC-сервер OnlineConf с защитой внутренним токеном.
func NewServer(reader snapshotReader, token string) *Server {
	return &Server{reader: reader, token: token}
}

// GetSnapshot возвращает все конфигурации и feature flags указанного scope.
func (s *Server) GetSnapshot(ctx context.Context, req *onlineconfv1.GetSnapshotRequest) (*onlineconfv1.GetSnapshotResponse, error) {
	if !s.validToken(ctx) {
		return nil, status.Error(codes.Unauthenticated, "invalid internal token")
	}
	snapshot, err := s.reader.Snapshot(ctx, domain.OnlineConfScope(req.GetScope()))
	if err != nil {
		if errors.Is(err, onlineconfusecase.ErrInvalidEntry) {
			return nil, status.Error(codes.InvalidArgument, "invalid scope")
		}
		return nil, status.Error(codes.Internal, "onlineconf snapshot failed")
	}

	response := &onlineconfv1.GetSnapshotResponse{Scope: string(snapshot.Scope)}
	configNames := make([]string, 0, len(snapshot.Configs))
	for name := range snapshot.Configs {
		configNames = append(configNames, name)
	}
	sort.Strings(configNames)
	for _, name := range configNames {
		response.Configs = append(response.Configs, &onlineconfv1.Config{
			Name:      name,
			ValueJson: append([]byte(nil), snapshot.Configs[name]...),
		})
	}
	flagNames := make([]string, 0, len(snapshot.Flags))
	for name := range snapshot.Flags {
		flagNames = append(flagNames, name)
	}
	sort.Strings(flagNames)
	for _, name := range flagNames {
		flag := snapshot.Flags[name]
		response.Flags = append(response.Flags, &onlineconfv1.FeatureFlag{
			Name:       name,
			Rollout:    int32(flag.Rollout),
			IncludeIds: flag.IncludeIDs,
			ExcludeIds: flag.ExcludeIDs,
		})
	}
	return response, nil
}

func (s *Server) validToken(ctx context.Context) bool {
	if s.token == "" {
		return false
	}
	values := metadata.ValueFromIncomingContext(ctx, internalTokenHeader)
	for _, value := range values {
		if subtle.ConstantTimeCompare([]byte(value), []byte(s.token)) == 1 {
			return true
		}
	}
	return false
}
