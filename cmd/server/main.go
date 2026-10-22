package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/chat"
	"kvmm/internal/handler/feed"
	"kvmm/internal/handler/posts"
	"kvmm/internal/handler/profile"

	"kvmm/internal/handler/relationship"
	"kvmm/internal/httpapi"
	"kvmm/internal/repository"
	authusecase "kvmm/internal/usecase/auth"
	chatusecase "kvmm/internal/usecase/chat"
	feedusecase "kvmm/internal/usecase/feed"
	postusecase "kvmm/internal/usecase/post"
	profileusecase "kvmm/internal/usecase/profile"

	relationshipusecase "kvmm/internal/usecase/relationship"
	sessionusecase "kvmm/internal/usecase/session"
)

// main запускает HTTP-сервер backend API.
func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), 10*time.Second)
	database, err := repository.NewPool(databaseContext, databaseURL)
	cancelDatabase()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	sessions := repository.NewSessionRepository()
	sessionReader := sessionusecase.New(sessions)
	profiles := repository.NewProfileRepository(database)
	postsRepository := repository.NewPostRepository(database)
	chats := repository.NewChatRepository(database)
	messages := repository.NewMessageRepository(database)
	relationships := repository.NewRelationshipRepository(database)

	authUseCase := authusecase.New(profiles, sessions)
	profileUseCase := profileusecase.New(profiles, sessionReader)
	feedUseCase := feedusecase.New(postsRepository)
	postUseCase := postusecase.New(postsRepository, sessionReader)
	chatUseCase := chatusecase.New(chats, messages, sessionReader)
	relationshipUseCase := relationshipusecase.New(relationships, sessionReader)

	log.Printf("server listening on %s", addr)

	server := http.Server{
		Addr: addr,
		Handler: httpapi.NewRouter(httpapi.Dependencies{
			Auth:         auth.NewAuthHandler(authUseCase),
			Chat:         chat.NewChatHandler(chatUseCase),
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
			log.Fatalf("server error: %v", err)
		}
	case signal := <-stop:
		log.Printf("received signal %s, shutting down", signal)
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}
}
