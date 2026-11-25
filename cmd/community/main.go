package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	communityv1 "kvmm/api/gen/community/v1"
	"kvmm/internal/authgrpc"
	"kvmm/internal/communitygrpc"
	"kvmm/internal/config"
	communityhandler "kvmm/internal/handler/community"
	"kvmm/internal/httpapi"
	"kvmm/internal/logging"
	"kvmm/internal/repository"
	communityusecase "kvmm/internal/usecase/community"
	sessionusecase "kvmm/internal/usecase/session"
	"kvmm/internal/usergrpc"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger, debug, err := logging.New(cfg.Logging.Mode)
	if err != nil {
		panic(err)
	}
	logging.SetDefault(logger)
	defer logger.Sync()
	target := func(addr, service string) string {
		if strings.HasPrefix(addr, ":") {
			return service + addr
		}
		return addr
	}
	sessions, err := authgrpc.Dial(target(cfg.Auth.GRPCAddr, "auth"))
	if err != nil {
		logger.Fatal("auth gRPC connection failed", zap.Error(err))
	}
	defer sessions.Close()
	profiles, err := usergrpc.Dial(target(cfg.User.GRPCAddr, "user"))
	if err != nil {
		logger.Fatal("user gRPC connection failed", zap.Error(err))
	}
	defer profiles.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := repository.NewPool(ctx, cfg.DatabaseURL, logger, debug)
	cancel()
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer db.Close()
	repo := repository.NewCommunityRepository(db, cfg.Media.Dir, cfg.Media.PublicURL)
	uc := communityusecase.NewCommunityUseCase(repo, sessionusecase.New(sessions), profiles)
	listener, err := net.Listen("tcp", cfg.Community.GRPCAddr)
	if err != nil {
		logger.Fatal("community gRPC listen failed", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	communityv1.RegisterCommunityServiceServer(grpcServer, communitygrpc.NewServer(repo))
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.Error("community gRPC stopped", zap.Error(err))
		}
	}()
	server := &http.Server{Addr: cfg.Community.HTTPAddr, Handler: httpapi.NewRouter(httpapi.Dependencies{Config: cfg, Communities: communityhandler.NewHandler(uc)}), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("community HTTP service failed", zap.Error(err))
		}
	}()
	logger.Info("community service started", zap.String("http", cfg.Community.HTTPAddr), zap.String("grpc", cfg.Community.GRPCAddr))
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		logger.Error("community HTTP shutdown failed", zap.Error(err))
	}
	grpcServer.GracefulStop()
}
