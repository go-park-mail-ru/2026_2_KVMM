package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
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

// RequestIDFromHeader возвращает ID из заголовка или создаёт новый
func RequestIDFromHeader(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 64 {
		return NewRequestID()
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') && char != '-' && char != '_' {
			return NewRequestID()
		}
	}
	return value
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
