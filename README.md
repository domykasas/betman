# 🪙 Coin Flip Game

An educational gambling game built with modern Go best practices, featuring both CLI and GUI interfaces. This project demonstrates clean architecture, dependency injection, comprehensive testing, and modern development practices.

![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)
![License](https://img.shields.io/badge/License-Educational-green.svg)
![Tests](https://img.shields.io/badge/Tests-90%25+%20Coverage-brightgreen.svg)
![Architecture](https://img.shields.io/badge/Architecture-Clean-orange.svg)

## ✨ Features

### 🎮 Game Features
- **Simple Coin Flip Betting**: Heads or tails with configurable payout ratios
- **Balance Management**: Starting balance and betting limits configuration
- **Game History**: Comprehensive tracking of all games with statistics
- **Fair Random Generation**: Cryptographically secure random number generation

### 🖥️ Dual Interface
- **CLI Interface**: Command-line interface with Cobra for scripting and automation
- **GUI Interface**: Cross-platform graphical interface built with Fyne
- **Interactive Mode**: Real-time gameplay with immediate feedback
- **Batch Mode**: Single-command betting for scripting

### 🏗️ Architecture
- **Clean Architecture**: Domain-driven design with clear separation of concerns
- **Dependency Injection**: Testable and modular component design
- **Interface-Based Design**: Easy mocking and testing
- **Context Support**: Proper cancellation and timeout handling

### 🧪 Quality & Testing
- **90%+ Test Coverage**: Comprehensive unit and integration tests
- **Benchmark Tests**: Performance validation
- **Mock-Based Testing**: Isolated component testing
- **Race Condition Testing**: Concurrent access validation

## 🚀 Quick Start

### Prerequisites
- **Go 1.21+**: Download from [golang.org](https://golang.org/dl/)
- **CGO Enabled**: Required for GUI components
- **Git**: For cloning the repository

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd coinflip-game

# Install dependencies
make deps

# Run tests to verify installation
make test

# Build both CLI and GUI
make build
```

### Running the Game

#### CLI Interface
```bash
# Interactive gameplay
make play
# or
./bin/coinflip-cli play

# Place a single bet
./bin/coinflip-cli bet --amount 10 --choice heads

# Check status and statistics
./bin/coinflip-cli status

# View game history
./bin/coinflip-cli history

# View configuration
./bin/coinflip-cli config
```

#### GUI Interface
```bash
# Launch GUI application
make run-gui
# or
./bin/coinflip-gui
```

## 🔧 Development

### Building

```bash
# Build for current platform
make build

# Build CLI only
make build-cli

# Build GUI only
make build-gui

# Cross-platform builds
make build-all

# Platform-specific builds
make build-cli-linux
make build-gui-windows
make build-cli-macos-arm64
```

### Testing

```bash
# Run all tests with coverage
make test

# Run tests with verbose output
make test-verbose

# Run benchmarks
make bench

# Generate coverage report (creates coverage.html)
make test
```

### Code Quality

```bash
# Run all quality checks
make check

# Individual checks
make fmt        # Format code
make vet        # Run go vet
make lint       # Run static analysis

# Security checks
make security

# View project statistics
make stats
```

### Development Tools

```bash
# Install development tools
make install-tools

# Run with hot reload
make dev

# Generate documentation
make docs
```

## 🏗️ Architecture

### Project Structure
```
coinflip-game/
├── cmd/                    # Application entry points
│   ├── cli/               # CLI implementation
│   │   ├── main.go
│   │   └── commands/      # Cobra commands
│   └── gui/               # GUI implementation
│       ├── main.go
│       └── ui/            # Fyne UI components
├── internal/              # Private application code
│   ├── game/              # Core game logic
│   ├── storage/           # Data persistence
│   ├── config/            # Configuration management
│   └── logger/            # Logging utilities
├── configs/               # Configuration files
├── tests/                 # Test files
├── main_cli.go           # CLI entry point
├── main_gui.go           # GUI entry point
├── Dockerfile            # Container definition
└── Makefile              # Build automation
```

### Clean Architecture Layers

1. **Domain Layer** (`internal/game/`): Core business logic
2. **Infrastructure Layer** (`internal/storage/`, `internal/logger/`): External concerns
3. **Interface Layer** (`cmd/cli/`, `cmd/gui/`): User interfaces
4. **Configuration Layer** (`internal/config/`): Application configuration

### Design Patterns

- **Dependency Injection**: Constructor injection with interfaces
- **Repository Pattern**: Abstract data access
- **Command Pattern**: CLI command structure
- **Factory Pattern**: Component creation
- **Observer Pattern**: UI updates

## ⚙️ Configuration

### Configuration File (`configs/config.json`)
```json
{
  "game": {
    "starting_balance": 1000.0,
    "min_bet": 1.0,
    "max_bet": 100.0,
    "payout_ratio": 2.0
  },
  "logging": {
    "level": "info",
    "development": false
  },
  "ui": {
    "theme": "dark",
    "window_width": 800,
    "window_height": 600
  }
}
```

### Environment Variables
```bash
# Game settings
export COINFLIP_GAME_STARTING_BALANCE=1500
export COINFLIP_GAME_MIN_BET=5
export COINFLIP_GAME_MAX_BET=200

# Logging settings
export COINFLIP_LOGGING_LEVEL=debug
export COINFLIP_LOGGING_DEVELOPMENT=true

# UI settings
export COINFLIP_UI_THEME=light
```

### Configuration Priority
1. Command line flags
2. Environment variables
3. Configuration file
4. Default values

## 🐳 Docker Support

### Building Images
```bash
# Build all Docker images
make docker-build

# Individual builds
docker build --target cli -t coinflip-game:cli .
docker build --target gui -t coinflip-game:gui .
docker build --target dev -t coinflip-game:dev .
```

### Running in Docker
```bash
# CLI in Docker
make docker-run-cli

# GUI in Docker (requires X11 forwarding)
make docker-run-gui

# Development environment
make docker-dev
```

## 🧪 Testing

### Test Coverage
The project maintains 90%+ test coverage across all packages:

- **Unit Tests**: Individual component testing
- **Integration Tests**: Cross-component interaction testing
- **Mock Testing**: Interface-based mocking for isolation
- **Benchmark Tests**: Performance validation
- **Race Condition Tests**: Concurrent access validation

### Running Specific Tests
```bash
# Run tests for specific package
go test -v ./internal/game/

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...

# Generate coverage for specific package
go test -coverprofile=coverage.out ./internal/game/
go tool cover -html=coverage.out
```

## 📚 Learning Resources

This project demonstrates several Go and software engineering concepts:

### Go Best Practices
- **Idiomatic Go**: Following Go conventions and style guidelines
- **Error Handling**: Proper error propagation and wrapping
- **Context Usage**: Cancellation and timeout patterns
- **Interface Design**: Small, focused interfaces
- **Package Organization**: Clear module boundaries

### Design Patterns
- **Clean Architecture**: Domain-driven design principles
- **Dependency Injection**: Testable component design
- **Repository Pattern**: Data access abstraction
- **Command Pattern**: CLI command structure

### Testing Strategies
- **Table-Driven Tests**: Comprehensive test case coverage
- **Mock Objects**: Interface-based testing
- **Test Coverage**: Measuring and maintaining coverage
- **Benchmark Testing**: Performance validation

## 🛠️ Troubleshooting

### Common Issues

#### Build Issues
```bash
# CGO build errors
export CGO_ENABLED=1

# Missing build tools (Ubuntu/Debian)
sudo apt-get install build-essential

# Missing build tools (macOS)
xcode-select --install
```

#### GUI Issues
```bash
# Linux GUI dependencies
sudo apt-get install libgl1-mesa-dev libxi-dev libxcursor-dev libxrandr-dev libxinerama-dev

# macOS GUI issues
# Ensure Xcode Command Line Tools are installed
```

#### Test Issues
```bash
# Race condition failures
export GOMAXPROCS=1

# Coverage issues
go clean -testcache
```

### Platform-Specific Notes

- **Linux**: Standard development tools sufficient
- **Windows**: Requires CGO-compatible compiler (mingw-w64)
- **macOS**: Requires Xcode Command Line Tools for GUI builds

## 🤝 Contributing

1. **Code Style**: Follow Go formatting conventions (`make fmt`)
2. **Testing**: Maintain 90%+ test coverage (`make test`)
3. **Documentation**: Update README and code comments
4. **Quality**: Pass all quality checks (`make check`)

### Development Workflow
```bash
# 1. Install development tools
make install-tools

# 2. Make changes
# ... edit code ...

# 3. Run quality checks
make check

# 4. Run tests
make test

# 5. Build and test
make build
```

## 📊 Project Statistics

```bash
# View project statistics
make stats

# Example output:
# Lines of code: 2,547
# Test files: 8
# Dependencies: 23
```

## 📝 License

This project is intended for educational purposes, demonstrating modern Go development practices, clean architecture, and comprehensive testing strategies.

## 🚦 Version History

- **v1.0.0**: Initial release with CLI and GUI interfaces
  - Clean architecture implementation
  - Comprehensive testing suite
  - Docker support
  - Cross-platform builds

---

**Built with ❤️ using Go, showcasing modern development practices for educational gambling applications.**

### Key Technologies
- **[Go 1.21+](https://golang.org/)**: Modern systems programming language
- **[Cobra](https://github.com/spf13/cobra)**: CLI framework
- **[Fyne](https://fyne.io/)**: Cross-platform GUI framework
- **[Viper](https://github.com/spf13/viper)**: Configuration management
- **[Zap](https://go.uber.org/zap)**: Structured logging
- **[Testify](https://github.com/stretchr/testify)**: Testing framework