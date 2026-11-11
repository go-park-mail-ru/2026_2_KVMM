.PHONY: help restart swagger test-coverage deploy

help:
	@echo "---------------------------------------------------------------------------------"
	@echo "KVMM Makefile"
	@echo ""
	@echo "	make restart"
	@echo "		Rebuild Docker images and restart all containers"
	@echo ""
	@echo "	make swagger"
	@echo "		Generate Swagger API documentation"
	@echo ""
	@echo "	make test-coverage"
	@echo "		Run tests and generate a coverage report"
	@echo ""
	@echo "	make deploy FRONT_BRANCH=<branch> BACK_BRANCH=<branch>"
	@echo "		Update frontend and backend, check .env,"
	@echo "		rebuild Docker and start the application"
	@echo "		Examples:"
	@echo "			make deploy FRONT_BRANCH=develop BACK_BRANCH=feature/auth"
	@echo "---------------------------------------------------------------------------------"

# Restart containers with rebuilding of images
restart:
	docker compose down
	docker compose up -d --build

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
db-migrate:
	migrate \
		-path ./db/migrations \
		-database "$(DATABASE_URL)" \
		up

# Deploy
FRONT_DIR := /home/ubuntu/kvmm/FRONT_2026_2_KVMM
BACK_DIR := /home/ubuntu/kvmm/BACK_2026_2_KVMM
deploy:
	@if [ -z "$(FRONT_BRANCH)" ]; then \
		echo "Error: FRONT_BRANCH is not specified"; \
		echo "Usage: make deploy FRONT_BRANCH=<branch> BACK_BRANCH=<branch>"; \
		exit 1; \
	fi
	@if [ -z "$(BACK_BRANCH)" ]; then \
		echo "Error: BACK_BRANCH is not specified"; \
		echo "Usage: make deploy FRONT_BRANCH=<branch> BACK_BRANCH=<branch>"; \
		exit 1; \
	fi

	@echo "======================================"
	@echo "Deploy"
	@echo "Frontend branch: $(FRONT_BRANCH)"
	@echo "Backend branch:	$(BACK_BRANCH)"
	@echo "======================================"

	@echo ">>> Updating frontend..."
	cd $(FRONT_DIR) && \
	git checkout $(FRONT_BRANCH) && \
	git pull origin $(FRONT_BRANCH)

	@echo ">>> Updating backend..."
	cd $(BACK_DIR) && \
	git checkout $(BACK_BRANCH) && \
	git pull origin $(BACK_BRANCH)

	@echo ">>> Checking .env..."
	@if [ -f "$(BACK_DIR)/deploy/.env" ]; then \
		echo "======================================"; \
		echo "Backend .env:"; \
		echo "======================================"; \
		cat "$(BACK_DIR)/deploy/.env"; \
		echo "======================================"; \
	else \
		echo "======================================"; \
		echo "ERROR: .env file not found!"; \
		exit 1; \
	fi

	@echo ">>> Building and starting Docker..."
	cd $(BACK_DIR) && \
	$(COMPOSE) up -d --build

	@echo ">>> Removing unused Docker images..."
	docker image prune -f

	@echo ">>> Docker status..."
	cd $(BACK_DIR) && \
	$(COMPOSE) ps

	@echo "======================================"
	@echo "Deploy completed!"
	@echo "======================================"