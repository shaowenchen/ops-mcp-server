# Prometheus Metrics 自动采集配置

本文档说明如何配置 Prometheus 自动采集 ops-mcp-server 的 metrics。

## 配置方式

### 方式 1: 使用 Kubernetes Annotations（推荐）

这是最简单的方式，适用于标准的 Prometheus 安装。

#### Service 配置

Service 中已经添加了以下 annotations：

```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "80"
    prometheus.io/path: "/mcp/metrics"
    prometheus.io/scheme: "http"
```

#### Pod 配置

Deployment 的 Pod template 中也添加了相同的 annotations，这样 Prometheus 可以直接从 Pod 发现并抓取 metrics。

#### Prometheus 配置

在你的 `prometheus.yml` 中添加以下配置：

```yaml
scrape_configs:
  - job_name: 'ops-mcp-server'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - ops-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        target_label: __address__
        regex: (.+)
        replacement: $1:__meta_kubernetes_pod_annotation_prometheus_io_port
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__meta_kubernetes_pod_name]
        target_label: pod
      - source_labels: [__meta_kubernetes_namespace]
        target_label: namespace
```

### 方式 2: 使用 ServiceMonitor（Prometheus Operator）

如果你使用 Prometheus Operator，可以使用 ServiceMonitor 资源。

#### 部署 ServiceMonitor

```bash
kubectl apply -f deploy/servicemonitor.yaml
```

#### 配置说明

ServiceMonitor 会自动被 Prometheus Operator 发现并配置到 Prometheus 中。

**注意**: 确保 ServiceMonitor 的 `spec.selector.matchLabels` 与 Service 的 labels 匹配，并且 `release` label 与你的 Prometheus Operator release 名称匹配。

### 方式 3: 静态配置

如果上述自动发现方式不适用，可以使用静态配置：

```yaml
scrape_configs:
  - job_name: 'ops-mcp-server'
    static_configs:
      - targets:
        - 'ops-mcp-server.ops-system.svc.cluster.local:80'
    metrics_path: '/mcp/metrics'
    scrape_interval: 15s
    scrape_timeout: 10s
```

## Metrics Endpoint

Metrics 端点路径取决于你的 MCP URI 配置：

- 默认路径: `/mcp/metrics`
- 如果配置了自定义 URI，路径为: `{your_uri}/metrics`

例如，如果 `SERVER_URI=/api/mcp`，则 metrics 路径为 `/api/mcp/metrics`。

## 验证配置

### 1. 检查 Metrics 端点是否可访问

```bash
# 从集群内部
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://ops-mcp-server.ops-system.svc.cluster.local:80/mcp/metrics

# 从 Pod 内部
kubectl exec -it <pod-name> -n ops-system -- \
  curl http://localhost:80/mcp/metrics
```

### 2. 检查 Prometheus 是否在抓取

在 Prometheus UI 中：
1. 访问 `http://prometheus:9090/targets`
2. 查找 `ops-mcp-server` job
3. 确认状态为 "UP"

### 3. 查询 Metrics

在 Prometheus UI 中尝试查询：

```promql
# HTTP 请求总数
http_requests_total

# SSE 活跃连接数
sse_active_connections

# MCP 工具调用率
rate(mcp_tool_calls_total[5m])
```

## 常见问题

### Q: Prometheus 无法发现 Pod

**A**: 检查以下几点：
1. Pod annotations 是否正确设置
2. Prometheus 的 `kubernetes_sd_configs` 配置是否正确
3. Prometheus 是否有权限访问目标 namespace

### Q: Metrics 端点返回 404

**A**: 检查：
1. MCP URI 配置是否正确
2. Metrics endpoint 路径是否正确
3. 服务器是否在 SSE 模式下运行（stdio 模式不提供 HTTP endpoints）

### Q: ServiceMonitor 不生效

**A**: 检查：
1. Prometheus Operator 是否已安装
2. ServiceMonitor 的 `release` label 是否匹配
3. ServiceMonitor 的 selector 是否匹配 Service labels
4. Prometheus Operator 是否有权限访问 ServiceMonitor

## 相关文件

- `deploy/service.yaml` - Service 配置（包含 annotations）
- `deploy/deployment.yaml` - Deployment 配置（包含 Pod annotations）
- `deploy/servicemonitor.yaml` - ServiceMonitor 配置（Prometheus Operator）
- `deploy/prometheus-scrape-config.yaml` - Prometheus 配置示例
- `grafana/dashboard.json` - Grafana Dashboard
- `grafana/README.md` - Dashboard 使用说明

## 参考文档

- [Prometheus Kubernetes Service Discovery](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#kubernetes_sd_config)
- [Prometheus Operator ServiceMonitor](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/user-guides/getting-started.md)
- [Prometheus Annotations](https://github.com/prometheus/prometheus/blob/main/documentation/examples/prometheus-kubernetes.yml)

