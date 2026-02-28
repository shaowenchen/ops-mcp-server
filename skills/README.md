# Ops MCP Server Skill

A comprehensive operational observability skill that provides AI assistants with access to infrastructure operational data through the Model Context Protocol (MCP).

## 📁 Directory Structure

```
ops-mcp-server/
├── SKILL.md          # 核心指令 + 元数据（283 行）
│
├── examples/         # 实用示例（给 Claude 看标准用法）
│   ├── README.md     # 示例总览
│   ├── events.md     # Kubernetes events 查询示例
│   ├── metrics.md    # Prometheus metrics 查询示例
│   ├── logs.md       # Elasticsearch logs 查询示例
│   ├── traces.md     # Jaeger traces 查询示例
│   └── sops.md       # SOPS 执行示例
│
└── references/       # 规范、设计文档
    ├── README.md
    └── design.md     # 事件格式规范和架构设计
```

## 🚀 Quick Start

See [SKILL.md](SKILL.md) for installation and usage instructions.

## 📚 Documentation

- **[SKILL.md](SKILL.md)** - Main skill guide with setup and usage
- **[examples/](examples/)** - Practical examples for all MCP tools
- **[references/](references/)** - Technical specifications and design docs

## 🎯 What This Skill Does

- **Monitor Kubernetes**: Track pods, deployments, and cluster events
- **Query Metrics**: Access Prometheus metrics with PromQL
- **Analyze Logs**: Search Elasticsearch with ES|QL and Query DSL
- **Trace Performance**: Investigate Jaeger distributed traces
- **Execute SOPs**: Run standardized operational procedures

## 🔧 Prerequisites

- mcporter CLI (`npm i -g mcporter`)
- MCP server connection configured
- Access to ops infrastructure endpoints

See [SKILL.md](SKILL.md) for detailed setup.
