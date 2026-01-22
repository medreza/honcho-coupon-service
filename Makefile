SERVICE_NAME=honcho-coupon-service
DOCKER_COMPOSE=docker compose
MAIN_PATH=./cmd/server/main.go

.PHONY: build run test

build:
	@echo "Building $(SERVICE_NAME)..."
	@$(DOCKER_COMPOSE) build

run:
	@echo "Running $(SERVICE_NAME)..."
	@$(DOCKER_COMPOSE) up --remove-orphans

test:
	@echo "Running tests..."
	@make build
	@$(DOCKER_COMPOSE) up --remove-orphans -d
	@go test -v -count=1 ./...
	@$(DOCKER_COMPOSE) down
