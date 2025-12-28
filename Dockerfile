# T111: Multi-stage Docker build for fast-ad-bidder
# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o bidder \
    cmd/bidder/main.go

# Stage 2: Create minimal runtime image
FROM alpine:3.19

# Install ca-certificates for HTTPS and tzdata for timezone support
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 bidder && \
    adduser -D -u 1000 -G bidder bidder

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bidder /app/bidder

# Copy configuration files (optional, can be mounted)
COPY --from=builder /build/config /app/config

# Change ownership
RUN chown -R bidder:bidder /app

# Switch to non-root user
USER bidder

# Expose bidder port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment defaults (can be overridden)
ENV SERVER_PORT=8080 \
    SERVER_READ_TIMEOUT=5s \
    SERVER_WRITE_TIMEOUT=10s \
    LOG_LEVEL=info

# Run the bidder
ENTRYPOINT ["/app/bidder"]
