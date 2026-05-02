APP_NAME := pack-calculator
MAIN_PACKAGE := ./cmd/server
BUILD_DIR := bin
GO ?= go
GOFLAGS := GO111MODULE=on
DOCKER_IMAGE ?= $(APP_NAME)
PORT ?= 8080

.PHONY: help run test test-domain test-api fmt tidy build clean docker-build docker-run

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Run the HTTP server locally
	$(GOFLAGS) $(GO) run $(MAIN_PACKAGE)

test: ## Run all tests
	$(GOFLAGS) $(GO) test ./...

test-domain: ## Run domain package tests
	$(GOFLAGS) $(GO) test ./internal/domain/orderpacks

test-api: ## Run HTTP adapter tests
	$(GOFLAGS) $(GO) test ./internal/interfaces/httpapi

fmt: ## Format all Go files
	$(GOFLAGS) $(GO) fmt ./...

tidy: ## Tidy Go module files
	$(GOFLAGS) $(GO) mod tidy

build: ## Build the server binary
	mkdir -p $(BUILD_DIR)
	$(GOFLAGS) $(GO) build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PACKAGE)

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run Docker image on PORT
	docker run --rm -p $(PORT):8080 $(DOCKER_IMAGE)
