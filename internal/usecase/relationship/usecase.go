package relationship

import (
	"kvmm/internal/usecase/session"
)

var (
	ErrUnauthorized = session.ErrUnauthorized
	ErrInvalidCSRF  = session.ErrInvalidCSRF
)

type UseCase struct {
	sessions *session.UseCase
}

// New создаёт usecase связей
func New(sessions *session.UseCase) *UseCase {
	return &UseCase{sessions: sessions}
}
