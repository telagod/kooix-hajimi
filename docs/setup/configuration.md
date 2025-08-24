# 配置指南

## 核心配置

### GitHub配置
```yaml
github:
  tokens: []  # 通过环境变量HAJIMI_GITHUB_TOKENS设置
  timeout: 30s
  max_retries: 5
```

### 扫描器配置
```yaml
scanner:
  worker_count: 20      # 并发工作线程数
  batch_size: 100       # 批处理大小
  scan_interval: 10s    # 连续扫描间隔
  date_range_days: 730  # 扫描文件的日期范围
  auto_start: false     # 启动时自动开始扫描
  query_file: "queries.txt"  # 查询规则文件
  
  # 文件过滤黑名单
  file_blacklist:
    - "test"
    - "spec"
    - "example"
    - "demo"
```

### 验证器配置
```yaml
validator:
  model_name: "gemini-2.5-flash"           # 验证模型
  tier_detection_model: "gemini-2.5-flash" # 层级检测模型
  worker_count: 5                          # 验证工作线程数
  timeout: 30s                             # 验证超时时间
  enable_tier_detection: true              # 启用层级检测
```

### 安全通知配置
```yaml
security_notifications:
  enabled: true              # 启用安全通知
  create_issues: true        # 自动创建GitHub issues
  notify_on_severity: "high" # 通知级别: all, high, critical
  dry_run: false            # 干运行模式
```

### 存储配置
```yaml
storage:
  type: "sqlite"              # sqlite 或 postgres
  dsn: "data/hajimi-king.db" # SQLite文件路径
  # PostgreSQL示例:
  # dsn: "postgres://user:pass@localhost:5432/hajimi_king"
```

### Web服务配置
```yaml
web:
  enabled: true
  host: "0.0.0.0"
  port: 8080
  cors_enabled: true
```

### 速率限制配置
```yaml
rate_limit:
  enabled: true
  requests_per_minute: 30    # 每分钟请求数
  burst_size: 10            # 突发请求数
  adaptive_enabled: true    # 自适应速率限制
  cooldown_duration: "5m"   # 冷却时间
  success_threshold: 0.8    # 成功率阈值
  backoff_multiplier: 2.0   # 退避倍数
```

## 环境变量

| 变量名 | 说明 | 默认值 | 必须 |
|--------|------|--------|------|
| `HAJIMI_GITHUB_TOKENS` | GitHub API Token(逗号分隔) | - | ✅ |
| `HAJIMI_LOG_LEVEL` | 日志级别(debug/info/warn/error) | info | - |
| `HAJIMI_WEB_PORT` | Web服务端口 | 8080 | - |
| `HAJIMI_STORAGE_TYPE` | 存储类型(sqlite/postgres) | sqlite | - |
| `HAJIMI_STORAGE_DSN` | 数据库连接字符串 | data/hajimi-king.db | - |

## 高级配置

### 生产环境配置
```yaml
# 高并发配置
scanner:
  worker_count: 50
  batch_size: 200
  
rate_limit:
  requests_per_minute: 100
  adaptive_enabled: true

# PostgreSQL配置
storage:
  type: "postgres"
  dsn: "postgres://user:password@localhost:5432/hajimi_king?sslmode=require"

# 日志配置
log:
  level: "info"
  output: "file"
  filename: "/app/logs/hajimi-king.log"
```

### 开发环境配置
```yaml
# 低资源配置
scanner:
  worker_count: 5
  batch_size: 50
  
validator:
  worker_count: 2

# 调试日志
log:
  level: "debug"
  output: "console"

# 测试模式
security_notifications:
  dry_run: true
```

## Web界面配置

所有配置都可以通过Web界面进行修改：

1. 访问 http://localhost:8080
2. 点击"设置"标签页
3. 修改配置参数
4. 点击"保存所有设置"

配置会立即生效，无需重启服务。