# Ops MCP Server

A modular MCP (Model Context Protocol) server providing operational monitoring capabilities including events, metrics, and logs.

## Features

- **Events Module**: Query Kubernetes events (pods, deployments, nodes)
- **Metrics Module**: Query monitoring metrics through Grafana/Prometheus
- **Logs Module**: Analyze and search application logs

## Metrics Module - Prometheus Integration

The metrics module supports querying monitoring data directly through Prometheus API, allowing you to execute Prometheus queries.

### Available Tools

#### Basic Query Tools
- `query_metrics` - Execute custom PromQL queries
- `query_metrics_range` - Execute range queries with time intervals
- `get_metrics_status` - Get metrics module status
- `get_system_overview` - Get system overview metrics

#### Kubernetes Resource Discovery
- `get_clusters` - List all available Kubernetes clusters
- `get_namespaces` - List namespaces (optionally filtered by cluster)
- `get_pods` - List pods (optionally filtered by cluster/namespace)

#### Kubernetes Resource Usage
- `get_pod_resource_usage` - Get CPU and memory usage for pods
- `get_node_resource_usage` - Get CPU and memory usage for nodes

### Configuration

Configure Prometheus integration in `configs/config.yaml`:

```yaml
metrics:
  enabled: true
  prometheus:
    endpoint: "http://localhost:9090/api/v1"
```

### Environment Variables

Configure endpoint via environment variable if needed:

```bash
export METRICS_PROMETHEUS_ENDPOINT="http://localhost:9090/api/v1"
```

### Getting Prometheus Configuration

1. **Endpoint**: Your Prometheus server endpoint URL with API version (e.g., `http://localhost:9090/api/v1`)

### Example Queries

#### List Clusters
```json
{
  "tool": "get_clusters"
}
```

#### Get Pods in Specific Namespace
```json
{
  "tool": "get_pods",
  "cluster": "my-cluster",
  "namespace": "default",
  "limit": "10"
}
```

#### Get Pod Resource Usage
```json
{
  "tool": "get_pod_resource_usage",
  "cluster": "my-cluster",
  "namespace": "kube-system",
  "limit": "20"
}
```

#### Get Node Resource Usage
```json
{
  "tool": "get_node_resource_usage",
  "cluster": "my-cluster",
  "limit": "10"
}
```

#### Custom PromQL Query
```json
{
  "tool": "query_metrics",
  "query": "up{job=\"kubernetes-nodes\"}"
}
```

### Query Examples

**CPU Usage:**
```
rate(cpu_usage_total[5m])
```

**Memory Usage:**
```
(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100
```

**HTTP Request Rate:**
```
rate(http_requests_total[5m])
```

**Pod Resource Usage:**
```
container_memory_usage_bytes{pod=~"my-app-.*"}
```

### Time Ranges

Supported time range values:
- `1h` - Last hour
- `24h` - Last 24 hours  
- `7d` - Last 7 days
- `30d` - Last 30 days

### Query Steps

For range queries, you can specify the resolution step:
- `30s` - 30 second intervals
- `1m` or `60s` - 1 minute intervals (default)
- `5m` - 5 minute intervals
- `1h` - 1 hour intervals

## Events Module

Query Kubernetes events with improved parameter names:

- `get_pod_events` - Use `pod` parameter for specific pod name
- `get_deployment_events` - Use `deployment` parameter for specific deployment name  
- `get_nodes_events` - Use `node` parameter for specific node name

## Quick Start

1. Configure your monitoring endpoints in `configs/config.yaml`
2. Set sensitive tokens via environment variables
3. Run the server in stdio mode:

```bash
./bin/ops-mcp-server --config configs/config.yaml
```

## Building

```bash
go build -o bin/ops-mcp-server cmd/server/main.go
```

