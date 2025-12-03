# Grafana Dashboard for Ops MCP Server

This directory contains the Grafana dashboard configuration for monitoring the Ops MCP Server.

## Dashboard Overview

The dashboard provides comprehensive monitoring of:

- **HTTP Metrics**: Request rate, duration, status codes, and request/response sizes
- **SSE Connections**: Active connections, connection rate, and duration
- **MCP Tool Calls**: Call rate, duration, and error rates by tool and module
- **Module Status**: Enabled/disabled status and request rates per module
- **Backend Services**: Request rates, durations, and errors for Prometheus, Elasticsearch, Jaeger, and Ops
- **Authentication**: Auth request rates and validation duration
- **System Resources**: Goroutine count and memory usage
- **Build Info**: Version, git commit, and build date

## Installation

### Method 1: Import via Grafana UI

1. Open Grafana and navigate to **Dashboards** → **Import**
2. Click **Upload JSON file** and select `dashboard.json`
3. Select your Prometheus data source
4. Click **Import**

### Method 2: Provision via Configuration

Add the dashboard to your Grafana provisioning configuration:

```yaml
# grafana/provisioning/dashboards/dashboard.yml
apiVersion: 1

providers:
  - name: 'Ops MCP Server'
    orgId: 1
    folder: 'MCP'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /path/to/grafana/dashboard.json
```

## Metrics Endpoint

The server exposes metrics at the `/metrics` endpoint (relative to your configured MCP URI).

For example, if your MCP URI is `/mcp`, the metrics endpoint will be:
```
http://localhost:80/mcp/metrics
```

## Prometheus Configuration

Add the following scrape configuration to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'ops-mcp-server'
    scrape_interval: 15s
    metrics_path: '/mcp/metrics'
    static_configs:
      - targets: ['localhost:80']
```

## Dashboard Panels

### HTTP Metrics
- **HTTP Request Rate**: Requests per second by method, endpoint, and status code
- **HTTP Request Duration (p95)**: 95th percentile latency
- **HTTP Requests In Flight**: Current number of concurrent requests
- **HTTP Status Code Distribution**: Pie chart of status codes

### SSE Metrics
- **SSE Active Connections**: Current number of active SSE connections
- **SSE Connection Rate**: New connections per second
- **SSE Connection Duration**: Connection lifetime percentiles

### MCP Tool Metrics
- **MCP Tool Calls Rate**: Tool invocation rate by tool and module
- **MCP Tool Call Duration**: Tool execution time percentiles
- **MCP Tool Errors**: Error rate by tool, module, and error type

### Module Metrics
- **Module Requests**: Request rate per module
- **Module Status**: Table showing enabled/disabled status

### Backend Metrics
- **Backend Request Rate**: Requests per second to backend services
- **Backend Request Duration**: Backend call latency percentiles
- **Backend Errors**: Error rate by backend and error type

### Authentication Metrics
- **Auth Requests**: Authentication request rate by status (success/failure/skipped)
- **Auth Validation Duration**: Token validation time percentiles

### System Metrics
- **Goroutines**: Number of active goroutines
- **Memory Usage**: Memory consumption by type (heap, stack, sys)
- **Build Info**: Version, git commit, and build date

## Customization

You can customize the dashboard by:

1. Modifying the time range defaults
2. Adjusting refresh intervals
3. Adding alert rules
4. Creating additional panels for specific metrics

## Troubleshooting

If metrics are not appearing:

1. Verify the metrics endpoint is accessible: `curl http://localhost:80/mcp/metrics`
2. Check Prometheus is scraping the endpoint
3. Verify the Prometheus data source is configured in Grafana
4. Check that the metric names match between the server and dashboard queries

## Related Documentation

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Ops MCP Server README](../README.md)

