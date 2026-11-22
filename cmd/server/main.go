package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"kvmm/internal/config"
	"kvmm/internal/handler/relationship"
	"kvmm/internal/httpapi"
	"kvmm/internal/logging"
	"kvmm/internal/repository"
	relationshipusecase "kvmm/internal/usecase/relationship"
	sessionusecase "kvmm/internal/usecase/session"

	"kvmm/internal/authgrpc"
)

var (
	gitBranch = "unknown"
	gitCommit = "unknown"
)

// main запускает HTTP-сервер backend API.
func main() {
	appConfig, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger, debugDatabase, err := logging.New(appConfig.Logging.Mode)
	if err != nil {
		panic(err)
	}
	logging.SetDefault(logger)
	defer logger.Sync()
	logger.Info("application configuration",
		zap.String("git_branch", gitBranch),
		zap.String("git_commit", gitCommit),
		zap.Any("config", appConfig.SafeForLog()),
	)
	grpcTarget := appConfig.Auth.GRPCAddr
	if strings.HasPrefix(grpcTarget, ":") {
		grpcTarget = "auth" + grpcTarget
	}
	sessionClient, err := authgrpc.Dial(grpcTarget)
	if err != nil {
		logger.Fatal("auth gRPC connection failed", zap.Error(err))
	}
	defer sessionClient.Close()

	databaseURL := appConfig.DatabaseURL
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL is required")
	}
	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), 10*time.Second)
	database, err := repository.NewPool(databaseContext, databaseURL, logger, debugDatabase)
	cancelDatabase()
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer database.Close()

	sessionReader := sessionusecase.New(sessionClient)
	relationships := repository.NewRelationshipRepository(database)
	relationshipUseCase := relationshipusecase.New(relationships, sessionReader)

	logger.Info("server listening", zap.String("addr", appConfig.HTTP.Addr))

	server := http.Server{
		Addr: appConfig.HTTP.Addr,
		Handler: httpapi.NewRouter(httpapi.Dependencies{
			Config:       appConfig,
			Relationship: relationship.NewRelationshipHandler(relationshipUseCase),
		}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server error", zap.Error(err))
		}
	case signal := <-stop:
		logger.Info("received signal", zap.String("signal", signal.String()))
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			logger.Error("graceful shutdown failed", zap.Error(err))
		}
	}
}
