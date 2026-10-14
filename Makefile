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
test-coverage:
	go test -coverpkg=./... -coverprofile=coverage.out ./internal/handler
	go tool cover -func=coverage.out

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
		echo "======================================"; \
		echo ""; \
		echo "Create the file:"; \
		echo "	$(BACK_DIR)/deploy/.env"; \
		echo ""; \
		echo "Required variables:"; \
		echo "	HTTP_ADDR=:8080"; \
		echo "	FRONTEND_ORIGIN=http://161.104.106.151"; \
		echo "	FRONTEND_URL=http://161.104.106.151"; \
		echo "	COOKIE_SECURE=false"; \
		echo "	COOKIE_SAMESITE=lax"; \
		echo ""; \
		echo "Example:"; \
		echo "	nano $(BACK_DIR)/deploy/.env"; \
		echo "======================================"; \
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