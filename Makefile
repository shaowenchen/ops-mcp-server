# Makefile for Ops MCP Server
.PHONY: all build clean test lint dev run docker-build docker-run help

# Variables
APP_NAME = ops-mcp-server
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "latest")
BUILD_DIR = bin
MAIN_PATH = cmd/server
DOCKER_IMAGE = shaowenchen/$(APP_NAME)
LDFLAGS = -X main.version=$(VERSION) -w -s

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

# Default target
.PHONY: all
all: clean test build

# Build the application
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./$(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"
# Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./$(MAIN_PATH)
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./$(MAIN_PATH)
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./$(MAIN_PATH)
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./$(MAIN_PATH)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./$(MAIN_PATH)
	@echo "Multi-platform build complete"

# Test the application
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Test with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run the application locally
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME) $(ARGS)

# Run with SOPS module
.PHONY: run-sops
run-sops: build
	@echo "Running $(APP_NAME) with SOPS module..."
	./$(BUILD_DIR)/$(APP_NAME) --enable-sops

# Run with events module
.PHONY: run-events
run-events: build
	@echo "Running $(APP_NAME) with events module..."
	./$(BUILD_DIR)/$(APP_NAME) --enable-events

# Run with metrics module
.PHONY: run-metrics
run-metrics: build
	@echo "Running $(APP_NAME) with metrics module..."
	./$(BUILD_DIR)/$(APP_NAME) --enable-metrics

# Run with logs module
.PHONY: run-logs
run-logs: build
	@echo "Running $(APP_NAME) with logs module..."
	./$(BUILD_DIR)/$(APP_NAME) --enable-logs

# Run with traces module
.PHONY: run-traces
run-traces: build
	@echo "Running $(APP_NAME) with traces module..."
	./$(BUILD_DIR)/$(APP_NAME) --enable-traces

# Run with all modules
.PHONY: run-all
run-all: build
	@echo "Running $(APP_NAME) with all modules..."
	./$(BUILD_DIR)/$(APP_NAME) --enable-sops --enable-events --enable-metrics --enable-logs --enable-traces

# Test MCP functionality
.PHONY: test-mcp
test-mcp: build
	@echo "Testing MCP functionality..."
	@echo "Sending test requests to MCP server..."
	@echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"capabilities": {}, "clientInfo": {"name": "test", "version": "1.0"}}}' | ./$(BUILD_DIR)/$(APP_NAME) --enable-events --enable-metrics 2>/dev/null | head -1
	@echo "✓ MCP server responds to initialize request"
	@echo ""
	@echo "Testing tools list..."
	@(echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"capabilities": {}, "clientInfo": {"name": "test", "version": "1.0"}}}'; echo '{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}') | ./$(BUILD_DIR)/$(APP_NAME) --enable-events --enable-metrics 2>/dev/null | tail -1 | grep -q "list_events" && echo "✓ Tools list working correctly" || echo "✗ Tools list failed"
	@echo ""
	@echo "MCP server test completed successfully!"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Vendor dependencies
.PHONY: vendor
vendor:
	@echo "Vendoring dependencies..."
	$(GOMOD) tidy
	$(GOMOD) vendor

# Update dependencies
.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	$(GOMOD) vendor

# Lint the code
.PHONY: lint
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Format the code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

.PHONY: docker-run
docker-run-sse:
	@echo "Running Docker container..."
	docker run -it --rm -p 3000:3000 $(DOCKER_IMAGE):latest --enable-sops --enable-events --enable-metrics --enable-logs --enable-traces --mode sse

docker-run-stdio:
	@echo "Running Docker container..."
	docker run -it --rm $(DOCKER_IMAGE):latest --enable-sops --enable-events --enable-metrics --enable-logs --enable-traces --mode stdio

.PHONY: docker-push
docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# Quick development cycle
.PHONY: quick
quick: fmt lint test build

# Development helpers
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	$(GOMOD) download
	$(GOMOD) vendor

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Clean, test, and build"
	@echo "  build        - Build the application"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  test-mcp     - Test MCP functionality"
	@echo "  run          - Build and run the application"
	@echo "  run-sops     - Run with SOPS module only"
	@echo "  run-events   - Run with events module only"
	@echo "  run-metrics  - Run with metrics module only" 
	@echo "  run-logs     - Run with logs module only"
	@echo "  run-traces   - Run with traces module only"
	@echo "  run-all      - Run with all modules"
	@echo "  clean        - Clean build artifacts"
	@echo "  vendor       - Vendor dependencies"
	@echo "  deps-update  - Update dependencies"
	@echo "  lint         - Lint the code"
	@echo "  fmt          - Format the code"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  docker-push  - Push Docker image"
	@echo "  quick        - Quick development cycle (fmt, lint, test, build)"
	@echo "  dev-setup    - Setup development environment"
	@echo ""
	@echo "Kubernetes deployment targets:"
	@echo "  k8s-deploy   - Deploy to Kubernetes"
	@echo "  k8s-status   - Show Kubernetes deployment status"
	@echo "  k8s-cleanup  - Clean up Kubernetes resources"
	@echo "  k8s-logs     - Show application logs"
	@echo "  help         - Show this help"

# Kubernetes deployment targets
.PHONY: k8s-deploy
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	chmod +x deploy/deploy.sh
	./deploy/deploy.sh deploy

.PHONY: k8s-status
k8s-status:
	@echo "Checking Kubernetes deployment status..."
	./deploy/deploy.sh status

.PHONY: k8s-cleanup
k8s-cleanup:
	@echo "Cleaning up Kubernetes resources..."
	./deploy/deploy.sh cleanup

.PHONY: k8s-logs
k8s-logs:
	@echo "Showing application logs..."
	kubectl logs -l app=ops-mcp-server -n ops-mcp-server --tail=100 -f

.PHONY: k8s-build-deploy
k8s-build-deploy: docker-build docker-push k8s-deploy
	@echo "Complete build and deploy cycle finished" 