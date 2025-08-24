# Kooix Hajimi 文档

欢迎来到 Kooix Hajimi 的完整文档。这里包含了安装、配置、部署和使用的详细指南。

## 📖 文档导航

### 🚀 快速开始
- **[安装指南](setup/installation.md)** - 从源码或Docker快速安装
- **[配置指南](setup/configuration.md)** - 详细的配置参数说明

### 🔒 安全配置
- **[GitHub权限配置](security/github-permissions.md)** - GitHub Token权限详解
- **[安全通知系统](security/notifications.md)** - 自动安全通知配置

### 🛠 开发和部署
- **[API文档](api/README.md)** - REST API和WebSocket接口
- **[部署指南](deployment/README.md)** - Docker、Kubernetes部署方案

### 💻 开发指南
- **[开发环境搭建](development/setup.md)** - 本地开发环境配置
- **[架构说明](development/architecture.md)** - 系统架构和设计原理
- **[贡献指南](development/contributing.md)** - 如何参与项目开发

## 🎯 主要功能

### 多提供商密钥支持
- **Gemini API**: `AIzaSy[A-Za-z0-9\-_]{33}` 模式
- **OpenAI API**: `sk-[A-Za-z0-9]{48}` 和项目密钥
- **Claude API**: `sk-ant-api03-[A-Za-z0-9\-_]{95}AA` 模式

### 安全通知系统
- 自动GitHub issue创建
- 智能严重级别分类
- 干运行测试模式
- 详细的安全修复指导

### 高性能扫描
- Go并发处理，5-10倍性能提升
- 智能限流和自适应算法
- 分阶段查询优化
- 智能去重和缓存

### 现代化界面
- 实时Web仪表板
- 中英文双语支持
- WebSocket实时更新
- 零配置管理

## 🚀 快速链接

| 需求 | 推荐文档 |
|------|----------|
| 快速体验 | [Docker安装](setup/installation.md#docker安装) |
| 生产部署 | [部署指南](deployment/README.md#生产环境部署) |
| 权限配置 | [GitHub权限](security/github-permissions.md) |
| 接口开发 | [API文档](api/README.md) |
| 问题排查 | [故障排除](deployment/README.md#故障排除) |

## 📊 性能基准

### 硬件要求
- **最低配置**: 1 CPU, 512MB RAM, 1GB 存储
- **推荐配置**: 2 CPU, 2GB RAM, 10GB 存储  
- **高负载配置**: 4+ CPU, 8GB+ RAM, SSD存储

### 性能指标
- **扫描速度**: 每分钟处理100-1000个查询
- **内存使用**: 50-200MB（取决于并发数）
- **启动时间**: < 1秒
- **API响应**: < 100ms

## 🔧 配置示例

### 基础配置
```yaml
github:
  tokens: []  # 通过环境变量设置

scanner:
  worker_count: 20
  batch_size: 100

security_notifications:
  enabled: true
  dry_run: true  # 建议先测试
```

### 生产配置
```yaml
scanner:
  worker_count: 50
  batch_size: 200

storage:
  type: "postgres"
  dsn: "postgres://user:pass@localhost/hajimi"

security_notifications:
  notify_on_severity: "high"
  dry_run: false
```

## 🆘 获取帮助

- **GitHub Issues**: [提交问题](https://github.com/telagod/kooix-hajimi/issues)
- **讨论区**: [GitHub Discussions](https://github.com/telagod/kooix-hajimi/discussions)
- **邮件支持**: 发送邮件至项目维护者

## 📝 更新日志

查看 [CHANGELOG.md](../CHANGELOG.md) 了解最新版本的更新内容。

## 🤝 贡献

我们欢迎任何形式的贡献：

- 🐛 报告Bug
- 💡 提出新功能建议
- 📝 改进文档
- 🔧 提交代码

查看 [贡献指南](development/contributing.md) 了解详细流程。

---

**最后更新**: 2024年1月20日  
**文档版本**: 2.0