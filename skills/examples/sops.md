# SOPS Operations Examples

Execute standardized operational procedures (SOPS) safely and consistently.

## Available Tools

### 1. list-sops-from-ops

List all available SOPS procedures.

**Parameters:** None

### 2. list-sops-parameters-from-ops

Get required parameters for a specific SOPS procedure.

**Parameters:**

- `sops_id` (required, string): ID of the SOPS procedure

### 3. execute-sops-from-ops

Execute a SOPS procedure.

**Parameters:**

- `sops_id` (required, string): ID of the SOPS procedure to execute
- `parameters` (optional, string): JSON string of parameters

## What are SOPS?

SOPS (Standard Operating Procedures) are predefined operational tasks that:

- Ensure consistent execution across teams
- Reduce human error
- Provide audit trails
- Enable safe automation

## Example 1: List Available SOPS

```bash
# List all SOPS procedures
mcporter call ops-mcp-server-mcp list-sops-from-ops
```

### Expected Response

```json
{
  "sops": [
    {
      "id": "pod-restart",
      "name": "Restart Pod",
      "description": "Safely restart a pod in a namespace",
      "category": "kubernetes"
    },
    {
      "id": "scale-deployment",
      "name": "Scale Deployment",
      "description": "Scale deployment replicas",
      "category": "kubernetes"
    },
    {
      "id": "clear-cache",
      "name": "Clear Cache",
      "description": "Clear application cache",
      "category": "maintenance"
    }
  ]
}
```

## Example 2: Get SOPS Parameters

```bash
# Get parameters for pod-restart
mcporter call ops-mcp-server-mcp list-sops-parameters-from-ops sops_id="pod-restart"

# Get parameters for scale-deployment
mcporter call ops-mcp-server-mcp list-sops-parameters-from-ops sops_id="scale-deployment"
```

### Expected Response

```json
{
  "sops_id": "pod-restart",
  "parameters": [
    {
      "name": "namespace",
      "type": "string",
      "required": true,
      "description": "Kubernetes namespace (e.g., kube-system)"
    },
    {
      "name": "pod_name",
      "type": "string",
      "required": true,
      "description": "Name of the pod to restart (e.g., calico-node-abc123)"
    },
    {
      "name": "force",
      "type": "boolean",
      "required": false,
      "default": false,
      "description": "Force restart even if unhealthy"
    }
  ]
}
```

## Example 3: Execute SOPS

```bash
# Execute pod restart
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="pod-restart" parameters='{"namespace":"kube-system","pod_name":"calico-node-abc123"}'

# Scale deployment
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="scale-deployment" parameters='{"namespace":"kube-system","deployment":"coredns","replicas":5}'

# Clear cache
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="clear-cache" parameters='{"environment":"kube-system","cache_type":"redis"}'
```

## Parameters Format

SOPS parameters must be a JSON string:

```json
// Simple parameters
"{\"namespace\":\"kube-system\",\"pod_name\":\"calico-node-abc123\"}"

// With optional parameters
"{\"namespace\":\"kube-system\",\"pod_name\":\"coredns-123\",\"force\":true}"

// Complex parameters
"{\"namespace\":\"kube-system\",\"deployment\":\"coredns\",\"replicas\":5,\"wait_for_ready\":true,\"timeout\":\"5m\"}"

// Nested parameters
"{\"namespace\":\"kube-system\",\"service\":\"kube-dns\",\"config\":{\"memory_limit\":\"2Gi\",\"cpu_limit\":\"1000m\"}}"
```

## Expected Response

### Execution Success

```json
{
  "sops_id": "pod-restart",
  "status": "success",
  "execution_id": "exec-12345",
  "timestamp": "2024-01-15T10:30:00Z",
  "result": {
    "message": "Pod calico-node-abc123 restarted successfully",
    "old_pod": "calico-node-abc123",
    "new_pod": "calico-node-def456",
    "restart_time": "2024-01-15T10:30:05Z"
  }
}
```

### Execution Failure

```json
{
  "sops_id": "pod-restart",
  "status": "failed",
  "execution_id": "exec-12346",
  "timestamp": "2024-01-15T10:31:00Z",
  "error": {
    "code": "PERMISSION_DENIED",
    "message": "Insufficient permissions to restart pod in kube-system namespace"
  }
}
```

## Troubleshooting

### SOPS Not Found

**Problem:** SOPS ID not found

**Solutions:**

1. List all SOPS: `list-sops-from-ops`
2. Check SOPS ID spelling
3. Verify SOPS module is enabled

### Missing Required Parameters

**Problem:** Execution fails due to missing parameters

**Solutions:**

1. Get parameter list: `list-sops-parameters-from-ops`
2. Check all required parameters are provided
3. Verify parameter names match exactly

### Invalid Parameter Format

**Problem:** JSON parse error

**Solutions:**

1. Ensure parameters is valid JSON string
2. Escape quotes properly
3. Use correct data types (string, number, boolean)

```json
// Correct
"{\"namespace\":\"kube-system\",\"replicas\":5}"

// Incorrect - missing quotes around JSON
{namespace:kube-system,replicas:5}

// Incorrect - wrong data type
"{\"namespace\":\"kube-system\",\"replicas\":\"5\"}"
```

### Permission Denied

**Problem:** Execution fails with permission error

**Solutions:**

1. Verify ops server authentication token
2. Check RBAC permissions for the operation
3. Ensure service account has necessary rights

## Safety Best Practices

## Real-World Scenarios

### Scenario 1: Incident Response

```bash
# Step 1: List available procedures
mcporter call ops-mcp-server-mcp list-sops-from-ops

# Step 2: Get parameters
mcporter call ops-mcp-server-mcp list-sops-parameters-from-ops sops_id="pod-restart"

# Step 3: Execute
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="pod-restart" parameters='{"namespace":"kube-system","pod_name":"calico-node-abc123"}'
```

### Scenario 2: Scheduled Maintenance

```bash
# Backup database
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="db-backup" parameters='{"database":"kube-system"}'

# Scale down
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="scale-deployment" parameters='{"namespace":"kube-system","deployment":"coredns","replicas":0}'

# Migrate database
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="db-migrate" parameters='{"database":"kube-system"}'

# Scale up
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="scale-deployment" parameters='{"namespace":"kube-system","deployment":"coredns","replicas":3}'
```

### Scenario 3: Load Testing Preparation

```bash
# Scale up deployment
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="scale-deployment" parameters='{"namespace":"kube-system","deployment":"coredns","replicas":20}'

# Increase resources
mcporter call ops-mcp-server-mcp execute-sops-from-ops \
  sops_id="increase-resources" parameters='{"namespace":"kube-system","deployment":"coredns","cpu":"2000m","memory":"4Gi"}'
```

## SOPS Naming Conventions

Common SOPS naming patterns:

- **Action-Resource**: `restart-pod`, `scale-deployment`, `delete-job`
- **Category**: `k8s-pod-restart`, `db-backup-mysql`, `cache-clear-redis`
- **Emergency**: `emergency-stop`, `rollback-deployment`, `failover-database`

## Parameter Types

Common parameter types:

- **string**: `"kube-system"`, `"calico-node-abc123"`
- **number**: `5`, `1024`, `3.14`
- **boolean**: `true`, `false`
- **array**: `["pod-1", "pod-2"]`
- **object**: `{"cpu": "1000m", "memory": "2Gi"}`

## Reference

- **Tools**: `list-sops-from-ops`, `list-sops-parameters-from-ops`, `execute-sops-from-ops`
- **JSON Format**: <https://www.json.org/>
- **Parameter Types**: string, number, boolean, array, object
- **Best Practice**: Always verify, document, and have rollback plan

## Important Notes

- ⚠️ **Production Safety**: Always verify before executing SOPS in production
- 🔐 **Access Control**: SOPS execution may require special permissions
- 📝 **Audit Trail**: All executions are logged for compliance
- 🔁 **Idempotency**: Most SOPS are designed to be safely re-executed
- 📋 **Documentation**: Keep reason and context for each execution
