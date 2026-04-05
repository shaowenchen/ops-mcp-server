# MCP Tools Usage Guide

This guide provides detailed usage instructions for all available MCP tools in the Ops MCP Server.

## Table of Contents

- [SOPS Module](#sops-module)
  - [execute-sop-from-ops](#execute-sop-from-ops)
  - [list-sops-from-ops](#list-sops-from-ops)
  - [get-sop-parameters-from-ops](#get-sop-parameters-from-ops)
- [Events Module](#events-module)
  - [list-events-from-ops](#list-events-from-ops)
  - [get-events-from-ops](#get-events-from-ops)
- [Metrics Module](#metrics-module)
  - [list-metrics-from-prometheus](#list-metrics-from-prometheus)
  - [query-metrics-from-prometheus](#query-metrics-from-prometheus)
  - [query-metrics-range-from-prometheus](#query-metrics-range-from-prometheus)
- [Logs Module](#logs-module)
  - [search-logs-from-elasticsearch](#search-logs-from-elasticsearch)
  - [list-log-indices-from-elasticsearch](#list-log-indices-from-elasticsearch)
  - [query-logs-from-elasticsearch](#query-logs-from-elasticsearch)
- [Traces Module](#traces-module)
  - [get-services-from-jaeger](#get-services-from-jaeger)
  - [get-operations-from-jaeger](#get-operations-from-jaeger)
  - [get-trace-from-jaeger](#get-trace-from-jaeger)
  - [find-traces-from-jaeger](#find-traces-from-jaeger)

---

## SOPS Module

Standard Operating Procedures (SOPS) tools for executing operational procedures.

> **⚠️ Important Workflow**: Always use `get-sop-parameters-from-ops` to check required parameters **before** executing a SOPS with `execute-sop-from-ops`. This ensures you provide all required parameters with correct values.

### Recommended Workflow

```
┌─────────────────────────────────────────────────────────────┐
│  SOPS Execution Workflow (Follow this order!)               │
├─────────────────────────────────────────────────────────────┤
│  1. list-sops-from-ops                                      │
│     └─> Find available SOPS IDs                            │
│                                                              │
│  2. get-sop-parameters-from-ops  ⚠️ REQUIRED             │
│     └─> Get parameter requirements                          │
│     └─> Check: required, enums, regex, examples            │
│     └─> Note: value, default fields                        │
│                                                              │
│  3. execute-sop-from-ops                                   │
│     └─> Execute with validated parameters                   │
└─────────────────────────────────────────────────────────────┘
```

**Step Details:**

1. **Discover** available SOPS: `list-sops-from-ops` (optional, if you don't know the SOPS ID)
2. **Check parameters** for a specific SOPS: `get-sop-parameters-from-ops` (**MANDATORY**)
   - Shows required vs optional parameters
   - Reveals validation rules (`enums`, `regex`)
   - Provides examples and defaults
3. **Execute** the SOPS with correct parameters: `execute-sop-from-ops`

### execute-sop-from-ops

Execute a standard operation procedure (SOPS).

> **⚠️ Before executing**: Use `get-sop-parameters-from-ops` to view required parameters, their types, validation rules, and examples.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `sops_id` | string | ✅ Yes | ID of the SOPS procedure to execute |
| *(pipeline variables)* | string / number / boolean | No | Pass each variable as a **top-level** key (same names as `get-sop-parameters-from-ops`). |
| `parameters` | string or object | No | Optional nested payload; merged with top-level keys, **top-level wins** on the same name. |

**Example (flat):**

```json
{
  "sops_id": "example-pipeline-a",
  "environment": "staging",
  "force": "false"
}
```

**Optional nested `parameters` (merged with top-level; top-level wins):**

```json
{
  "sops_id": "example-pipeline-a",
  "parameters": "{\"environment\":\"staging\",\"force\":\"false\"}"
}
```

**Complete Workflow Example:**

```bash
# Step 1: List all available SOPS (optional, to find the SOPS ID)
# Tool: list-sops-from-ops
{}

# Response shows:
# {
#   "available_sops": [
#     {"id": "example-pipeline-a", "description": "Example procedure A"}
#   ]
# }

# Step 2: Check required parameters (REQUIRED before execution)
# Tool: get-sop-parameters-from-ops
{
  "sops_id": "example-pipeline-a"
}

# Response shows:
# {
#   "parameters": [
#     {
#       "name": "environment",
#       "required": true,
#       "enums": ["dev", "staging", "production"],
#       "description": "Target environment"
#     },
#     {
#       "name": "force",
#       "required": false,
#       "default": "false",
#       "description": "Force restart without graceful shutdown"
#     }
#   ]
# }

# Step 3: Execute with correct parameters (flat keys)
# Tool: execute-sop-from-ops
{
  "sops_id": "example-pipeline-a",
  "environment": "staging",
  "force": "false"
}
```

**Use Cases:**

- Execute automated operational procedures
- Run maintenance tasks with predefined workflows
- Trigger deployment or rollback procedures

**Response Example:**

```markdown
# Pipeline Run: example-pipeline-a
Status: Success
Duration: 45s
...
```

---

### list-sops-from-ops

List all available SOPS procedures.

**Parameters:** None

**Example:**

```json
{}
```

**Use Cases:**

- Discover available operational procedures
- Get an overview of all SOPS in the system
- Find the correct SOPS ID before execution

**Response Example:**

```json
{
  "available_sops": [
    {
      "id": "example-pipeline-a",
      "description": "Example procedure A",
      "variables": {...}
    },
    {
      "id": "example-pipeline-b",
      "description": "Example procedure B",
      "variables": {...}
    }
  ],
  "count": 2
}
```

---

### get-sop-parameters-from-ops

Get the parameter schema for a specific SOPS procedure (required flags, types, enums, regex, examples).

> **⚠️ REQUIRED STEP**: This tool MUST be called before executing any SOPS. It reveals all parameter requirements, validation rules, and examples needed for successful execution.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `sops_id` | string | ✅ Yes | ID of the SOPS procedure to get parameters for |

**Example:**

```json
{
  "sops_id": "example-pipeline-a"
}
```

**Use Cases:**

- ✅ **REQUIRED**: Check parameters before executing any SOPS
- Understand parameter requirements and validation rules
- Get parameter descriptions, examples, and default values
- Identify required vs optional parameters
- View validation constraints (enums, regex patterns)

**Response Example:**

```json
{
  "sops_id": "example-pipeline-a",
  "parameters": [
    {
      "name": "environment",
      "description": "Target environment",
      "required": true,
      "display": "Environment",
      "enums": ["dev", "staging", "production"],
      "examples": ["staging"],
      "value": ""  // Pre-filled value (if any)
    },
    {
      "name": "force",
      "description": "Skip graceful shutdown when true",
      "required": false,
      "display": "Force",
      "default": "false",
      "value": "false"  // Pre-filled with default value
    },
    {
      "name": "timeout",
      "description": "Timeout in seconds",
      "required": false,
      "display": "Timeout",
      "default": "30",
      "regex": "^[0-9]+$",  // Validation pattern
      "examples": ["30", "60", "120"]
    }
  ],
  "count": 3
}
```

**Response Fields Explained:**

| Field | Description | Usage |
|-------|-------------|-------|
| `name` | Parameter name | Use as top-level argument key in `execute-sop-from-ops` |
| `description` | Parameter description | Understand what the parameter does |
| `required` | Is this parameter required? | Must provide if true |
| `display` | Human-friendly display name | For UI display |
| `value` | Pre-filled value | Use this value if present |
| `default` | Default value if not provided | Can be omitted if using default |
| `enums` | List of valid values | Must use one of these values |
| `regex` | Validation pattern | Value must match this pattern |
| `examples` | Example values | Reference for valid inputs |

---

## Events Module

Kubernetes and system event monitoring tools.

### list-events-from-ops

List available event types by querying the backend API. Supports search filtering and pagination to discover different event categories and patterns.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `page` | string | No | Page number for pagination (default: 1) |
| `page_size` | string | No | Number of event types to return (default: 10) |
| `search` | string | No | Search term to filter event types (optional) |

**Example:**

```json
{
  "page": "1",
  "page_size": "20",
  "search": "pod"
}
```

**Use Cases:**

- Discover available event types in the system
- Search for specific event patterns
- Browse event categories with pagination

**Response Example:**

```json
{
  "event_types": [
    "ops.clusters.*.namespaces.*.pods.*.event",
    "ops.clusters.*.namespaces.*.deployments.*.event"
  ],
  "page": 1,
  "page_size": 20,
  "total": 50
}
```

---

### get-events-from-ops

Get events using raw NATS subject patterns. Supports three query types:

1. Direct query (exact subject)
2. Wildcard query (using `*` for single level)
3. Prefix matching (using `>` for multi-level suffix)

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `subject_pattern` | string | ✅ Yes | NATS subject pattern for event querying (supports wildcards `*` and `>`) |
| `page` | string | No | Page number for pagination (default: 1) |
| `page_size` | string | No | Number of events per page (default: 10) |
| `start_time` | string | No | Start time for filtering events (Unix epoch in milliseconds) |

**Examples:**

```json
// Direct query - exact subject
{
  "subject_pattern": "ops.clusters.example-cluster.namespaces.default.pods.example-pod.event",
  "page_size": "50"
}

// Wildcard query - single level wildcard
{
  "subject_pattern": "ops.clusters.*.namespaces.ops-system.webhooks.*",
  "start_time": "1700000000000"
}

// Prefix matching - multi-level suffix
{
  "subject_pattern": "ops.clusters.*.namespaces.default.hosts.>",
  "page": "1",
  "page_size": "100"
}
```

**Use Cases:**

- Monitor pod lifecycle events
- Track deployment changes
- Investigate host-level issues
- Query events across multiple clusters

**Subject Pattern Examples:**

| Pattern | Description |
|---------|-------------|
| `ops.clusters.{cluster}.namespaces.{namespace}.pods.{pod-name}.event` | Events for a specific pod |
| `ops.clusters.*.namespaces.ops-system.webhooks.*` | All webhook events in ops-system namespace across clusters |
| `ops.clusters.*.namespaces.{namespace}.hosts.>` | All host events in a namespace across clusters |

---

## Metrics Module

Prometheus metrics querying tools.

### list-metrics-from-prometheus

List all available metrics from Prometheus. Returns metric names, types, and basic information.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `limit` | string | No | Maximum number of metrics to return (default: 100) |
| `search` | string | No | Filter metrics by name pattern (optional) |

**Example:**

```json
{
  "limit": "50",
  "search": "http"
}
```

**Use Cases:**

- Discover available metrics in Prometheus
- Search for specific metric names
- Explore metric types and metadata

**Response Example:**

```json
{
  "metrics": [
    {
      "name": "http_requests_total",
      "type": "counter",
      "help": "Total number of HTTP requests"
    },
    {
      "name": "http_request_duration_seconds",
      "type": "histogram",
      "help": "HTTP request duration in seconds"
    }
  ],
  "count": 50
}
```

---

### query-metrics-from-prometheus

Execute a custom PromQL instant query.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | ✅ Yes | PromQL query expression to execute |

**Examples:**

```json
// Simple query - check service health
{
  "query": "up"
}

// Metric query
{
  "query": "cpu_usage_percent"
}

// Aggregation query
{
  "query": "sum(rate(ops_mcp_server_http_requests_total[5m]))"
}

// Query with labels
{
  "query": "http_requests_total{status=\"200\",method=\"GET\"}"
}
```

**Use Cases:**

- Check current metric values
- Validate service health status
- Get instant metric snapshots
- Calculate real-time aggregations

**PromQL Query Examples:**

| Query | Description |
|-------|-------------|
| `up` | Check if targets are up |
| `node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes` | Memory usage percentage |
| `rate(http_requests_total[5m])` | Request rate over 5 minutes |
| `sum(container_memory_usage_bytes) by (pod)` | Memory usage grouped by pod |

---

### query-metrics-range-from-prometheus

Execute a custom PromQL range query over a time period.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | ✅ Yes | PromQL query expression to execute |
| `time_range` | string | ✅ Yes | Time range for query (examples: 5m, 10m, 1h, 2h, 24h, 7d) |
| `step` | string | No | Query resolution step (default: 15s, examples: 15s, 30s, 60s, 1m, 5m) |

**Examples:**

```json
// CPU usage over last hour
{
  "query": "rate(cpu_usage[5m])",
  "time_range": "1h",
  "step": "30s"
}

// Memory usage by pod over last 24 hours
{
  "query": "sum(memory_usage_bytes) by (pod)",
  "time_range": "24h",
  "step": "5m"
}

// Request rate over last 7 days
{
  "query": "rate(http_requests_total[5m])",
  "time_range": "7d",
  "step": "1h"
}
```

**Use Cases:**

- Analyze metric trends over time
- Create time-series visualizations
- Investigate performance issues
- Track resource usage patterns

**Time Range Format:**

| Format | Description |
|--------|-------------|
| `5m` | 5 minutes |
| `1h` | 1 hour |
| `24h` | 24 hours |
| `7d` | 7 days |

**Step Format:**

| Format | Description |
|--------|-------------|
| `15s` | 15 seconds |
| `1m` | 1 minute |
| `5m` | 5 minutes |
| `1h` | 1 hour |

---

## Logs Module

Elasticsearch log searching and querying tools.

### search-logs-from-elasticsearch

Full-text search across log messages using Elasticsearch Query DSL.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `index` | string | ✅ Yes | Index name or pattern to search (e.g., 'logs-*', 'filebeat-*') |
| `body` | string | ✅ Yes | Complete Elasticsearch query body as JSON string |

**Examples:**

```json
// Simple text search
{
  "index": "logs-*",
  "body": "{\"query\":{\"query_string\":{\"query\":\"error\"}},\"size\":10}"
}

// Search with aggregation
{
  "index": "filebeat-*",
  "body": "{\"size\":0,\"query\":{\"query_string\":{\"query\":\"error\"}},\"aggs\":{\"by_level\":{\"terms\":{\"field\":\"level.keyword\"}}}}"
}

// Advanced query with filters
{
  "index": "logs-*",
  "body": "{\"query\":{\"bool\":{\"must\":[{\"match\":{\"message\":\"exception\"}},{\"range\":{\"@timestamp\":{\"gte\":\"now-1h\"}}}]}},\"size\":50,\"sort\":[{\"@timestamp\":{\"order\":\"desc\"}}]}"
}
```

**Use Cases:**

- Search for error messages
- Find specific log patterns
- Aggregate logs by fields
- Filter logs by time range and conditions

**Query DSL Features:**

The `body` parameter supports all Elasticsearch Query DSL features:

- `query` - Search queries (match, term, range, etc.)
- `aggs` - Aggregations (terms, date_histogram, etc.)
- `size` - Number of results to return
- `from` - Pagination offset
- `sort` - Sort order
- `_source` - Fields to return

---

### list-log-indices-from-elasticsearch

List all available log indices in Elasticsearch.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `format` | string | No | Output format (table, json) - default: table |
| `health` | string | No | Filter by health status (green, yellow, red) |
| `status` | string | No | Filter by status (open, close) |

**Example:**

```json
{
  "format": "json",
  "health": "green",
  "status": "open"
}
```

**Use Cases:**

- Discover available log indices
- Check index health status
- Monitor index sizes and document counts
- Find the correct index pattern for searching

**Response Example:**

```json
{
  "indices": [
    {
      "index": "logs-2024.01.15",
      "health": "green",
      "status": "open",
      "docs.count": "1234567",
      "store.size": "2.5gb"
    },
    {
      "index": "filebeat-7.x-2024.01.15",
      "health": "green",
      "status": "open",
      "docs.count": "987654",
      "store.size": "1.8gb"
    }
  ]
}
```

---

### query-logs-from-elasticsearch

Query logs using ES|QL (Elasticsearch Query Language).

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | ✅ Yes | ES\|QL query string |
| `format` | string | No | Response format (json, csv, tsv, txt) - default: json |
| `columnar` | string | No | Return results in columnar format (true or false) - default: false |

**Examples:**

```json
// Count logs by level in the last hour
{
  "query": "FROM logs-* | WHERE @timestamp > NOW() - 1 hour | STATS count() BY level"
}

// Get error logs with specific fields
{
  "query": "FROM logs-* | WHERE level == 'error' | KEEP @timestamp, message, host | LIMIT 100",
  "format": "json"
}

// Aggregate by host and time
{
  "query": "FROM filebeat-* | STATS count = COUNT(*) BY host, bucket = BUCKET(@timestamp, 1 hour)"
}

// CSV export
{
  "query": "FROM logs-* | WHERE @timestamp > NOW() - 24 hours | KEEP @timestamp, level, message",
  "format": "csv"
}
```

**Use Cases:**

- Perform SQL-like queries on logs
- Create custom aggregations and statistics
- Export logs in various formats
- Analyze log patterns with familiar SQL syntax

**ES|QL Commands:**

| Command | Description |
|---------|-------------|
| `FROM` | Specify the index pattern |
| `WHERE` | Filter conditions |
| `STATS` | Aggregations (COUNT, SUM, AVG, etc.) |
| `KEEP` | Select specific fields |
| `LIMIT` | Limit number of results |
| `SORT` | Sort results |
| `BUCKET` | Time bucketing for aggregations |

---

## Traces Module

Jaeger distributed tracing tools.

### get-services-from-jaeger

Gets the service names as JSON array of strings.

**Parameters:** None

**Example:**

```json
{}
```

**Use Cases:**

- List all services reporting traces
- Discover service names for further trace queries
- Monitor which services are being traced

**Response Example:**

```json
{
  "data": ["frontend", "backend-api", "database-service", "cache-service"]
}
```

---

### get-operations-from-jaeger

Gets the operations as JSON array of objects with name and spanKind properties.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `service` | string | ✅ Yes | Filters operations by service name |
| `spanKind` | string | No | Filters operations by OpenTelemetry span kind |

**Span Kinds:**

- `server` - Server-side operation
- `client` - Client-side operation
- `producer` - Message producer
- `consumer` - Message consumer
- `internal` - Internal operation

**Examples:**

```json
// Get all operations for a service
{
  "service": "backend-api"
}

// Get only server operations
{
  "service": "backend-api",
  "spanKind": "server"
}

// Get client operations
{
  "service": "frontend",
  "spanKind": "client"
}
```

**Use Cases:**

- List all operations/endpoints in a service
- Filter operations by type (server, client, etc.)
- Identify available operations before searching traces

**Response Example:**

```json
{
  "data": [
    {
      "name": "GET /api/users",
      "spanKind": "server"
    },
    {
      "name": "POST /api/users",
      "spanKind": "server"
    },
    {
      "name": "HTTP GET",
      "spanKind": "client"
    }
  ]
}
```

---

### get-trace-from-jaeger

Gets the spans by the given trace ID. Returns both original Jaeger format and converted OpenTelemetry format.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `traceId` | string | ✅ Yes | OpenTelemetry compatible trace ID (32-character hexadecimal string) |
| `startTime` | string | No | Start time to filter spans (RFC 3339 format) |
| `endTime` | string | No | End time to filter spans (RFC 3339 format) |

**Example:**

```json
{
  "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
  "startTime": "2024-01-15T10:00:00Z",
  "endTime": "2024-01-15T11:00:00Z"
}
```

**Use Cases:**

- Investigate specific trace by ID
- Analyze request flow through services
- Debug performance issues
- Understand service dependencies

**Time Format:**

- RFC 3339, section 5.6 format
- Example: `2017-07-21T17:32:28Z`

**Response Example:**

```json
{
  "jaeger_trace": {
    "data": [{
      "traceID": "4bf92f3577b34da6a3ce929d0e0e4736",
      "spans": [...]
    }]
  },
  "otel_trace": {
    "resourceSpans": [...]
  }
}
```

---

### find-traces-from-jaeger

Searches for traces based on criteria. Returns both original Jaeger format and converted OpenTelemetry format.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `serviceName` | string | ✅ Yes | Filters spans by service name |
| `startTimeMin` | string | ✅ Yes | Start of time interval (inclusive, RFC 3339 format) |
| `startTimeMax` | string | ✅ Yes | End of time interval (exclusive, RFC 3339 format) |
| `operationName` | string | No | The operation name to filter spans |
| `durationMin` | string | No | Minimum duration of a span in milliseconds |
| `durationMax` | string | No | Maximum duration of a span in milliseconds |
| `searchDepth` | string | No | Defines the maximum search depth |

**Examples:**

```json
// Find all traces for a service in the last hour
{
  "serviceName": "backend-api",
  "startTimeMin": "2024-01-15T10:00:00Z",
  "startTimeMax": "2024-01-15T11:00:00Z"
}

// Find slow traces (duration > 1 second)
{
  "serviceName": "backend-api",
  "startTimeMin": "2024-01-15T10:00:00Z",
  "startTimeMax": "2024-01-15T11:00:00Z",
  "durationMin": "1000"
}

// Find traces for specific operation
{
  "serviceName": "backend-api",
  "operationName": "GET /api/users",
  "startTimeMin": "2024-01-15T10:00:00Z",
  "startTimeMax": "2024-01-15T11:00:00Z"
}

// Find traces within duration range
{
  "serviceName": "frontend",
  "startTimeMin": "2024-01-15T10:00:00Z",
  "startTimeMax": "2024-01-15T11:00:00Z",
  "durationMin": "500",
  "durationMax": "2000"
}
```

**Use Cases:**

- Find slow requests (high latency)
- Search traces by service and operation
- Analyze performance patterns over time
- Identify bottlenecks in service calls

**Duration Examples:**

| Value | Description |
|-------|-------------|
| `100` | 100 milliseconds |
| `1000` | 1 second |
| `5000` | 5 seconds |
| `60000` | 1 minute |

---

## Tips and Best Practices

### General Guidelines

1. **Start with list/discovery tools** before executing specific queries
   - **SOPS**: Use `list-sops-from-ops` → `get-sop-parameters-from-ops` → `execute-sop-from-ops` (parameter check is mandatory!)
   - **Metrics**: Use `list-metrics-from-prometheus` before querying specific metrics
   - **Traces**: Use `get-services-from-jaeger` before searching traces

2. **Use pagination** for large result sets
   - Most list tools support `page` and `page_size` parameters
   - Default page sizes are typically 10-100 items

3. **Filter early** to improve performance
   - Use time ranges, search patterns, and filters
   - Narrow down results before fetching detailed data

4. **Check required parameters** marked with ✅
   - These must be provided for the tool to work
   - Optional parameters can enhance or filter results

### Module-Specific Tips

#### SOPS

- **⚠️ CRITICAL**: ALWAYS use `get-sop-parameters-from-ops` to check parameters **before** executing any SOPS
  - This shows required vs optional parameters
  - Reveals validation rules (enums, regex patterns)
  - Provides examples and default values
- Follow the recommended workflow: List → Check Parameters → Execute
- Validate parameter values match the required format (check `enums`, `regex` fields)
- Use examples from parameter metadata as reference
- Pay attention to the `value` field which may contain pre-filled values
- Never execute a SOPS without knowing its parameters first - this can prevent errors and failed executions

#### Events

- Use wildcard patterns (`*`) for flexible matching
- Use `>` for hierarchical matching (e.g., all sub-levels)
- Filter by `start_time` to reduce data volume

#### Metrics

- Start with instant queries before range queries
- Use appropriate step sizes for range queries (smaller = more data)
- Leverage PromQL functions: `rate()`, `sum()`, `avg()`, etc.

#### Logs

- Use ES|QL for SQL-like queries (more intuitive)
- Use Query DSL for advanced features (aggregations, complex filters)
- Specify time ranges to limit search scope
- Choose appropriate output format (JSON for processing, CSV for export)

#### Traces

- Always specify time ranges to improve query performance
- Use duration filters to find slow requests
- Start with service-level queries, then drill down to operations
- Trace IDs are 32-character hexadecimal strings

---

## Error Handling

Common error scenarios and solutions:

| Error | Cause | Solution |
|-------|-------|----------|
| SOPS execution failed | Missing required parameters or invalid values | **Always** use `get-sop-parameters-from-ops` first to check requirements |
| SOPS not found | Invalid `sops_id` | Use `list-sops-from-ops` to list available SOPS IDs |
| Parameter validation error | Value doesn't match required format | Check `enums`, `regex`, and `examples` from `get-sop-parameters-from-ops` |
| Missing required parameter | Required parameter not provided | Check tool definition and provide all required parameters |
| Invalid time format | Incorrect timestamp or time range format | Use RFC 3339 format for timestamps, duration strings for ranges |
| Index not found | Elasticsearch index doesn't exist | Use `list-log-indices-from-elasticsearch` to find available indices |
| Service not found | Jaeger service name doesn't exist | Use `get-services-from-jaeger` to list available services |
| Invalid JSON | Malformed JSON in body/parameters | Validate JSON syntax before sending |
| Authentication failed | Missing or invalid credentials | Check `SERVER_TOKEN` configuration and headers |

---

## Getting Help

- Check the [README.md](../README.md) for server configuration
- See [Authentication Examples](authentication-examples.md) for auth setup
- See [Client Configuration](client-configuration.md) for client-specific setup
- Review logs with `LOG_LEVEL=debug` for troubleshooting
