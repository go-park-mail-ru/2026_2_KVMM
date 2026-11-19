package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"kvmm/internal/config"
	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/chat"
	"kvmm/internal/handler/feed"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/handler/posts"
	"kvmm/internal/handler/profile"
	"kvmm/internal/handler/relationship"
	"kvmm/internal/httpapi"
	"kvmm/internal/logging"
	"kvmm/internal/mailer"
	"kvmm/internal/repository"
	authusecase "kvmm/internal/usecase/auth"
	chatusecase "kvmm/internal/usecase/chat"
	feedusecase "kvmm/internal/usecase/feed"
	postusecase "kvmm/internal/usecase/post"
	profileusecase "kvmm/internal/usecase/profile"
	relationshipusecase "kvmm/internal/usecase/relationship"
	sessionusecase "kvmm/internal/usecase/session"
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

	sessions := repository.NewSessionRepository()
	sessionReader := sessionusecase.New(sessions)
	profiles := repository.NewProfileRepository(database)
	postsRepository := repository.NewPostRepository(database)
	chats := repository.NewChatRepository(database)
	messages := repository.NewMessageRepository(database)
	relationships := repository.NewRelationshipRepository(database)
	avatars := repository.NewAvatarRepository(database, appConfig.Media.Dir, appConfig.Media.PublicURL)
	mailSender := mailer.New(mailer.SMTPSettings{
		Host:     appConfig.SMTP.Host,
		Port:     appConfig.SMTP.Port,
		Username: appConfig.SMTP.Username,
		Password: appConfig.SMTP.Password,
		From:     appConfig.SMTP.From,
		FromName: appConfig.SMTP.FromName,
		LogoURL:  appConfig.SMTP.LogoURL,
	})

	authUseCase := authusecase.New(profiles, sessions, mailSender)
	profileUseCase := profileusecase.New(profiles, avatars, sessionReader)
	feedUseCase := feedusecase.New(postsRepository)
	postUseCase := postusecase.New(postsRepository, sessionReader)
	chatUseCase := chatusecase.New(chats, messages, sessionReader)
	relationshipUseCase := relationshipusecase.New(relationships, sessionReader)

	logger.Info("server listening", zap.String("addr", appConfig.HTTP.Addr))

	server := http.Server{
		Addr: appConfig.HTTP.Addr,
		Handler: httpapi.NewRouter(httpapi.Dependencies{
			Config: appConfig,
			Auth: auth.NewAuthHandler(authUseCase, helpers.CookieSettings{
				Secure:   appConfig.Cookie.Secure,
				SameSite: appConfig.Cookie.SameSite,
			}),
			Chat:         chat.NewChatHandler(chatUseCase, sessionReader, appConfig.Frontend.Origin),
			Feed:         feed.NewFeedHandler(feedUseCase),
			Posts:        posts.NewPostHandler(postUseCase),
			Profile:      profile.NewProfileHandler(profileUseCase),
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
