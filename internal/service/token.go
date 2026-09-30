package service

import (
	"crypto/rand"
	"encoding/hex"
)

// randomToken генерирует криптографически случайный токен.
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
