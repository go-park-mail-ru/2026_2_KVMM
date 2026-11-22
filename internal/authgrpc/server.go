package authgrpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	authv1 "kvmm/api/gen/auth/v1"
	"kvmm/internal/repository"
)

type sessionReader interface {
	Read(string) (int64, string, error)
	ValidateCSRF(string, string) error
}

type Server struct {
	authv1.UnimplementedSessionServiceServer
	sessions sessionReader
}

func NewServer(sessions sessionReader) *Server { return &Server{sessions: sessions} }

func (s *Server) ReadSession(_ context.Context, req *authv1.ReadSessionRequest) (*authv1.ReadSessionResponse, error) {
	userID, csrf, err := s.sessions.Read(req.GetToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "session not found")
	}
	return &authv1.ReadSessionResponse{UserId: userID, CsrfToken: csrf}, nil
}

func (s *Server) ValidateCSRF(_ context.Context, req *authv1.ValidateCSRFRequest) (*authv1.ValidateCSRFResponse, error) {
	if err := s.sessions.ValidateCSRF(req.GetToken(), req.GetCsrfToken()); err != nil {
		if err == repository.ErrInvalidCSRF || err == repository.ErrSessionNotFound {
			return &authv1.ValidateCSRFResponse{Valid: false}, nil
		}
		return nil, status.Error(codes.Internal, "session validation failed")
	}
	return &authv1.ValidateCSRFResponse{Valid: true}, nil
}
