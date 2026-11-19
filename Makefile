.PHONY: restart swagger test-coverage deploy migrate-up migrate-dowm

# Restart containers with rebuilding of images
restart:
	docker compose down
	GIT_BRANCH=$$(git branch --show-current) GIT_COMMIT=$$(git rev-parse HEAD) docker compose up -d --build

# Generate Swagger API documentation
swagger:
	swag init -g doc.go -d internal/handler,internal/domain -o internal/docs --parseInternal --parseDependency --v3.1

# Run tests and generate coverage report
test:
	go test -coverpkg=./internal/... -coverprofile=coverage.out ./internal/...
	grep -v '/internal/mocks/' coverage.out > coverage-no-mocks.out
	go tool cover -func=coverage-no-mocks.out | tail -1

# Generate mocks
MOCKGEN := go run go.uber.org/mock/mockgen
mocks:
	$(MOCKGEN) -source=internal/usecase/auth/usecase.go \
		-destination=internal/mocks/auth/mock.go \
		-package=authmock

	$(MOCKGEN) -source=internal/usecase/chat/usecase.go \
		-destination=internal/mocks/chat/mock.go \
		-package=chatmock

	$(MOCKGEN) -source=internal/usecase/feed/usecase.go \
		-destination=internal/mocks/feed/mock.go \
		-package=feedmock

	$(MOCKGEN) -source=internal/usecase/post/usecase.go \
		-destination=internal/mocks/post/mock.go \
		-package=postmock

	$(MOCKGEN) -source=internal/usecase/profile/usecase.go \
		-destination=internal/mocks/profile/mock.go \
		-package=profilemock

	$(MOCKGEN) -source=internal/usecase/relationship/usecase.go \
		-destination=internal/mocks/relationship/mock.go \
		-package=relationshipmock

	$(MOCKGEN) -source=internal/usecase/session/usecase.go \
		-destination=internal/mocks/session/mock.go \
		-package=sessionmock

	gofmt -w internal/mocks

# Run database migrations
-include deploy/.env
MIGRATIONS_PATH ?= ./db/migrations

DATABASE_LOCAL_HOST_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_LOCAL_HOST_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_LOCAL_HOST_URL)" down
