# Kooix Hajimi

一个高性能的GitHub API密钥发现工具的Go重构版本，具备Web界面和现代化架构。

## 🚀 主要特性

### 性能提升
- **高并发扫描**: 使用goroutines实现真正的并发处理
- **智能限流**: 自适应限流算法，最大化API利用率
- **内存优化**: 低内存占用，支持大规模扫描
- **快速部署**: 单二进制文件，秒级启动

### 功能增强
- **实时Web界面**: 现代化仪表板和监控
- **多存储支持**: SQLite、PostgreSQL支持
- **WebSocket实时更新**: 实时状态和进度推送
- **RESTful API**: 完整的API接口
- **配置热更新**: 无需重启修改配置

### 架构改进
- **模块化设计**: 清晰的分层架构
- **可扩展性**: 支持水平扩展部署
- **监控完善**: 详细的指标和日志
- **容器优化**: 优化的Docker镜像

## 📁 项目结构

```
kooix-hajimi/
├── cmd/                    # 应用入口
│   ├── server/            # Web服务器
│   └── cli/               # 命令行工具
├── internal/              # 内部包
│   ├── config/           # 配置管理
│   ├── github/           # GitHub API客户端
│   ├── scanner/          # 扫描器核心
│   ├── storage/          # 数据存储层
│   ├── validator/        # 密钥验证器
│   ├── ratelimit/        # 限流管理
│   ├── sync/            # 外部同步
│   └── web/             # Web服务
├── pkg/                   # 公共包
│   ├── logger/          # 日志工具
│   └── utils/           # 通用工具
├── web/                   # Web资源
│   ├── static/          # 静态文件
│   └── templates/       # HTML模板
├── configs/              # 配置文件
├── scripts/             # 构建脚本
└── docs/                # 文档
```

## 🛠️ 快速开始

### 环境要求
- Go 1.21+
- Docker (可选)
- SQLite3

### 本地开发

1. **克隆项目**
```bash
git clone <repo-url>
cd kooix-hajimi
```

2. **配置环境**
```bash
# 复制配置文件
cp configs/config.yaml.example configs/config.yaml

# 设置GitHub Token
export HAJIMI_GITHUB_TOKENS="your_token_1,your_token_2"
```

3. **安装依赖**
```bash
go mod tidy
```

4. **运行服务**
```bash
# 开发模式
go run cmd/server/main.go

# 或使用构建脚本
./scripts/build.sh all
./build/hajimi-king-server
```

5. **访问界面**
打开浏览器访问: http://localhost:8080

### Docker部署

#### 使用GitHub Container Registry

**从GitHub Container Registry拉取镜像：**
```bash
# 拉取最新镜像
docker pull ghcr.io/your-username/kooix-hajimi:latest

# 拉取指定版本
docker pull ghcr.io/your-username/kooix-hajimi:v1.0.0
```

**运行容器：**
```bash
# 设置环境变量
export GITHUB_TOKENS="your_token_1,your_token_2"

# 单独运行
docker run -d \
  --name kooix-hajimi \
  -p 8080:8080 \
  -e HAJIMI_GITHUB_TOKENS="$GITHUB_TOKENS" \
  -v ./data:/app/data \
  ghcr.io/your-username/kooix-hajimi:latest
```

#### 使用docker-compose

1. **修改docker-compose.yml镜像地址：**
```yaml
services:
  kooix-hajimi:
    image: ghcr.io/your-username/kooix-hajimi:latest
    # ... 其他配置
```

2. **启动服务：**
```bash
# 设置环境变量
export GITHUB_TOKENS="your_token_1,your_token_2"

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

#### 自动构建

**GitHub Actions自动构建：**
- ✅ 推送到 `main`/`develop` 分支时自动构建
- ✅ 发布标签时自动构建版本镜像
- ✅ 支持多平台镜像 (AMD64/ARM64)
- ✅ 发布到 GitHub Container Registry (ghcr.io)
- ✅ 无需配置额外secrets，使用GitHub原生支持

## ⚙️ 配置说明

### 核心配置

```yaml
# GitHub配置
github:
  tokens: []  # 从环境变量读取
  timeout: 30s
  max_retries: 5

# 扫描器配置
scanner:
  worker_count: 20      # 并发工作数
  batch_size: 100       # 批处理大小
  scan_interval: 10s    # 扫描间隔
  auto_start: false     # 自动启动

# Web服务配置  
web:
  enabled: true
  host: "0.0.0.0"
  port: 8080
  cors_enabled: true

# 存储配置
storage:
  type: "sqlite"        # sqlite, postgres
  dsn: "data/hajimi-king.db"
```

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `HAJIMI_GITHUB_TOKENS` | GitHub API Token(逗号分隔) | 必填 |
| `HAJIMI_LOG_LEVEL` | 日志级别 | info |
| `HAJIMI_WEB_PORT` | Web服务端口 | 8080 |
| `HAJIMI_SCANNER_WORKER_COUNT` | 扫描并发数 | 20 |

## 🖥️ Web界面功能

### 仪表板
- 实时扫描状态监控
- 密钥发现统计图表
- 系统资源使用情况
- 最近发现的密钥列表

### 密钥管理
- 有效密钥列表和详情
- 限流密钥管理
- 批量操作和搜索
- 导出功能

### 扫描控制
- 一键启动/停止扫描
- 扫描进度实时跟踪
- 配置参数调整
- 查询表达式管理

### 日志监控
- 实时日志流
- 日志级别过滤
- 错误统计和告警
- 系统健康检查

## 📊 性能对比

| 指标 | Python版本 | Go版本 | 提升 |
|------|------------|--------|------|
| 内存使用 | ~500MB | ~100MB | 5x |
| 并发处理 | 单线程 | 多goroutine | 20x |
| 启动时间 | ~5s | ~0.5s | 10x |
| 扫描速度 | 基准 | 5-10x | 5-10x |
| 部署大小 | ~200MB | ~50MB | 4x |

## 🔧 API接口

### 系统状态
- `GET /api/status` - 系统状态
- `GET /api/stats` - 统计信息

### 扫描控制
- `POST /api/scan/start` - 开始扫描
- `POST /api/scan/stop` - 停止扫描
- `GET /api/scan/status` - 扫描状态

### 密钥管理
- `GET /api/keys/valid` - 获取有效密钥
- `GET /api/keys/rate-limited` - 获取限流密钥
- `DELETE /api/keys/valid/:id` - 删除密钥

### WebSocket
- `WS /api/ws` - 实时数据推送

## 🚀 部署建议

### 生产环境
```bash
# 使用GitHub Container Registry镜像
docker run -d \
  --name kooix-hajimi \
  -p 8080:8080 \
  -e HAJIMI_GITHUB_TOKENS="your_tokens" \
  -e HAJIMI_STORAGE_TYPE="postgres" \
  -e HAJIMI_STORAGE_DSN="postgres://..." \
  ghcr.io/your-username/kooix-hajimi:latest

# 或使用PostgreSQL compose配置
docker-compose --profile postgres up -d
```

### 高可用部署
- 使用PostgreSQL集群
- Redis缓存分布式锁
- 负载均衡多实例
- Prometheus监控

### 监控告警
```yaml
# docker-compose.monitoring.yml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus
    # ... 配置省略
  
  grafana:
    image: grafana/grafana
    # ... 配置省略
```

## 🔒 安全建议

1. **Token管理**
   - 定期轮换GitHub Token
   - 使用最小权限原则
   - 环境变量存储敏感信息

2. **网络安全**  
   - 启用HTTPS
   - 配置防火墙
   - API访问限制

3. **数据安全**
   - 数据库加密
   - 备份策略
   - 访问日志审计

## 📈 监控指标

- 扫描进度和速度
- API请求成功率
- 内存和CPU使用率
- 数据库连接状态
- Token限流状态

## 🤝 贡献指南

1. Fork项目
2. 创建功能分支
3. 提交变更
4. 发起Pull Request

## 📄 许可证

MIT License

## 🆘 支持

- 问题反馈: [GitHub Issues](https://github.com/your-repo/issues)
- 文档: [在线文档](https://docs.your-domain.com)
- 社区: [Discussion](https://github.com/your-repo/discussions)