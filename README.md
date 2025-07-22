# Ops MCP Server

A Model Context Protocol (MCP) server for operational tools including events, metrics, and logs management.

## Features

### Modules

- **Events Module**: Monitor Kubernetes events (pods, deployments, nodes)
- **Metrics Module**: Query Prometheus metrics and monitoring data
- **Logs Module**: Search and analyze logs via Elasticsearch

### Tools

The server provides the following MCP tools with configurable naming:

#### Events Tools

- `get-pod-events` - Get Kubernetes pod events from all pods in specified namespace/cluster
- `get-deployment-events` - Get Kubernetes deployment events from all deployments in specified namespace/cluster
- `get-node-events` - Get Kubernetes node events from all nodes in specified cluster

#### Metrics Tools

- `list-metrics` - List all available metrics from Prometheus
- `query-metrics` - Execute instant PromQL queries
- `query-metrics-range` - Execute range PromQL queries over a time period

#### Logs Tools

- `search-logs` - Full-text search across log messages
- `list-log-indices` - List all indices in the Elasticsearch cluster
- `get-pod-logs` - Query logs for a specific Kubernetes pod

### Tool Naming Convention

Tools use a consistent naming convention with **hyphens** as separators:

- **Format**: `{prefix}{verb-noun-context}{suffix}`
- **Examples**: `get-pod-events`, `list-metrics`, `search-logs`
- **Configurable**: Both prefix and suffix can be customized per module

## Configuration

Configure the server using a YAML file (default: `configs/config.yaml`):

```yaml
log:
  level: info

server:
  host: 0.0.0.0
  port: 3000
  mode: stdio # or "sse"

# Events Module Configuration
events:
  enabled: true
  endpoint: "https://ops-server.your-company.com/api/v1/events"
  token: "${EVENTS_API_TOKEN}"
  tools:
    prefix: "default-"
    suffix: "-provided-by-nats"

# Metrics Module Configuration
metrics:
  enabled: true
  tools:
    prefix: "default-"
    suffix: "-provided-by-prometheus"
  prometheus:
    endpoint: "https://prometheus.your-company.com/api/v1"
    timeout: 30

# Logs Module Configuration
logs:
  enabled: true
  tools:
    prefix: "default-"
    suffix: "-provided-by-elasticsearch"
  elasticsearch:
    endpoint: "https://elasticsearch.your-company.com:9200"
    username: "${ELASTICSEARCH_USER}"
    password: "${ELASTICSEARCH_PASSWORD}"
    timeout: 30
```

### Environment Variables

Set up your production environment by configuring these variables:

```bash
# Events API Configuration
export EVENTS_API_TOKEN="your-events-api-token"

# Elasticsearch Configuration
export ELASTICSEARCH_USER="elastic"
export ELASTICSEARCH_PASSWORD="your-elasticsearch-password"

# Alternative: Use API Key for Elasticsearch
# export ELASTICSEARCH_API_KEY="your-api-key"

# Optional: Prometheus Authentication
# export PROMETHEUS_TOKEN="your-prometheus-token"
```

### Tool Name Configuration

With the example configuration above, the actual tool names will be:

#### Events Tools

- `default-get-pod-events-provided-by-nats`
- `default-get-deployment-events-provided-by-nats`
- `default-get-node-events-provided-by-nats`

#### Metrics Tools

- `default-list-metrics-provided-by-prometheus`
- `default-query-metrics-provided-by-prometheus`
- `default-query-metrics-range-provided-by-prometheus`

#### Logs Tools

- `default-search-logs-provided-by-elasticsearch`
- `default-list-log-indices-provided-by-elasticsearch`
- `default-get-pod-logs-provided-by-elasticsearch`

To use default tool names (without prefix/suffix), set both `prefix` and `suffix` to empty strings `""`.

## Usage

### Tool Execution

Tools can be called with parameters (using actual configured tool names):

```javascript
// Execute metrics query
const result = await mcpClient.callTool(
  "default-query-metrics-provided-by-prometheus",
  {
    query: "count by (cluster) (up)",
  }
);

// Get pod events
const events = await mcpClient.callTool(
  "default-get-pod-events-provided-by-nats",
  {
    cluster: "production",
    namespace: "ai-nlp-fcheck",
    limit: "20",
  }
);

// Search logs
const logs = await mcpClient.callTool(
  "default-search-logs-provided-by-elasticsearch",
  {
    search_term: "error",
    limit: "50",
  }
);
```

## Running the Server

### Docker Container (Recommended)

#### Quick Start with Docker

```bash
# Run with default configuration
docker run -d \
  --name ops-mcp-server \
  -p 3000:3000 \
  -e EVENTS_API_TOKEN="your-events-api-token" \
  -e ELASTICSEARCH_USER="elastic" \
  -e ELASTICSEARCH_PASSWORD="your-elasticsearch-password" \
  shaowenchen/ops-mcp-server:latest \
  --mode=sse --enable-events --enable-metrics --enable-logs
```

#### Docker with Custom Configuration

```bash
# Run with custom config file
docker run -d \
  --name ops-mcp-server \
  -p 3000:3000 \
  -v $(pwd)/configs/config.yaml:/app/config/config.yaml \
  -e EVENTS_API_TOKEN="your-events-api-token" \
  -e ELASTICSEARCH_USER="elastic" \
  -e ELASTICSEARCH_PASSWORD="your-elasticsearch-password" \
  shaowenchen/ops-mcp-server:latest \
  --config=/app/config/config.yaml --mode=sse
```

#### Docker Compose

```yaml
version: "3.8"
services:
  ops-mcp-server:
    image: shaowenchen/ops-mcp-server:latest
    ports:
      - "3000:3000"
    environment:
      - OPS_MCP_ENV=production
      - OPS_MCP_LOG_LEVEL=info
      - EVENTS_API_TOKEN=${EVENTS_API_TOKEN}
      - ELASTICSEARCH_USER=${ELASTICSEARCH_USER}
      - ELASTICSEARCH_PASSWORD=${ELASTICSEARCH_PASSWORD}
    command:
      ["--mode=sse", "--enable-events", "--enable-metrics", "--enable-logs"]
    healthcheck:
      test:
        [
          "CMD",
          "wget",
          "--no-verbose",
          "--tries=1",
          "--spider",
          "http://localhost:3000/health",
        ]
      interval: 30s
      timeout: 3s
      retries: 3
    restart: unless-stopped
```

### Server Modes

#### SSE Mode (Server-Sent Events)

The server runs in SSE mode for web-based clients and HTTP API access:

```bash
# Access the server at http://localhost:3000
# Health check endpoint: http://localhost:3000/health
# API endpoints available for web clients
```

### CLI Options

- `--mode`: Server mode (`stdio` or `sse`, default: `stdio`)
- `--config`: Config file path (default: `configs/config.yaml`)
- `--enable-events`: Enable events module
- `--enable-metrics`: Enable metrics module
- `--enable-logs`: Enable logs module
- `--port`: Server port (default: 3000)
- `--host`: Server host (default: 0.0.0.0)
