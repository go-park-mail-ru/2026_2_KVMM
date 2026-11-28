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

	onlineconfv1 "kvmm/api/gen/onlineconf/v1"
	"kvmm/internal/authgrpc"
	"kvmm/internal/config"
	onlineconfhandler "kvmm/internal/handler/onlineconf"
	"kvmm/internal/logging"
	"kvmm/internal/onlineconfgrpc"
	"kvmm/internal/repository"
	onlineconfusecase "kvmm/internal/usecase/onlineconf"
	sessionusecase "kvmm/internal/usecase/session"
	"kvmm/internal/usergrpc"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// main запускает сервис OnlineConf и подготавливает bootstrap-администратора.
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
	if strings.TrimSpace(cfg.OnlineConf.InternalToken) == "" {
		logger.Fatal("onlineconf internal token is required")
	}

	dialTarget := func(address, service string) string {
		if strings.HasPrefix(address, ":") {
			return service + address
		}
		return address
	}
	authClient, err := authgrpc.Dial(dialTarget(cfg.Auth.GRPCAddr, "auth"))
	if err != nil {
		logger.Fatal("auth gRPC connection failed", zap.Error(err))
	}
	defer authClient.Close()
	profilesClient, err := usergrpc.Dial(dialTarget(cfg.User.GRPCAddr, "user"))
	if err != nil {
		logger.Fatal("user gRPC connection failed", zap.Error(err))
	}
	defer profilesClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	database, err := repository.NewPool(ctx, cfg.DatabaseURL, logger, debugDatabase)
	cancel()
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer database.Close()

	bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 10*time.Second)
	bootstrap, err := onlineconfusecase.NewBootstrapper(repository.NewProfileRepository(database)).EnsureProfile(
		bootstrapCtx,
		cfg.OnlineConf.BootstrapAdminLogin,
		cfg.OnlineConf.BootstrapAdminPassword,
	)
	bootstrapCancel()
	if err != nil {
		logger.Fatal("onlineconf bootstrap profile failed", zap.Error(err))
	}
	if bootstrap.Created {
		logger.Info("onlineconf bootstrap Nexly profile created",
			zap.String("nickname", bootstrap.Profile.Nickname),
			zap.Int64("profile_id", bootstrap.Profile.ID),
		)
	} else {
		logger.Info("onlineconf bootstrap Nexly profile already exists",
			zap.String("nickname", bootstrap.Profile.Nickname),
			zap.Int64("profile_id", bootstrap.Profile.ID),
		)
	}

	onlineConfRepository := repository.NewOnlineConfRepository(database)
	adminUseCase := onlineconfusecase.NewAdminUseCase(
		onlineConfRepository,
		sessionusecase.New(authClient),
		profilesClient,
		bootstrap.Profile.ID,
	)
	grpcListener, err := net.Listen("tcp", cfg.OnlineConf.GRPCAddr)
	if err != nil {
		logger.Fatal("onlineconf gRPC listen failed", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	onlineconfv1.RegisterOnlineConfServiceServer(
		grpcServer,
		onlineconfgrpc.NewServer(adminUseCase, cfg.OnlineConf.InternalToken),
	)
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Error("onlineconf gRPC server stopped", zap.Error(err))
		}
	}()

	server := &http.Server{
		Addr:              cfg.OnlineConf.HTTPAddr,
		Handler:           onlineconfhandler.NewRouter(onlineconfhandler.NewHandler(adminUseCase, cfg.OnlineConf.InternalToken)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("onlineconf service failed", zap.Error(err))
		}
	}()
	logger.Info("onlineconf service started",
		zap.String("addr", cfg.OnlineConf.HTTPAddr),
		zap.String("grpc", cfg.OnlineConf.GRPCAddr),
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("onlineconf service shutdown failed", zap.Error(err))
	}
	grpcServer.GracefulStop()
}
