// Package logging содержит общую настройку логирования приложения
package logging

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
)

var defaultLogger = zap.NewNop()

// SetDefault сохраняет общий логер приложения для обработчиков и usecase
func SetDefault(logger *zap.Logger) {
	if logger != nil {
		defaultLogger = logger
	}
}

// Logger возвращает общий логер приложения
func Logger() *zap.Logger {
	return defaultLogger
}

// LogUseCaseError записывает ошибку usecase на debug-уровне
func LogUseCaseError(component, operation string, err error) {
	if err == nil {
		return
	}
	defaultLogger.Debug("usecase operation failed",
		zap.String("component", component),
		zap.String("operation", operation),
		zap.Error(err),
	)
}

// LogUseCaseErrorContext записывает ошибку usecase вместе с ID запроса
func LogUseCaseErrorContext(ctx context.Context, component, operation string, err error) {
	if err == nil {
		return
	}
	LoggerWithContext(ctx).Debug("usecase operation failed",
		zap.String("component", component),
		zap.String("operation", operation),
		zap.Error(err),
	)
}

// LoggerWithContext возвращает логер с полями из context.Context
func LoggerWithContext(ctx context.Context) *zap.Logger {
	if requestID := RequestID(ctx); requestID != "" {
		return defaultLogger.With(zap.String("request_id", requestID))
	}
	return defaultLogger
}

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
