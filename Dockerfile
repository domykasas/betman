# Multi-stage Dockerfile for the Coin Flip Game
# This builds both CLI and GUI versions of the application

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies for GUI
RUN apk add --no-cache \
    gcc \
    musl-dev \
    mesa-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    libgl1-mesa-dev \
    libxft-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build CLI application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o bin/coinflip-cli main_cli.go

# Build GUI application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o bin/coinflip-gui main_gui.go

# Run tests
RUN go test -v ./...

# Production stage for CLI
FROM alpine:latest AS cli

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S coinflip && \
    adduser -u 1001 -S coinflip -G coinflip

WORKDIR /home/coinflip

# Copy CLI binary and config
COPY --from=builder /app/bin/coinflip-cli /usr/local/bin/coinflip-cli
COPY --from=builder /app/configs/ ./configs/

# Change ownership
RUN chown -R coinflip:coinflip /home/coinflip

# Switch to non-root user
USER coinflip

# Default command
ENTRYPOINT ["coinflip-cli"]
CMD ["--help"]

# Production stage for GUI (requires X11 forwarding)
FROM alpine:latest AS gui

# Install GUI runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    mesa-dri-gallium \
    libxcursor \
    libxrandr \
    libxinerama \
    libxi \
    libgl \
    libxft

# Create non-root user
RUN addgroup -g 1001 -S coinflip && \
    adduser -u 1001 -S coinflip -G coinflip

WORKDIR /home/coinflip

# Copy GUI binary and config
COPY --from=builder /app/bin/coinflip-gui /usr/local/bin/coinflip-gui
COPY --from=builder /app/configs/ ./configs/

# Change ownership
RUN chown -R coinflip:coinflip /home/coinflip

# Switch to non-root user
USER coinflip

# Set display environment variable
ENV DISPLAY=:0

# Default command
ENTRYPOINT ["coinflip-gui"]

# Development stage with all tools
FROM golang:1.21-alpine AS dev

# Install development tools
RUN apk add --no-cache \
    gcc \
    musl-dev \
    mesa-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    libgl1-mesa-dev \
    libxft-dev \
    git \
    make

# Install Go tools
RUN go install golang.org/x/tools/cmd/goimports@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    go install golang.org/x/lint/golint@latest

WORKDIR /app

# Copy source
COPY . .

# Download dependencies
RUN go mod download

# Default command for development
CMD ["make", "help"]