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

	userv1 "kvmm/api/gen/user/v1"
	"kvmm/internal/usergrpc"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"kvmm/internal/authgrpc"
	"kvmm/internal/config"
	"kvmm/internal/handler/chat"
	"kvmm/internal/handler/profile"
	searchhandler "kvmm/internal/handler/search"
	"kvmm/internal/logging"
	"kvmm/internal/repository"
	chatusecase "kvmm/internal/usecase/chat"
	profileusecase "kvmm/internal/usecase/profile"
	sessionusecase "kvmm/internal/usecase/session"
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
	target := cfg.Auth.GRPCAddr
	if strings.HasPrefix(target, ":") {
		target = "auth" + target
	}
	sessions, err := authgrpc.Dial(target)
	if err != nil {
		logger.Fatal("auth gRPC connection failed", zap.Error(err))
	}
	defer sessions.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := repository.NewPool(ctx, cfg.DatabaseURL, logger, debug)
	cancel()
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer db.Close()
	sessionReader := sessionusecase.New(sessions)
	profiles := repository.NewProfileRepository(db)
	grpcListener, err := net.Listen("tcp", cfg.User.GRPCAddr)
	if err != nil {
		logger.Fatal("user gRPC listen failed", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	userv1.RegisterUserServiceServer(grpcServer, usergrpc.NewServer(profiles))
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Error("user gRPC server stopped", zap.Error(err))
		}
	}()
	avatars := repository.NewAvatarRepository(db, cfg.Media.Dir, cfg.Media.PublicURL)
	chats := repository.NewChatRepository(db)
	messages := repository.NewMessageRepository(db)
	profileUC := profileusecase.New(profiles, avatars, sessionReader)
	chatUC := chatusecase.New(chats, messages, sessionReader)
	elasticClient, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{cfg.Elasticsearch.URL}})
	if err != nil {
		logger.Fatal("elasticsearch client failed", zap.Error(err))
	}
	searchHandler := searchhandler.NewHandler(elasticClient)
	searchCtx, searchCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if err := searchHandler.ResetIndex(searchCtx, "profiles"); err != nil {
		searchCancel()
		logger.Fatal("profile search index reset failed", zap.Error(err))
	}
	const searchPageSize = searchhandler.BackfillPageSize
	profileCount := 0
	for offset := 0; ; offset += searchPageSize {
		profilesForSearch, err := profiles.ListForSearch(searchCtx, offset, searchPageSize)
		if err != nil {
			searchCancel()
			logger.Fatal("profile search backfill query failed", zap.Error(err))
		}
		if len(profilesForSearch) == 0 {
			break
		}
		if err := searchHandler.IndexProfiles(searchCtx, profilesForSearch); err != nil {
			searchCancel()
			logger.Fatal("profile search index backfill failed", zap.Error(err))
		}
		profileCount += len(profilesForSearch)
		if len(profilesForSearch) < searchPageSize {
			break
		}
	}
	searchCancel()
	logger.Info("profile search index backfill completed", zap.Int("profiles", profileCount), zap.Int("page_size", searchPageSize))
	router := http.NewServeMux()
	profileHTTPHandler := profile.NewProfileHandler(profileUC)
	profileHTTPHandler.SetSearchClient(searchHandler)
	chatHTTPHandler := chat.NewChatHandler(chatUC, sessionReader, cfg.Frontend.Origin)
	userRouter := newUserRouter(profileHTTPHandler, chatHTTPHandler, searchHandler, cfg)
	router.Handle("/", userRouter)
	server := &http.Server{Addr: cfg.User.HTTPAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("user service failed", zap.Error(err))
		}
	}()
	logger.Info("user service started", zap.String("addr", cfg.User.HTTPAddr), zap.String("grpc", cfg.User.GRPCAddr))
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = server.Shutdown(shutdown)
	grpcServer.GracefulStop()
}

func newUserRouter(profileHandler *profile.ProfileHandler, chatHandler *chat.ChatHandler, searchHandler *searchhandler.Handler, cfg config.Config) http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/api/profile/update", profileHandler.Update).Methods(http.MethodPatch)
	router.HandleFunc("/api/profiles/find", profileHandler.Find).Methods(http.MethodGet)
	router.HandleFunc("/api/profiles/search", searchHandler.Profiles).Methods(http.MethodGet)
	router.HandleFunc("/api/profile/avatar", profileHandler.UploadAvatar).Methods(http.MethodPost)
	router.HandleFunc("/api/profile/avatars", profileHandler.ListAvatars).Methods(http.MethodGet)
	router.HandleFunc("/api/profile/avatars/{uid}", profileHandler.DeleteAvatar).Methods(http.MethodDelete)
	router.PathPrefix("/api/media/").Handler(http.StripPrefix("/api/media/", http.FileServer(http.Dir(cfg.Media.Dir))))
	router.HandleFunc("/api/chats/list", chatHandler.List).Methods(http.MethodGet)
	router.HandleFunc("/api/chats/messages", chatHandler.Messages).Methods(http.MethodGet)
	router.HandleFunc("/api/chats/send", chatHandler.Send).Methods(http.MethodPost)
	router.HandleFunc("/api/chats/create", chatHandler.Create).Methods(http.MethodPost)
	router.HandleFunc("/api/chats/ws", chatHandler.WebSocket).Methods(http.MethodGet)
	return router
}
