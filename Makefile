.PHONY: help build test clean run docker-build docker-push deploy

# Variables
SHORTENING_SERVICE := shortening-service
REDIRECT_SERVICE := redirect-service
BIN_DIR := bin
DOCKER_REGISTRY := gcr.io
GCP_PROJECT ?= your-gcp-project-id
VERSION ?= latest

# Go build variables
GO := go
GOFLAGS := -v
LDFLAGS := -w -s

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

## help: Display this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'

## install-deps: Install Go dependencies
install-deps:
	@echo "${GREEN}Installing dependencies...${NC}"
	$(GO) mod download
	$(GO) mod verify

## tidy: Tidy Go modules
tidy:
	@echo "${GREEN}Tidying Go modules...${NC}"
	$(GO) mod tidy

## build: Build both services
build: build-shortening build-redirect

## build-shortening: Build shortening service
build-shortening:
	@echo "${GREEN}Building $(SHORTENING_SERVICE)...${NC}"
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(SHORTENING_SERVICE) ./cmd/$(SHORTENING_SERVICE)

## build-redirect: Build redirect service
build-redirect:
	@echo "${GREEN}Building $(REDIRECT_SERVICE)...${NC}"
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(REDIRECT_SERVICE) ./cmd/$(REDIRECT_SERVICE)

## run-shortening: Run shortening service locally
run-shortening:
	@echo "${GREEN}Running $(SHORTENING_SERVICE)...${NC}"
	$(GO) run ./cmd/$(SHORTENING_SERVICE)

## run-redirect: Run redirect service locally
run-redirect:
	@echo "${GREEN}Running $(REDIRECT_SERVICE)...${NC}"
	$(GO) run ./cmd/$(REDIRECT_SERVICE)

## test: Run unit tests
test:
	@echo "${GREEN}Running unit tests...${NC}"
	$(GO) test -v -race -timeout 30s ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "${GREEN}Running tests with coverage...${NC}"
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Coverage report generated: coverage.html${NC}"

## test-integration: Run integration tests
test-integration:
	@echo "${GREEN}Running integration tests...${NC}"
	$(GO) test -v -race -tags=integration ./...

## bench: Run benchmarks
bench:
	@echo "${GREEN}Running benchmarks...${NC}"
	$(GO) test -bench=. -benchmem ./...

## lint: Run linter
lint:
	@echo "${GREEN}Running linter...${NC}"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "${YELLOW}golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin${NC}"; \
	fi

## fmt: Format code
fmt:
	@echo "${GREEN}Formatting code...${NC}"
	$(GO) fmt ./...

## vet: Run go vet
vet:
	@echo "${GREEN}Running go vet...${NC}"
	$(GO) vet ./...

## clean: Clean build artifacts
clean:
	@echo "${GREEN}Cleaning build artifacts...${NC}"
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	$(GO) clean

## docker-build: Build Docker images for both services
docker-build: docker-build-shortening docker-build-redirect

## docker-build-shortening: Build Docker image for shortening service
docker-build-shortening:
	@echo "${GREEN}Building Docker image for $(SHORTENING_SERVICE)...${NC}"
	docker build -t $(DOCKER_REGISTRY)/$(GCP_PROJECT)/$(SHORTENING_SERVICE):$(VERSION) \
		-f deployments/docker/Dockerfile.shortening .

## docker-build-redirect: Build Docker image for redirect service
docker-build-redirect:
	@echo "${GREEN}Building Docker image for $(REDIRECT_SERVICE)...${NC}"
	docker build -t $(DOCKER_REGISTRY)/$(GCP_PROJECT)/$(REDIRECT_SERVICE):$(VERSION) \
		-f deployments/docker/Dockerfile.redirect .

## docker-push: Push Docker images to registry
docker-push: docker-push-shortening docker-push-redirect

## docker-push-shortening: Push shortening service Docker image
docker-push-shortening:
	@echo "${GREEN}Pushing Docker image for $(SHORTENING_SERVICE)...${NC}"
	docker push $(DOCKER_REGISTRY)/$(GCP_PROJECT)/$(SHORTENING_SERVICE):$(VERSION)

## docker-push-redirect: Push redirect service Docker image
docker-push-redirect:
	@echo "${GREEN}Pushing Docker image for $(REDIRECT_SERVICE)...${NC}"
	docker push $(DOCKER_REGISTRY)/$(GCP_PROJECT)/$(REDIRECT_SERVICE):$(VERSION)

## docker-compose-up: Start local development environment with docker-compose
docker-compose-up:
	@echo "${GREEN}Starting local development environment...${NC}"
	docker-compose up -d

## docker-compose-down: Stop local development environment
docker-compose-down:
	@echo "${GREEN}Stopping local development environment...${NC}"
	docker-compose down

## docker-compose-logs: View docker-compose logs
docker-compose-logs:
	docker-compose logs -f

## migrate-create: Create a new migration (usage: make migrate-create NAME=create_users_table)
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "${YELLOW}Usage: make migrate-create NAME=migration_name${NC}"; \
		exit 1; \
	fi
	@echo "${GREEN}Creating migration: $(NAME)...${NC}"
	@timestamp=$$(date +%Y%m%d%H%M%S); \
	touch migrations/$${timestamp}_$(NAME).up.sql; \
	touch migrations/$${timestamp}_$(NAME).down.sql; \
	echo "Created migrations/$${timestamp}_$(NAME).up.sql"; \
	echo "Created migrations/$${timestamp}_$(NAME).down.sql"

## migrate-up: Run database migrations up
migrate-up:
	@echo "${GREEN}Running database migrations up...${NC}"
	@if command -v migrate > /dev/null; then \
		migrate -path migrations -database "$(DATABASE_URL)" up; \
	else \
		echo "${YELLOW}migrate tool not installed. Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest${NC}"; \
	fi

## migrate-down: Rollback last database migration
migrate-down:
	@echo "${GREEN}Rolling back database migration...${NC}"
	@if command -v migrate > /dev/null; then \
		migrate -path migrations -database "$(DATABASE_URL)" down 1; \
	else \
		echo "${YELLOW}migrate tool not installed. Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest${NC}"; \
	fi

## deploy-staging: Deploy to staging environment
deploy-staging:
	@echo "${GREEN}Deploying to staging...${NC}"
	./scripts/deploy.sh staging

## deploy-prod: Deploy to production environment
deploy-prod:
	@echo "${GREEN}Deploying to production...${NC}"
	./scripts/deploy.sh production

## load-test-shortening: Run load test for shortening service
load-test-shortening:
	@echo "${GREEN}Running load test for $(SHORTENING_SERVICE)...${NC}"
	@if command -v k6 > /dev/null; then \
		k6 run scripts/load-test-shortening.js; \
	else \
		echo "${YELLOW}k6 not installed. Install from: https://k6.io/docs/getting-started/installation/${NC}"; \
	fi

## load-test-redirect: Run load test for redirect service
load-test-redirect:
	@echo "${GREEN}Running load test for $(REDIRECT_SERVICE)...${NC}"
	@if command -v k6 > /dev/null; then \
		k6 run scripts/load-test-redirect.js; \
	else \
		echo "${YELLOW}k6 not installed. Install from: https://k6.io/docs/getting-started/installation/${NC}"; \
	fi

## dev-setup: Setup development environment
dev-setup: install-deps docker-compose-up migrate-up
	@echo "${GREEN}Development environment setup complete!${NC}"

## ci: Run CI checks (lint, test, build)
ci: lint vet test build
	@echo "${GREEN}CI checks passed!${NC}"

.DEFAULT_GOAL := help
