# Ops MCP Server

A Model Context Protocol (MCP) server that provides AI assistants with access to operational data from Kubernetes, Prometheus, Elasticsearch, and Jaeger.

## Overview

Ops MCP Server enables AI assistants to query and interact with your observability stack through a unified MCP interface:

- **Kubernetes Events**: Monitor pods, deployments, and cluster events
- **Prometheus Metrics**: Query metrics with natural language
- **Elasticsearch Logs**: Search and analyze logs
- **SOPS Operations**: Execute standardized operational procedures
- **Jaeger Traces**: Investigate performance issues

## Features

- **Modular Design**: Enable only the modules you need
- **Multiple Protocols**: HTTP/SSE and stdio modes
- **Production Ready**: Built with Go, optimized for performance
- **Configurable**: YAML configuration with environment variable support

## Available Tools

### SOPS Module
- `execute-sops-from-ops` - Execute operational procedures
- `list-sops-from-ops` - List available procedures
- `list-sops-parameters-from-ops` - Get procedure parameters

### Events Module
- `get-events-from-ops` - Get Kubernetes events
- `list-events-from-ops` - List event types

### Metrics Module
- `list-metrics-from-prometheus` - List available metrics
- `query-metrics-from-prometheus` - Execute instant queries
- `query-metrics-range-from-prometheus` - Execute range queries

### Logs Module
- `search-logs-from-elasticsearch` - Search logs
- `list-log-indices-from-elasticsearch` - List log indices
- `get-pod-logs-from-elasticsearch` - Get pod logs

### Traces Module
- `get-services-from-jaeger` - List services
- `get-operations-from-jaeger` - List operations
- `get-trace-from-jaeger` - Get trace details
- `find-traces-from-jaeger` - Search traces

## Configuration

Configure the server using `configs/config.yaml`:

```yaml
log:
  level: info

server:
  host: 0.0.0.0
  port: 80
  mode: sse
  uri: /mcp
  # token: "${SERVER_TOKEN}"  # Optional: Uncomment to enable authentication

# Enable modules
sops:
  enabled: false
  tools:
    prefix: ""
    suffix: "-from-ops"
  ops:
    endpoint: "https://ops-server.your-company.com"
    token: "${SOPS_OPS_TOKEN}"

events:
  enabled: false
  tools:
    prefix: ""
    suffix: "-from-ops"
  ops:
    endpoint: "https://ops-server.your-company.com"
    token: "${EVENTS_OPS_TOKEN}"

metrics:
  enabled: false
  tools:
    prefix: ""
    suffix: "-from-prometheus"
  prometheus:
    endpoint: "https://prometheus.your-company.com"
    timeout: 30

logs:
  enabled: false
  tools:
    prefix: ""
    suffix: "-from-elasticsearch"
  elasticsearch:
    endpoint: "https://elasticsearch.your-company.com"
    username: "${LOGS_ELASTICSEARCH_USERNAME}"
    password: "${LOGS_ELASTICSEARCH_PASSWORD}"
    timeout: 30

traces:
  enabled: false
  tools:
    prefix: ""
    suffix: "-from-jaeger"
  jaeger:
    endpoint: "https://jaeger.your-company.com"
    timeout: 30
```

### Environment Variables

```bash
# Enable modules
export SOPS_ENABLED="true"
export EVENTS_ENABLED="true"
export METRICS_ENABLED="true"
export LOGS_ENABLED="true"
export TRACES_ENABLED="true"

# API endpoints
export SOPS_OPS_ENDPOINT="https://ops-server.your-company.com"
export SOPS_OPS_TOKEN="your-token"
export EVENTS_OPS_ENDPOINT="https://ops-server.your-company.com"
export EVENTS_OPS_TOKEN="your-token"
export LOGS_ELASTICSEARCH_USERNAME="elastic"
export LOGS_ELASTICSEARCH_PASSWORD="your-password"
export TRACES_JAEGER_ENDPOINT="https://jaeger.your-company.com"
# export SERVER_TOKEN="your-server-token"  # Optional: Uncomment to enable authentication
```

## Authentication

The MCP server supports optional token-based authentication. By default, no authentication is required. When a `token` is configured in the server configuration, protected endpoints will require a valid `Authorization` header with a Bearer token.

### Configuration

Set the `SERVER_TOKEN` environment variable or configure it in the YAML file:

```yaml
server:
  token: "${SERVER_TOKEN}"
```

### Usage

#### Default Behavior (No Authentication)
By default, all endpoints are accessible without authentication:

```bash
# All endpoints accessible without authentication
curl http://localhost:80/mcp/healthz
curl http://localhost:80/mcp/docs
curl http://localhost:80/mcp/sse
curl http://localhost:80/mcp/message
```

#### With Authentication Enabled
When `SERVER_TOKEN` is set, protected endpoints require authentication:

```bash
# Protected endpoints (require authentication)
curl -H "Authorization: Bearer your-server-token" http://localhost:80/mcp/sse
curl -H "Authorization: Bearer your-server-token" http://localhost:80/mcp/message

# Public endpoints (always accessible)
curl http://localhost:80/mcp/healthz
curl http://localhost:80/mcp/docs
```

### Security Notes

- **Default behavior**: If no `SERVER_TOKEN` is set, authentication is disabled and all requests are allowed
- **When token is configured**: The token is validated for MCP endpoints that require authentication:
  - SSE endpoint (`/mcp/sse`)
  - Message endpoint (`/mcp/message`) 
  - Main MCP handler (`/mcp`)
- **Always public endpoints** (never require authentication):
  - Health check endpoint (`/mcp/healthz`) - for monitoring
  - Documentation endpoint (`/mcp/docs`) - for API documentation
- Use strong, randomly generated tokens in production

## Usage

### Running the Server

#### Docker
```bash
docker run -d \
  --name ops-mcp-server \
  -p 80:80 \
  -e SOPS_ENABLED="true" \
  -e EVENTS_ENABLED="true" \
  -e METRICS_ENABLED="true" \
  -e LOGS_ENABLED="true" \
  -e TRACES_ENABLED="true" \
  shaowenchen/ops-mcp-server:latest \
  --mode=sse --enable-sops --enable-events --enable-metrics --enable-logs --enable-traces
```

#### Local Development
```bash
make build
./bin/ops-mcp-server --enable-sops --enable-events --enable-metrics --enable-logs --enable-traces
```

### Endpoints

- **MCP**: `http://localhost:80/mcp`
- **Health**: `http://localhost:80/mcp/healthz`
- **Docs**: `http://localhost:80/mcp/docs`
- **SSE**: `http://localhost:80/mcp/sse`
- **Message**: `http://localhost:80/mcp/message`

## Development

### Build
```bash
make build
```

### Test
```bash
make test
```

### Run
```bash
make run-all
```

## License

MIT License - see LICENSE file for details.
