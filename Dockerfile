# Multi-stage build for optimal image size
FROM shaowenchen/builder-golang:1.24 AS builder

# Set working directory to the default workspace
WORKDIR /builder

COPY . .

# Set Go environment variables to skip checksum verification
ENV GOSUMDB=off
ENV GONOSUMDB=*
ENV GOPROXY=direct
ENV GO111MODULE=on

# Download dependencies first
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o bin/ops-mcp-server cmd/server/main.go

FROM shaowenchen/runtime-ubuntu:22.04

# Set working directory to the default workspace
WORKDIR /runtime

# Install ca-certificates and timezone data (using apt for Ubuntu)
RUN groupadd -g 1000 appgroup \
    && useradd -u 1000 -g appgroup -s /bin/bash -m appuser

# Copy binary from builder stage
COPY --from=builder /builder/bin/ops-mcp-server .

# Copy configuration files
COPY --from=builder /builder/configs ./configs

# Create necessary directories and set ownership
RUN mkdir -p /runtime/logs && \
    chown -R appuser:appgroup /runtime

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:80/healthz || exit 1

# Set environment variables
ENV OPS_MCP_ENV=production \
    OPS_MCP_LOG_LEVEL=info \
    OPS_MCP_SERVER_HOST=0.0.0.0 \
    OPS_MCP_SERVER_PORT=80

# Default command
ENTRYPOINT ["./ops-mcp-server"]
CMD ["--config", "./configs/config.yaml"]

# Labels for better organization
LABEL maintainer="mail@chenshaowen.com" \
    version="1.0.0" \
    description="Ops MCP Server - Modular operational data querying server" \
    org.opencontainers.image.source="https://github.com/shaowenchen/ops-mcp-server" \
    org.opencontainers.image.documentation="https://github.com/shaowenchen/ops-mcp-server/blob/master/README.md" 