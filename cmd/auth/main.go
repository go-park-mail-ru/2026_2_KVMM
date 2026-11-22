package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	authv1 "kvmm/api/gen/auth/v1"
	"kvmm/internal/authgrpc"
	"kvmm/internal/config"
	authHandler "kvmm/internal/handler/auth"
	"kvmm/internal/logging"
	"kvmm/internal/mailer"
	"kvmm/internal/repository"
	authusecase "kvmm/internal/usecase/auth"
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
	ttl, err := cfg.Redis.TTLDuration()
	if err != nil {
		logger.Fatal("invalid Redis session TTL", zap.Error(err))
	}
	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	defer rdb.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	sessions, err := repository.NewRedisSessionRepository(rdb, ttl)
	if err == nil {
		err = sessions.Ping(ctx)
	}
	cancel()
	if err != nil {
		logger.Fatal("Redis unavailable", zap.Error(err))
	}
	dbctx, dbcancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := repository.NewPool(dbctx, cfg.DatabaseURL, logger, debug)
	dbcancel()
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer db.Close()
	profiles := repository.NewProfileRepository(db)
	sender := mailer.New(mailer.SMTPSettings{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From, FromName: cfg.SMTP.FromName, LogoURL: cfg.SMTP.LogoURL})
	uc := authusecase.New(profiles, sessions, sender)
	httpServer := &http.Server{Addr: cfg.Auth.HTTPAddr, Handler: authHandler.NewRouter(authHandler.NewAuthHandler(uc)), ReadHeaderTimeout: 5 * time.Second}
	listener, err := net.Listen("tcp", cfg.Auth.GRPCAddr)
	if err != nil {
		logger.Fatal("gRPC listen failed", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	authv1.RegisterSessionServiceServer(grpcServer, authgrpc.NewServer(sessions))
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.Error("gRPC server stopped", zap.Error(err))
		}
	}()
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("auth HTTP server failed", zap.Error(err))
		}
	}()
	logger.Info("auth service started", zap.String("http", cfg.Auth.HTTPAddr), zap.String("grpc", cfg.Auth.GRPCAddr))
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = httpServer.Shutdown(shutdown)
	grpcServer.GracefulStop()
}
