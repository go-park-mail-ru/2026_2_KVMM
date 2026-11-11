package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

// NewRequestID создаёт id запроса
func NewRequestID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(value)
}

// WithRequestID сохраняет ID запроса в context.Context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestID возвращает ID запроса из context.Context
func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}
