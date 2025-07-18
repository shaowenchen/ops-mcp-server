# Ops MCP Server

A modular Model Context Protocol (MCP) server for operational data querying, built with Go and using the official MCP protocol implementation.

## Overview

Ops MCP Server is a real MCP (Model Context Protocol) server that provides tools for operational data access through a standardized protocol. It uses the `github.com/mark3labs/mcp-go/server` library to provide genuine MCP protocol support via stdio communication.

## Available Modules and Tools

### Events Module
When enabled, provides these MCP tools:
- `list_events`: List all available events with optional limit and offset
- `get_event`: Get detailed information about a specific event by ID

### Metrics Module
When enabled, provides these MCP tools:
- `get_metrics_status`: Get the current status of the metrics module

### Logs Module  
When enabled, provides these MCP tools:
- `get_logs_status`: Get the current status of the logs module

## Installation

### Using Docker (Recommended)

```bash
# Run with events module enabled
docker run -it shaowenchen/ops-mcp-server --enable-events

# Run with multiple modules
docker run -it shaowenchen/ops-mcp-server --enable-events --enable-metrics --enable-logs

# Run with configuration file
docker run -it -v $(pwd)/configs:/app/configs shaowenchen/ops-mcp-server -c /app/configs/config.yaml
```

### Using Go

```bash
# Install from source
go install github.com/shaowenchen/ops-mcp-server/cmd/server@latest

# Or build locally
git clone https://github.com/shaowenchen/ops-mcp-server.git
cd ops-mcp-server
make build
```

## Usage

### Server Modes

The server supports two modes of operation:

1. **stdio mode** (default): Standard MCP protocol communication via stdin/stdout
   - This is the standard MCP transport used by most clients
   - Required for Claude Desktop, mcphost, and other MCP clients
2. **sse mode**: Server-Sent Events based MCP server for web clients
   - Uses Streamable HTTP transport with SSE as defined in the MCP specification
   - Provides real-time communication over HTTP using Server-Sent Events
   - Useful for web applications and custom integrations

#### Stdio Mode (Default)

Perfect for MCP clients like Claude Desktop:

```bash
# Enable events module
ops-mcp-server --events

# Enable multiple modules with stdio mode (default)
ops-mcp-server --enable-events --enable-metrics --enable-logs

# Use with config file
ops-mcp-server -c configs/config.yaml
```

#### SSE Mode

For web-based MCP clients using Server-Sent Events:

```bash
# Run SSE server on 0.0.0.0:3000 (accessible from any interface)
ops-mcp-server --mode sse --host 0.0.0.0 --port 3000 --enable-events

# Access MCP endpoint at http://localhost:3000/mcp (or use server IP)
```

### Basic Usage

The server communicates via the selected mode using the MCP protocol:

```bash
# Enable events module (stdio mode by default)
ops-mcp-server --enable-events

# Enable multiple modules
ops-mcp-server --enable-events --enable-metrics --enable-logs

# Use with config file
ops-mcp-server -c configs/config.yaml

# Switch to SSE mode
ops-mcp-server --mode sse --enable-events --enable-metrics --enable-logs
```

### MCP Client Integration

To use this server with MCP clients (like Claude Desktop, mcphost, etc.), configure your client to launch the server. Here are three common deployment patterns:

#### Configuration Examples

##### 1. SSE Mode (HTTP Transport)

For web-based clients or when you need HTTP-based communication:

```json
{
  "mcpServers": {
    "local-sse": {
      "disabled": false,
      "timeout": 60,
      "url": "http://localhost:3000/mcp",
      "transportType": "sse"
    }
  }
}
```

Start the server in SSE mode:
```bash
ops-mcp-server --mode sse --host localhost --port 3000 --enable-events --enable-metrics --enable-logs
```

##### 2. Local Binary (STDIO Mode)

For direct execution of the local binary:

```json
{
  "mcpServers": {
    "local-stdio": {
      "command": "/bin/ops-mcp-server",
      "timeout": 600,
      "args": [
        "--mode",
        "stdio",
        "--enable-events"
      ]
    }
  }
}
```

Build the binary first:
```bash
make build
# Binary will be available at ./bin/ops-mcp-server
```

##### 3. Docker Container (STDIO Mode)

For containerized deployment using the published Docker image:

```json
{
  "mcpServers": {
    "local-docker-stdio": {
      "command": "docker",
      "timeout": 600,
      "args": [
        "run",
        "-i",
        "--rm",
        "shaowenchen/ops-mcp-server:latest",
        "--enable-events",
        "--enable-metrics",
        "--enable-logs",
        "--mode",
        "stdio"
      ]
    }
  }
}
```

