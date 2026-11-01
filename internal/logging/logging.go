package logging

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
)

// New создаёт zap-логер
func New() (*zap.Logger, bool, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_MODE")))
	if mode == "" {
		mode = "prod"
	}

	var config zap.Config
	var debug bool
	switch mode {
	case "debug":
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		debug = true
	case "prod", "production":
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	default:
		return nil, false, fmt.Errorf("unsupported LOG_MODE %q: use debug or prod", mode)
	}

	logger, err := config.Build()
	return logger, debug, err
}
