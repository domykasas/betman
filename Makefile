# Makefile for Coin Flip Game
# Educational gambling game with CLI and GUI interfaces

APP_NAME=coinflip-game
CLI_NAME=coinflip
GUI_NAME=coinflip-gui
SERVER_NAME=coinflip-server
VERSION=1.0.0

# Build directories
BIN_DIR=bin
RELEASE_DIR=release

# Go build flags
GO_BUILD_FLAGS=-ldflags="-s -w" -trimpath
CGO_ENABLED=1

# Default target
.DEFAULT_GOAL := help

## Display this help message
help:
	@echo "Coin Flip Game - Build and Development Commands"
	@echo "================================================"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

## Install and verify dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	go mod verify
	@echo "✅ Dependencies installed"

## Run all quality checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "✅ All quality checks passed"

## Format Go code
fmt:
	@echo "🔧 Formatting code..."
	go fmt ./...
	goimports -w .
	@echo "✅ Code formatted"

## Run go vet
vet:
	@echo "🔍 Running go vet..."
	go vet ./...
	@echo "✅ Vet passed"

## Run static analysis
lint:
	@echo "🔍 Running static analysis..."
	@command -v staticcheck >/dev/null 2>&1 || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...
	@echo "✅ Lint passed"

## Run tests with coverage
test:
	@echo "🧪 Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"
	@echo "✅ Tests passed"

## Run tests with verbose output and coverage
test-verbose:
	@echo "🧪 Running verbose tests..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo "✅ Verbose tests completed"

## Build CLI application (main.go)
build-cli: deps
	@echo "🔨 Building CLI application..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(CLI_NAME) .
	@echo "✅ CLI built: $(BIN_DIR)/$(CLI_NAME)"

## Build GUI application (main_gui.go)
build-gui: deps
	@echo "🔨 Building GUI application..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(GUI_NAME) -tags gui .
	@echo "✅ GUI built: $(BIN_DIR)/$(GUI_NAME)"

## Build server application (main_server.go)
build-server: deps
	@echo "🔨 Building server application..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(SERVER_NAME) -tags server .
	@echo "✅ Server built: $(BIN_DIR)/$(SERVER_NAME)"

## Build all applications (CLI, GUI, Server)
build: build-cli build-gui build-server
	@echo "✅ All applications built"

## Build CLI for Linux
build-cli-linux: deps
	@echo "🔨 Building CLI for Linux..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(CLI_NAME)-linux-amd64 ./cmd/cli-app
	@echo "✅ Linux CLI built"

## Build GUI for Linux
build-gui-linux: deps
	@echo "🔨 Building GUI for Linux..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(GUI_NAME)-linux-amd64 ./cmd/gui-app
	@echo "✅ Linux GUI built"

## Build CLI for Windows
build-cli-windows: deps
	@echo "🔨 Building CLI for Windows..."
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(CLI_NAME)-windows-amd64.exe ./cmd/cli-app
	@echo "✅ Windows CLI built"

## Build GUI for Windows (requires cross-compilation setup)
build-gui-windows: deps
	@echo "🔨 Building GUI for Windows..."
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(GUI_NAME)-windows-amd64.exe ./cmd/gui-app
	@echo "✅ Windows GUI built"

## Build CLI for macOS
build-cli-macos: deps
	@echo "🔨 Building CLI for macOS..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(CLI_NAME)-darwin-amd64 ./cmd/cli-app
	@echo "✅ macOS CLI built"

## Build GUI for macOS
build-gui-macos: deps
	@echo "🔨 Building GUI for macOS..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(GUI_NAME)-darwin-amd64 ./cmd/gui-app
	@echo "✅ macOS GUI built"

## Build CLI for macOS Apple Silicon
build-cli-macos-arm64: deps
	@echo "🔨 Building CLI for macOS Apple Silicon..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(CLI_NAME)-darwin-arm64 ./cmd/cli-app
	@echo "✅ macOS ARM64 CLI built"

## Build GUI for macOS Apple Silicon
build-gui-macos-arm64: deps
	@echo "🔨 Building GUI for macOS Apple Silicon..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(GUI_NAME)-darwin-arm64 ./cmd/gui-app
	@echo "✅ macOS ARM64 GUI built"

## Build for all platforms
build-all: build-cli-linux build-gui-linux build-cli-windows build-gui-windows build-cli-macos build-gui-macos build-cli-macos-arm64 build-gui-macos-arm64
	@echo "✅ All platform builds completed"

## Run CLI application in development mode
run-cli: build-cli
	@echo "🚀 Running CLI application..."
	./$(BIN_DIR)/$(CLI_NAME) --help

## Run GUI application in development mode
run-gui: build-gui
	@echo "🚀 Running GUI application..."
	./$(BIN_DIR)/$(GUI_NAME)

## Run server in development mode
run-server: build-server
	@echo "🚀 Starting multiplayer server..."
	./$(BIN_DIR)/$(SERVER_NAME)

## Run CLI with play command
play: build-cli
	@echo "🎮 Starting interactive game..."
	./$(BIN_DIR)/$(CLI_NAME) play

## Run development server with hot reload
dev:
	@echo "🔄 Starting development mode..."
	@command -v air >/dev/null 2>&1 || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

## Generate documentation
docs:
	@echo "📚 Generating documentation..."
	@command -v godoc >/dev/null 2>&1 || (echo "Installing godoc..." && go install golang.org/x/tools/cmd/godoc@latest)
	@echo "📖 Run 'godoc -http=:6060' to view documentation at http://localhost:6060"

## Build Docker images
docker-build:
	@echo "🐳 Building Docker images..."
	docker build --target cli -t $(APP_NAME):cli .
	docker build --target gui -t $(APP_NAME):gui .
	docker build --target dev -t $(APP_NAME):dev .
	@echo "✅ Docker images built"

## Run CLI in Docker
docker-run-cli:
	@echo "🐳 Running CLI in Docker..."
	docker run --rm -it $(APP_NAME):cli play

## Run GUI in Docker (requires X11 forwarding)
docker-run-gui:
	@echo "🐳 Running GUI in Docker (requires X11)..."
	@echo "Note: You may need to run 'xhost +local:docker' first"
	docker run --rm -it -e DISPLAY=$$DISPLAY -v /tmp/.X11-unix:/tmp/.X11-unix $(APP_NAME):gui

## Run development environment in Docker
docker-dev:
	@echo "🐳 Running development environment..."
	docker run --rm -it -v $(PWD):/app $(APP_NAME):dev bash

## Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf $(BIN_DIR)/
	rm -rf $(RELEASE_DIR)/
	rm -f coverage.out coverage.html
	go clean -cache -modcache -testcache
	@echo "✅ Clean completed"

## Create release packages
release: clean check build-all
	@echo "📦 Creating release packages..."
	@mkdir -p $(RELEASE_DIR)/$(VERSION)
	@cp -r $(BIN_DIR)/* $(RELEASE_DIR)/$(VERSION)/
	@cp README.md $(RELEASE_DIR)/$(VERSION)/
	@cp configs/config.json $(RELEASE_DIR)/$(VERSION)/
	@cd $(RELEASE_DIR) && tar -czf $(APP_NAME)-$(VERSION)-linux.tar.gz $(VERSION)/*linux*
	@cd $(RELEASE_DIR) && zip -r $(APP_NAME)-$(VERSION)-windows.zip $(VERSION)/*windows* $(VERSION)/README.md $(VERSION)/config.json
	@cd $(RELEASE_DIR) && tar -czf $(APP_NAME)-$(VERSION)-darwin.tar.gz $(VERSION)/*darwin*
	@echo "✅ Release packages created in $(RELEASE_DIR)/"

## Install development tools
install-tools:
	@echo "🔧 Installing development tools..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/cosmtrek/air@latest
	go install golang.org/x/tools/cmd/godoc@latest
	@echo "✅ Development tools installed"

## Run security checks
security:
	@echo "🔒 Running security checks..."
	@command -v gosec >/dev/null 2>&1 || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...
	@echo "✅ Security checks completed"

## Run benchmarks
bench:
	@echo "🏃 Running benchmarks..."
	go test -bench=. -benchmem ./...
	@echo "✅ Benchmarks completed"

## Show project statistics
stats:
	@echo "📊 Project Statistics"
	@echo "===================="
	@echo "Lines of code:"
	@find . -name '*.go' -not -path './vendor/*' | xargs wc -l | tail -1
	@echo ""
	@echo "Test files:"
	@find . -name '*_test.go' | wc -l
	@echo ""
	@echo "Dependencies:"
	@go mod graph | wc -l

.PHONY: help deps check fmt vet lint test test-verbose build-cli build-gui build build-cli-linux build-gui-linux build-cli-windows build-gui-windows build-cli-macos build-gui-macos build-cli-macos-arm64 build-gui-macos-arm64 build-all run-cli run-gui play dev docs docker-build docker-run-cli docker-run-gui docker-dev clean release install-tools security bench stats