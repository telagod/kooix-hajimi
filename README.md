# Kooix Hajimi

一个高性能的多提供商API密钥发现工具的Go重构版本，支持Gemini、OpenAI、Claude密钥发现和智能层级检测。

## 🚀 主要特性

### 🔑 多提供商API密钥支持
- **Gemini API**: `AIzaSy[A-Za-z0-9\-_]{33}` 模式识别
- **OpenAI API**: `sk-[A-Za-z0-9]{48}` 和 `sk-proj-[A-Za-z0-9]{48}` 模式
- **Claude API**: `sk-ant-api03-[A-Za-z0-9\-_]{95}AA` 模式
- **智能验证**: HTTP请求验证，成本优化的检测方案
- **错误分类**: 精确识别无效密钥、限流状态、账户禁用等
- **层级检测**: 自动识别Gemini免费/付费账户，优先使用付费密钥

### 性能提升
- **高并发扫描**: 使用goroutines实现真正的并发处理
- **智能限流**: 自适应限流算法，最大化API利用率
- **内存优化**: 低内存占用，支持大规模扫描
- **快速部署**: 单二进制文件，秒级启动

### 🌐 Web界面和配置管理
- **实时Web界面**: 现代化仪表板和设置页面
- **无需环境变量**: Web界面直接配置，实时生效
- **多存储支持**: SQLite、PostgreSQL支持
- **WebSocket实时更新**: 实时状态和进度推送
- **RESTful API**: 完整的API接口
- **密钥管理**: 层级显示、批量操作、智能过滤

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
  
  # 验证器配置 🆕
  validator:
    model_name: "gemini-2.5-flash"           # 验证模型
    tier_detection_model: "gemini-2.5-flash" # 层级检测模型
    worker_count: 5                          # 验证器worker数
    timeout: 30s                             # 验证超时
    enable_tier_detection: true              # 启用层级检测

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

# 限流配置
rate_limit:
  enabled: true
  requests_per_minute: 30
  adaptive_enabled: true    # 自适应限流
```

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `HAJIMI_GITHUB_TOKENS` | GitHub API Token(逗号分隔) | 必填 |
| `HAJIMI_LOG_LEVEL` | 日志级别 | info |
| `HAJIMI_WEB_PORT` | Web服务端口 | 8080 |
| `HAJIMI_SCANNER_WORKER_COUNT` | 扫描并发数 | 20 |
| `HAJIMI_VALIDATOR_MODEL_NAME` | 验证模型 | gemini-2.5-flash |
| `HAJIMI_ENABLE_TIER_DETECTION` | 启用层级检测 | true |

## 🖥️ Web界面功能

### 仪表板
- 实时扫描状态监控
- 密钥发现统计图表
- 系统资源使用情况
- 最近发现的密钥列表

### 密钥管理
- 多提供商密钥列表和详情
- 智能层级显示（免费/付费/未知）
- 限流密钥管理
- 批量操作和搜索
- 按层级和提供商过滤

### 🖥️ Web界面功能

### 仪表板
- 实时扫描状态监控
- 多提供商密钥发现统计图表
- 系统资源使用情况
- 最近发现的密钥列表

### 密钥管理
- **多提供商密钥**: Gemini、OpenAI、Claude密钥统一管理
- **智能层级显示**: 免费/付费/未知层级，置信度显示
- **提供商过滤**: 按API提供商类型筛选
- **批量操作**: 多选删除、导出等功能
- **搜索功能**: 按仓库名、文件路径搜索

### 设置页面 🆕
- **验证器设置**: 验证模型、层级检测模型、worker配置
- **扫描器设置**: 并发数、批次大小、扫描间隔、日期范围
- **限流设置**: 请求频率、突发大小、自适应限流
- **层级筛选**: 优先使用付费密钥、按层级过滤
- **实时配置**: 配置更改立即生效，无需重启服务器

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

## 🆕 新功能亮点

### 多提供商API密钥支持
- **全面支持**: 支持Gemini、OpenAI、Claude三大主流AI服务商
- **智能检测**: 使用优化的正则表达式和HTTP验证
- **统一管理**: 单一界面管理所有类型的API密钥
- **错误分类**: 精确区分无效、限流、禁用等状态

### 智能层级检测系统
- **自动识别**: 通过模型访问测试区分免费/付费Gemini账户
- **置信度评估**: 提供检测结果的可信度评分
- **优先级管理**: 优先使用付费密钥，提高成功率
- **可配置检测**: 支持自定义检测模型和参数

### Web配置管理
- **零依赖配置**: 完全通过Web界面配置，无需修改环境变量
- **实时生效**: 配置更改立即生效，无需重启服务
- **可视化设置**: 直观的表单界面，支持参数验证
- **配置持久化**: 自动保存到配置文件，重启后保持

## 📊 性能对比

| 指标 | Python版本 | Go版本 | 提升 |
|------|------------|--------|------|
| 内存使用 | ~500MB | ~100MB | 5x |
| 并发处理 | 单线程 | 多goroutine | 20x |
| 启动时间 | ~5s | ~0.5s | 10x |
| 扫描速度 | 基准 | 5-10x | 5-10x |
| 部署大小 | ~200MB | ~50MB | 4x |
| API提供商 | 仅Gemini | Gemini/OpenAI/Claude | 3x |
| 层级检测 | 无 | 智能检测 | ∞ |

## 🔧 API接口

### 系统状态
- `GET /api/status` - 系统状态
- `GET /api/stats` - 统计信息

### 扫描控制
- `POST /api/scan/start` - 开始扫描
- `POST /api/scan/stop` - 停止扫描
- `GET /api/scan/status` - 扫描状态

### 密钥管理
- `GET /api/keys/valid` - 获取有效密钥（支持提供商、层级过滤）
- `GET /api/keys/rate-limited` - 获取限流密钥
- `DELETE /api/keys/valid/:id` - 删除密钥

### 配置管理 🆕
- `GET /api/config` - 获取当前配置
- `PUT /api/config` - 更新配置（实时生效）

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