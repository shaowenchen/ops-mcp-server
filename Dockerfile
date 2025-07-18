# Multi-stage build for optimal image size
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for go modules)
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments for multi-arch support
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG BUILD_TIME=unknown

# Build the application
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -a -installsuffix cgo \
    -o ops-mcp-server \
    ./cmd/server

# Create final minimal image
FROM alpine:3.18

# Install ca-certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/ops-mcp-server .

# Copy configuration files
COPY --from=builder /app/configs ./configs

# Create necessary directories and set ownership
RUN mkdir -p /app/logs && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Set environment variables
ENV OPS_MCP_ENV=production \
    OPS_MCP_LOG_LEVEL=info \
    OPS_MCP_SERVER_HOST=0.0.0.0 \
    OPS_MCP_SERVER_PORT=3000

# Default command
ENTRYPOINT ["./ops-mcp-server"]
CMD ["--config", "./configs/config.yaml"]

# Labels for better organization
LABEL maintainer="mail@chenshaowen.com" \
      version="1.0.0" \
      description="Ops MCP Server - Modular operational data querying server" \
      org.opencontainers.image.source="https://github.com/shaowenchen/ops-mcp-server" \
      org.opencontainers.image.documentation="https://github.com/shaowenchen/ops-mcp-server/blob/main/README.md" 