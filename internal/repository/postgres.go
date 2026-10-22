package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"go.uber.org/zap"
)

// NewPool создаёт и проверяет пул соединений с PostgreSQL
func NewPool(ctx context.Context, databaseURL string, logger *zap.Logger, debug bool) (*pgxpool.Pool, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("database URL is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 10
	if logger != nil {
		logLevel := tracelog.LogLevelError
		if debug {
			logLevel = tracelog.LogLevelTrace
		}
		config.ConnConfig.Tracer = &tracelog.TraceLog{
			Logger:   newPGXLogger(logger),
			LogLevel: logLevel,
		}
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func newPGXLogger(logger *zap.Logger) tracelog.Logger {
	return tracelog.LoggerFunc(func(ctx context.Context, level tracelog.LogLevel, message string, data map[string]any) {
		fields := make([]zap.Field, 0, len(data)+1)
		fields = append(fields, zap.String("component", "postgres"))
		for key, value := range data {
			fields = append(fields, zap.Any(key, value))
		}

		switch level {
		case tracelog.LogLevelTrace, tracelog.LogLevelDebug:
			logger.Debug(message, fields...)
		case tracelog.LogLevelInfo:
			logger.Info(message, fields...)
		case tracelog.LogLevelWarn:
			logger.Warn(message, fields...)
		default:
			logger.Error(message, fields...)
		}
	})
}
