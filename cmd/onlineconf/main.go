package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kvmm/internal/config"
	"kvmm/internal/logging"
	"kvmm/internal/repository"
	onlineconfusecase "kvmm/internal/usecase/onlineconf"

	"go.uber.org/zap"
)

// healthResponse описывает ответ проверки работоспособности сервиса.
type healthResponse struct {
	Status string `json:"status"`
}

// main запускает сервис OnlineConf и создаёт первоначального администратора Nexly.
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	database, err := repository.NewPool(ctx, cfg.DatabaseURL, logger, debugDatabase)
	cancel()
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer database.Close()

	bootstrap := onlineconfusecase.New(
		repository.NewProfileRepository(database),
		repository.NewOnlineConfRepository(database),
	)
	bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 10*time.Second)
	result, err := bootstrap.EnsureBootstrapAdmin(
		bootstrapCtx,
		cfg.OnlineConf.BootstrapAdminLogin,
		cfg.OnlineConf.BootstrapAdminPassword,
	)
	bootstrapCancel()
	if err != nil {
		logger.Fatal("onlineconf bootstrap admin failed", zap.Error(err))
	}
	if result.ProfileCreated {
		logger.Info("onlineconf bootstrap Nexly profile created",
			zap.String("nickname", result.Profile.Nickname),
			zap.Int64("profile_id", result.Profile.ID),
		)
	} else {
		logger.Info("onlineconf bootstrap Nexly profile already exists",
			zap.String("nickname", result.Profile.Nickname),
			zap.Int64("profile_id", result.Profile.ID),
		)
	}
	if result.AdminCreated {
		logger.Info("onlineconf bootstrap admin access granted", zap.Int64("profile_id", result.Profile.ID))
	} else {
		logger.Info("onlineconf bootstrap admin already exists", zap.Int64("profile_id", result.Profile.ID))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	})
	server := &http.Server{
		Addr:              cfg.OnlineConf.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("onlineconf service failed", zap.Error(err))
		}
	}()
	logger.Info("onlineconf service started", zap.String("addr", cfg.OnlineConf.HTTPAddr))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("onlineconf service shutdown failed", zap.Error(err))
	}
}
