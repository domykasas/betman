# Multi-stage build for P2P Coin Flip Betting
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev \
    pkgconfig \
    mesa-dev \
    libx11-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxcursor-dev \
    libxi-dev \
    glfw-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o coinflip-p2p .

# Runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    mesa-gl \
    libx11 \
    libxrandr \
    libxinerama \
    libxcursor \
    libxi \
    glfw \
    ca-certificates

# Create app user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/coinflip-p2p .

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to app user
USER appuser

# Expose default port
EXPOSE 8080

# Run the application
CMD ["./coinflip-p2p"]