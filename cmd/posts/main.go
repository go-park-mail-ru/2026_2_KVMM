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

	"kvmm/internal/authgrpc"
	"kvmm/internal/communitygrpc"
	"kvmm/internal/config"
	"kvmm/internal/handler/feed"
	"kvmm/internal/handler/posts"
	"kvmm/internal/httpapi"
	"kvmm/internal/logging"
	"kvmm/internal/repository"
	feedusecase "kvmm/internal/usecase/feed"
	postusecase "kvmm/internal/usecase/post"
	sessionusecase "kvmm/internal/usecase/session"
	"kvmm/internal/usergrpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger, debugDatabase, err := logging.New(cfg.Logging.Mode)
	if err != nil {
		panic(err)
	}
	logging.SetDefault(logger)
	defer logger.Sync()

	dialTarget := func(address, service string) string {
		if strings.HasPrefix(address, ":") {
			return service + address
		}
		return address
	}
	sessions, err := authgrpc.Dial(dialTarget(cfg.Auth.GRPCAddr, "auth"))
	if err != nil {
		logger.Fatal("auth gRPC connection failed", zap.Error(err))
	}
	defer sessions.Close()
	profiles, err := usergrpc.Dial(dialTarget(cfg.User.GRPCAddr, "user"))
	if err != nil {
		logger.Fatal("user gRPC connection failed", zap.Error(err))
	}
	defer profiles.Close()
	communities, err := communitygrpc.Dial(dialTarget(cfg.Community.GRPCAddr, "community"))
	if err != nil {
		logger.Fatal("community gRPC connection failed", zap.Error(err))
	}
	defer communities.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := repository.NewPool(ctx, cfg.DatabaseURL, logger, debugDatabase)
	cancel()
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer db.Close()

	sessionReader := sessionusecase.New(sessions)
	postRepository := repository.NewPostRepository(db, profiles, communities)
	feedUseCase := feedusecase.New(postRepository, sessionReader)
	postUseCase := postusecase.New(postRepository, sessionReader, communities)
	server := &http.Server{
		Addr: cfg.Posts.HTTPAddr,
		Handler: httpapi.NewRouter(httpapi.Dependencies{
			Config: cfg,
			Feed:   feed.NewFeedHandler(feedUseCase),
			Posts:  posts.NewPostHandler(postUseCase),
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("posts service failed", zap.Error(err))
		}
	}()
	logger.Info("posts service started", zap.String("addr", cfg.Posts.HTTPAddr))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdown); err != nil {
		logger.Error("posts service shutdown failed", zap.Error(err))
	}
}
