# Kooix Hajimi

一个高性能的多提供商API密钥发现和安全通知工具，使用Go重写，支持Gemini、OpenAI、Claude密钥发现、智能层级检测和自动安全通知。

## ✨ 主要特性

- 🔐 **多提供商支持**: Gemini、OpenAI、Claude密钥自动发现和验证
- 🎯 **安全通知系统**: 发现密钥泄露时自动创建GitHub issue提醒
- 🧠 **智能层级检测**: 自动识别免费/付费账户，优先使用付费密钥
- 🚀 **高性能扫描**: Go并发处理，5-10倍性能提升
- 🌐 **现代Web界面**: 实时仪表板，中英文支持，WebSocket更新
- ⚙️ **零配置管理**: Web界面直接配置，实时生效

## 🚀 快速开始

### Docker部署（推荐）

```bash
# 拉取最新镜像
docker pull ghcr.io/telagod/kooix-hajimi:latest

# 设置GitHub Token
export GITHUB_TOKENS="your_token_1,your_token_2"

# 运行容器
docker run -d \
  --name kooix-hajimi \
  -p 8080:8080 \
  -e HAJIMI_GITHUB_TOKENS="$GITHUB_TOKENS" \
  -v ./data:/app/data \
  ghcr.io/telagod/kooix-hajimi:latest
```

### 源码安装

```bash
# 克隆项目
git clone https://github.com/telagod/kooix-hajimi.git
cd kooix-hajimi

# 构建运行
./scripts/build.sh all
export HAJIMI_GITHUB_TOKENS="your_tokens"
./build/hajimi-server
```

### 访问界面

打开浏览器访问: http://localhost:8080

## 📚 文档

| 文档 | 说明 |
|------|------|
| [安装指南](docs/setup/installation.md) | 详细安装步骤和环境配置 |
| [配置指南](docs/setup/configuration.md) | 完整配置参数说明 |
| [GitHub权限](docs/security/github-permissions.md) | GitHub Token权限配置 |
| [API文档](docs/api/README.md) | REST API和WebSocket接口 |
| [部署指南](docs/deployment/README.md) | Docker、K8s等部署方案 |

## 🔑 GitHub Token权限

### 基础扫描功能
- ✅ `public_repo` - 搜索公共仓库
- ✅ `read:user` - API配额管理

### 安全通知功能（可选）
- ⚠️ `repo` - 创建安全警告issue
- ⚠️ `write:issues` - issue管理权限

> **重要**: 安全通知功能会在发现密钥的仓库中自动创建public issue。建议先使用`dry_run: true`模式测试。

## ⚙️ 核心配置

```yaml
# GitHub配置
github:
  tokens: []  # 通过HAJIMI_GITHUB_TOKENS环境变量设置

# 扫描器配置  
scanner:
  worker_count: 20
  batch_size: 100
  auto_start: false

# 安全通知配置
security_notifications:
  enabled: true              # 启用安全通知
  create_issues: true        # 自动创建GitHub issues
  notify_on_severity: "high" # 通知级别: all, high, critical  
  dry_run: false            # 测试模式

# 验证器配置
validator:
  model_name: "gemini-2.5-flash"
  enable_tier_detection: true  # 启用层级检测
```

## 🔒 安全特性

### 严重级别分类
- 🔴 **Critical**: AWS、GitHub、Stripe等高风险服务
- 🟠 **High**: OpenAI、Gemini、Claude等AI服务  
- 🟡 **Medium**: 其他API服务

### 智能通知策略
- **干运行模式**: 测试配置而不创建真实issue
- **级别过滤**: 可配置只对特定级别创建通知
- **详细模板**: 提供专业的安全修复指导

## 📊 性能对比

| 指标 | Python版本 | Go版本 | 提升 |
|------|------------|--------|------|
| 内存使用 | ~500MB | ~100MB | 5x |
| 并发处理 | 单线程 | 多goroutine | 20x |
| 启动时间 | ~5s | ~0.5s | 10x |
| 扫描速度 | 基准 | 5-10x | 5-10x |
| API提供商 | 仅Gemini | 多提供商 | 3x |

## 🔧 主要接口

### REST API
- `GET /api/stats` - 系统统计
- `POST /api/scan/start` - 开始扫描  
- `GET /api/keys/valid` - 有效密钥列表
- `PUT /api/config` - 更新配置

### WebSocket
- `WS /api/ws` - 实时数据推送

## 🚀 部署方案

### 开发环境
```bash
docker-compose up -d
```

### 生产环境
```bash
# PostgreSQL + 高可用
docker-compose --profile postgres up -d
```

### Kubernetes
参见 [部署指南](docs/deployment/README.md) 中的K8s配置。

## 🤝 贡献

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

MIT License - 查看 [LICENSE](LICENSE) 文件了解详情

## 🆘 支持

- 🐛 问题反馈: [GitHub Issues](https://github.com/telagod/kooix-hajimi/issues)
- 💬 讨论交流: [GitHub Discussions](https://github.com/telagod/kooix-hajimi/discussions)
- 📖 在线文档: [docs/](docs/)

---

**⚠️ 免责声明**: 此工具仅用于安全研究和漏洞披露。使用者需自行承担使用责任，确保遵守相关法律法规。