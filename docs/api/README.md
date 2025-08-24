# API文档

Kooix Hajimi 提供完整的RESTful API和WebSocket接口。

## 基础信息

- **Base URL**: `http://localhost:8080`
- **Content-Type**: `application/json`
- **响应格式**: 统一JSON格式

## 通用响应格式

```json
{
  "code": 0,           // 0: 成功, 非0: 错误
  "message": "success",// 响应消息
  "data": {}          // 响应数据
}
```

## 系统状态接口

### GET /api/status
获取系统运行状态

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "status": "running",
    "uptime": "2h30m15s",
    "version": "2.0.0"
  }
}
```

### GET /api/stats
获取系统统计信息

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "storage": {
      "valid_keys": 150,
      "rate_limited_keys": 23,
      "total_files_scanned": 5420
    },
    "scan": {
      "is_active": false,
      "total_queries": 100,
      "processed_queries": 100,
      "current_query": "",
      "processed_files": 1200
    }
  }
}
```

## 扫描控制接口

### POST /api/scan/start
开始扫描

**响应示例**:
```json
{
  "code": 0,
  "message": "Scan started successfully"
}
```

### POST /api/scan/stop
停止扫描

**响应示例**:
```json
{
  "code": 0,
  "message": "Scan stopped successfully"
}
```

### GET /api/scan/status
获取扫描状态

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "is_scanning": true,
    "start_time": "2024-01-20T10:30:00Z",
    "current_query": "AIzaSy language:python",
    "progress": {
      "total_queries": 100,
      "processed_queries": 45,
      "processed_files": 1230
    }
  }
}
```

## 密钥管理接口

### GET /api/keys/valid
获取有效密钥列表

**查询参数**:
- `limit`: 返回数量限制（默认20）
- `offset`: 偏移量（默认0）
- `provider`: 提供商过滤（gemini/openai/claude）
- `tier`: 层级过滤（free/paid/unknown）
- `repo`: 仓库名过滤
- `source`: 来源过滤

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "keys": [
      {
        "id": 1,
        "key": "AIzaSy...",
        "provider": "gemini",
        "key_type": "api_key",
        "tier": "paid",
        "tier_confidence": 0.95,
        "repo_name": "example/repo",
        "file_path": "config/api.py",
        "file_url": "https://github.com/example/repo/blob/main/config/api.py",
        "validated_at": "2024-01-20T10:30:00Z"
      }
    ],
    "total": 150
  }
}
```

### GET /api/keys/rate-limited
获取限流密钥列表

**查询参数**: 与有效密钥接口相同

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "keys": [
      {
        "id": 1,
        "key": "AIzaSy...",
        "provider": "gemini",
        "repo_name": "example/repo",
        "file_path": "src/main.py",
        "reason": "rate_limited",
        "created_at": "2024-01-20T10:30:00Z"
      }
    ],
    "total": 23
  }
}
```

### DELETE /api/keys/valid/:id
删除有效密钥

**响应示例**:
```json
{
  "code": 0,
  "message": "Key deleted successfully"
}
```

### DELETE /api/keys/rate-limited/:id
删除限流密钥

**响应示例**:
```json
{
  "code": 0,
  "message": "Key deleted successfully"
}
```

## 配置管理接口

### GET /api/config
获取当前配置

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "scanner": {
      "worker_count": 20,
      "batch_size": 100,
      "auto_start": false
    },
    "validator": {
      "model_name": "gemini-2.5-flash",
      "worker_count": 5,
      "enable_tier_detection": true
    },
    "security_notifications": {
      "enabled": true,
      "create_issues": true,
      "notify_on_severity": "high",
      "dry_run": false
    }
  }
}
```

### PUT /api/config
更新配置

**请求体示例**:
```json
{
  "scanner": {
    "worker_count": 30,
    "batch_size": 150
  },
  "security_notifications": {
    "enabled": true,
    "dry_run": true
  }
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "Configuration updated successfully",
  "data": {
    "note": "Some changes require restart to take effect"
  }
}
```

## 查询规则管理

### GET /api/queries
获取查询规则

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "content": "# Phase 1: Core Detection\nAIzaSy language:python\n...",
    "total_queries": 150
  }
}
```

### PUT /api/queries
更新查询规则

**请求体示例**:
```json
{
  "content": "# Updated queries\nAIzaSy language:javascript\n..."
}
```

### GET /api/queries/default
获取默认查询规则

## WebSocket接口

### WS /api/ws
实时数据推送

**连接**: `ws://localhost:8080/api/ws`

**消息类型**:

1. **统计更新**:
```json
{
  "type": "stats_update",
  "data": {
    "valid_keys": 151,
    "scan_progress": 75
  }
}
```

2. **扫描进度**:
```json
{
  "type": "scan_update",
  "data": {
    "current_query": "AIzaSy language:python",
    "processed_queries": 46,
    "total_queries": 100
  }
}
```

3. **日志条目**:
```json
{
  "type": "log_entry",
  "data": {
    "level": "info",
    "message": "Found 3 potential keys in repository",
    "timestamp": "2024-01-20T10:30:00Z"
  }
}
```

## 错误代码

| 代码 | 说明 |
|------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 资源不存在 |
| 1003 | 权限不足 |
| 2001 | 系统内部错误 |
| 2002 | 数据库错误 |
| 2003 | 网络错误 |
| 3001 | 扫描器忙碌 |
| 3002 | 配置错误 |

## 使用示例

### Python示例
```python
import requests

# 获取统计信息
response = requests.get('http://localhost:8080/api/stats')
data = response.json()
if data['code'] == 0:
    print(f"发现有效密钥: {data['data']['storage']['valid_keys']}")

# 开始扫描
response = requests.post('http://localhost:8080/api/scan/start')
if response.json()['code'] == 0:
    print("扫描已开始")
```

### JavaScript示例
```javascript
// 获取密钥列表
fetch('/api/keys/valid?limit=10&provider=gemini')
  .then(response => response.json())
  .then(data => {
    if (data.code === 0) {
      console.log('有效密钥数量:', data.data.total);
    }
  });

// WebSocket连接
const ws = new WebSocket('ws://localhost:8080/api/ws');
ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  if (data.type === 'stats_update') {
    console.log('统计更新:', data.data);
  }
};
```

### cURL示例
```bash
# 获取系统状态
curl http://localhost:8080/api/status

# 开始扫描
curl -X POST http://localhost:8080/api/scan/start

# 获取有效密钥
curl "http://localhost:8080/api/keys/valid?limit=5&provider=gemini"

# 更新配置
curl -X PUT http://localhost:8080/api/config \
  -H "Content-Type: application/json" \
  -d '{"scanner":{"worker_count":25}}'
```