# ops-mcp-server Metrics Documentation

This document describes all Prometheus metrics exported by the ops-mcp-server.

## Metric Naming Convention

All metrics are prefixed with `ops_mcp_server_` to avoid naming conflicts.

## HTTP Metrics

### `ops_mcp_server_http_requests_total`
- **Type**: Counter
- **Description**: Total number of HTTP requests processed by the server
- **Labels**:
  - `method`: HTTP method (GET, POST, etc.)
  - `endpoint`: Request endpoint path
  - `status_code`: HTTP status code (200, 404, 500, etc.)
  - `mode`: Server mode (stdio, sse)
- **Use Cases**: Monitor request volume, track error rates by status code

### `ops_mcp_server_http_request_duration_seconds`
- **Type**: Histogram
- **Description**: HTTP request duration in seconds
- **Labels**:
  - `method`: HTTP method
  - `endpoint`: Request endpoint path
  - `status_code`: HTTP status code
- **Buckets**: Default Prometheus buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)
- **Use Cases**: Monitor request latency, identify slow endpoints

### `ops_mcp_server_http_request_size_bytes`
- **Type**: Histogram
- **Description**: HTTP request size in bytes
- **Labels**:
  - `method`: HTTP method
  - `endpoint`: Request endpoint path
- **Buckets**: Exponential buckets from 100B to 100MB (100, 1000, 10000, 100000, 1000000, 10000000, 100000000)
- **Use Cases**: Monitor request payload sizes

### `ops_mcp_server_http_response_size_bytes`
- **Type**: Histogram
- **Description**: HTTP response size in bytes
- **Labels**:
  - `method`: HTTP method
  - `endpoint`: Request endpoint path
- **Buckets**: Exponential buckets from 100B to 100MB
- **Use Cases**: Monitor response payload sizes

### `ops_mcp_server_http_requests_in_flight`
- **Type**: Gauge
- **Description**: Number of HTTP requests currently being processed
- **Labels**:
  - `endpoint`: Request endpoint path
- **Use Cases**: Monitor server load, detect request queuing

## SSE (Server-Sent Events) Metrics

### `ops_mcp_server_sse_connections_total`
- **Type**: Counter
- **Description**: Total number of SSE connections established
- **Labels**: None
- **Use Cases**: Track connection growth over time

### `ops_mcp_server_sse_active_connections`
- **Type**: Gauge
- **Description**: Number of currently active SSE connections
- **Labels**: None
- **Use Cases**: Monitor current connection count, capacity planning

### `ops_mcp_server_sse_connection_duration_seconds`
- **Type**: Histogram
- **Description**: SSE connection duration in seconds
- **Labels**: None
- **Buckets**: [1s, 5s, 10s, 30s, 1m, 5m, 10m, 30m, 1h]
- **Use Cases**: Monitor connection lifetime, detect connection issues

## MCP Tool Metrics

### `ops_mcp_server_mcp_tool_calls_total`
- **Type**: Counter
- **Description**: Total number of MCP tool calls
- **Labels**:
  - `tool_name`: Name of the MCP tool
  - `module`: Module name (sops, events, metrics, logs, traces)
  - `status`: Call status (success, failure)
- **Use Cases**: Track tool usage, monitor success/failure rates

### `ops_mcp_server_mcp_tool_call_duration_seconds`
- **Type**: Histogram
- **Description**: MCP tool call duration in seconds
- **Labels**:
  - `tool_name`: Name of the MCP tool
  - `module`: Module name
- **Buckets**: Default Prometheus buckets
- **Use Cases**: Monitor tool performance, identify slow tools

### `ops_mcp_server_mcp_tool_errors_total`
- **Type**: Counter
- **Description**: Total number of MCP tool errors
- **Labels**:
  - `tool_name`: Name of the MCP tool
  - `module`: Module name
  - `error_type`: Type of error (e.g., "validation_error", "execution_error")
- **Use Cases**: Track error rates, identify problematic tools

## Module Metrics

### `ops_mcp_server_module_enabled`
- **Type**: Gauge
- **Description**: Module enabled status (0=disabled, 1=enabled)
- **Labels**:
  - `module_name`: Module name (sops, events, metrics, logs, traces)
- **Use Cases**: Verify module configuration, track module availability

### `ops_mcp_server_module_requests_total`
- **Type**: Counter
- **Description**: Total number of requests per module
- **Labels**:
  - `module_name`: Module name
- **Use Cases**: Track module usage, load distribution

## Backend Service Metrics

### `ops_mcp_server_backend_requests_total`
- **Type**: Counter
- **Description**: Total number of backend service requests
- **Labels**:
  - `backend`: Backend service name
  - `status`: Request status (success, error)
- **Use Cases**: Monitor backend service usage, track error rates

### `ops_mcp_server_backend_request_duration_seconds`
- **Type**: Histogram
- **Description**: Backend service request duration in seconds
- **Labels**:
  - `backend`: Backend service name
- **Buckets**: Default Prometheus buckets
- **Use Cases**: Monitor backend service performance

### `ops_mcp_server_backend_errors_total`
- **Type**: Counter
- **Description**: Total number of backend service errors
- **Labels**:
  - `backend`: Backend service name
  - `error_type`: Type of error
- **Use Cases**: Track backend service reliability

## Authentication Metrics

### `ops_mcp_server_auth_requests_total`
- **Type**: Counter
- **Description**: Total number of authentication requests
- **Labels**:
  - `status`: Authentication status (success, failure)
- **Use Cases**: Monitor authentication attempts, track failures

### `ops_mcp_server_auth_token_validation_duration_seconds`
- **Type**: Histogram
- **Description**: Authentication token validation duration in seconds
- **Labels**: None
- **Buckets**: [0.0001s, 0.0005s, 0.001s, 0.005s, 0.01s, 0.05s, 0.1s]
- **Use Cases**: Monitor authentication performance

## System Metrics

### `ops_mcp_server_process_goroutines`
- **Type**: Gauge
- **Description**: Number of goroutines in the process
- **Labels**: None
- **Use Cases**: Monitor goroutine leaks, track concurrency

### `ops_mcp_server_process_memory_bytes`
- **Type**: Gauge
- **Description**: Process memory usage in bytes
- **Labels**:
  - `type`: Memory type (e.g., "heap", "stack", "sys")
- **Use Cases**: Monitor memory consumption, detect memory leaks

## Build Information

### `ops_mcp_server_build_info`
- **Type**: Gauge
- **Description**: Build information (always set to 1)
- **Labels**:
  - `version`: Application version
  - `git_commit`: Git commit hash
  - `build_date`: Build date
- **Use Cases**: Track deployed versions, identify instances

## Common Queries

### Request Rate
```promql
sum(rate(ops_mcp_server_http_requests_total[5m])) by (status_code)
```

### Error Rate
```promql
sum(rate(ops_mcp_server_mcp_tool_errors_total[5m])) by (tool_name, module)
```

### P95 Latency
```promql
histogram_quantile(0.95, rate(ops_mcp_server_http_request_duration_seconds_bucket[5m]))
```

### Active Connections
```promql
ops_mcp_server_sse_active_connections
```

### Tool Call Success Rate
```promql
sum(rate(ops_mcp_server_mcp_tool_calls_total{status="success"}[5m])) / sum(rate(ops_mcp_server_mcp_tool_calls_total[5m]))
```

## Metric Export

Metrics are exposed at the `/mcp/metrics` endpoint when the server is running in SSE mode.

## Labels

### Common Labels
- `pod`: Pod name (when running in Kubernetes)
- `tool_name`: MCP tool name
- `module`: Module name (sops, events, metrics, logs, traces)
- `status_code`: HTTP status code
- `method`: HTTP method
- `endpoint`: Request endpoint path

### Label Values
- **Module names**: `sops`, `events`, `metrics`, `logs`, `traces`
- **Status codes**: Standard HTTP status codes (200, 404, 500, etc.)
- **HTTP methods**: `GET`, `POST`, `PUT`, `DELETE`, etc.
- **Tool status**: `success`, `failure`
- **Auth status**: `success`, `failure`

