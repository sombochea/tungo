# TunGo Makefile

.PHONY: all build test clean run-server run-client docker-build docker-run help

# Variables
BINARY_SERVER=bin/server
BINARY_CLIENT=bin/client
VERSION?=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-w -s -X main.Version=$(VERSION)"

# Default target
all: test build

## help: Display this help message
help:
	@echo "TunGo Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^##' Makefile | sed 's/##//'

## build: Build both server and client binaries
build: build-server build-client

## build-server: Build the server binary
build-server:
	@echo "Building server..."
	@mkdir -p bin
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY_SERVER) ./cmd/server

## build-client: Build the client binary
build-client:
	@echo "Building client..."
	@mkdir -p bin
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY_CLIENT) ./cmd/client

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

## test-coverage: Run tests and display coverage
test-coverage: test
	go tool cover -html=coverage.out

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out
	rm -rf data/

## run-server: Run the server
run-server:
	@echo "Starting server..."
	go run ./cmd/server --config configs/server.example.yaml

## run-client: Run the client
run-client:
	@echo "Starting client..."
	go run ./cmd/client --port 8000 --verbose

## docker-build: Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker build -f Dockerfile.server -t tungo-server:$(VERSION) -t tungo-server:latest .
	docker build -f Dockerfile.client -t tungo-client:$(VERSION) -t tungo-client:latest .

## docker-run: Run server with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

## docker-stop: Stop Docker Compose services
docker-stop:
	docker-compose down

## docker-logs: View Docker Compose logs
docker-logs:
	docker-compose logs -f

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## lint: Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## install: Install binaries to $GOPATH/bin
install:
	@echo "Installing binaries..."
	go install $(LDFLAGS) ./cmd/server
	go install $(LDFLAGS) ./cmd/client

## dev: Run in development mode with auto-reload
dev:
	@echo "Running in development mode..."
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

## docker-cluster: Start 3-server Docker cluster
docker-cluster:
	@echo "Starting Docker cluster..."
	docker-compose up -d
	@sleep 3
	@docker-compose ps
	@echo ""
	@echo "✓ Cluster started! Servers available on:"
	@echo "  Server 1: localhost:5555 (proxy: 8080)"
	@echo "  Server 2: localhost:5556 (proxy: 8081)"
	@echo "  Server 3: localhost:5557 (proxy: 8082)"

## docker-cluster-down: Stop Docker cluster
docker-cluster-down:
	@echo "Stopping Docker cluster..."
	docker-compose down

## docker-cluster-logs: View Docker cluster logs
docker-cluster-logs:
	docker-compose logs -f

## docker-cluster-ps: Show Docker cluster status
docker-cluster-ps:
	docker-compose ps

## docker-cluster-restart: Restart Docker cluster
docker-cluster-restart:
	@echo "Restarting Docker cluster..."
	docker-compose restart

## docker-cluster-clean: Clean Docker cluster (remove containers and volumes)
docker-cluster-clean:
	@echo "Cleaning Docker cluster..."
	docker-compose down -v
	rm -rf data/server-*

## docker-stop-1: Stop server 1 (test failover)
docker-stop-1:
	docker-compose stop tungo-server-1

## docker-stop-2: Stop server 2 (test failover)
docker-stop-2:
	docker-compose stop tungo-server-2

## docker-stop-3: Stop server 3 (test failover)
docker-stop-3:
	docker-compose stop tungo-server-3

## docker-client-config: Generate client config for Docker cluster
docker-client-config:
	@mkdir -p config
	@echo "server_cluster:" > config/client-docker.yaml
	@echo "  - host: localhost" >> config/client-docker.yaml
	@echo "    port: 5555" >> config/client-docker.yaml
	@echo "  - host: localhost" >> config/client-docker.yaml
	@echo "    port: 5556" >> config/client-docker.yaml
	@echo "  - host: localhost" >> config/client-docker.yaml
	@echo "    port: 5557" >> config/client-docker.yaml
	@echo "" >> config/client-docker.yaml
	@echo "local_host: localhost" >> config/client-docker.yaml
	@echo "local_port: 3000" >> config/client-docker.yaml
	@echo "" >> config/client-docker.yaml
	@echo "connect_timeout: 10s" >> config/client-docker.yaml
	@echo "retry_interval: 5s" >> config/client-docker.yaml
	@echo "max_retries: 3" >> config/client-docker.yaml
	@echo "" >> config/client-docker.yaml
	@echo "log_level: info" >> config/client-docker.yaml
	@echo "log_format: console" >> config/client-docker.yaml
	@echo "✓ Client config created at config/client-docker.yaml"
	@echo ""
	@echo "Start client with:"
	@echo "  go run cmd/client/main.go --config config/client-docker.yaml --local-port 3000"

.DEFAULT_GOAL := help
