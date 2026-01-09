# Variables
BINARY_NAME=online-clipboard
DOCKER_COMPOSE=docker-compose -f hack/docker-compose.yml
MODULE_PATH=github.com/msniranjan18/online-clipboard

.PHONY: help build run test clean docker-up docker-down

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

tidy: ## Add missing and remove unused modules
	go mod tidy

build: tidy ## Build the binary
	go build -o bin/$(BINARY_NAME) ./cmd/main.go

run: build ## Build and run the Go application locally
	./bin/$(BINARY_NAME)

docker-up: ## Start Postgres and Redis containers
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop containers
	$(DOCKER_COMPOSE) down

docker-logs: ## Follow docker logs
	$(DOCKER_COMPOSE) logs -f

db-shell: ## Enter Postgres shell for debugging
	docker exec -it $$(docker ps -qf "name=postgres") psql -U user -d clipdb

clean: ## Remove binary and stop containers
	rm -rf bin/
	$(DOCKER_COMPOSE) down -v