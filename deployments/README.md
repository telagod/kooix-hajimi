# Kooix Hajimi - 部署指南

Kooix Hajimi 提供三种灵活的部署方式，适应不同的使用场景和基础设施需求。

## 🎯 部署方式对比

| 部署方式 | 适用场景 | 复杂度 | 性能 | 可扩展性 | 维护成本 |
|---------|---------|--------|------|-----------|----------|
| **快速部署** | 个人使用、快速测试 | ⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐ |
| **生产级部署** | 企业级、高负载 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **本机服务部署** | 现有基础设施 | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |

## 📋 部署前准备

### 系统要求
- Docker 20.0+
- Docker Compose 2.0+
- 2GB+ 可用内存 (快速部署)
- 8GB+ 可用内存 (生产级部署)

### 获取 GitHub Token
1. 访问 [GitHub Settings > Personal access tokens](https://github.com/settings/tokens)
2. 创建新的 Token，选择权限：`public_repo`
3. 复制 Token 备用 (格式: `ghp_xxxxxxxxxxxx`)

---

## 🚀 方式一：快速部署

**适合场景**：个人使用、快速测试、学习目的

**特点**：
- ✅ 零配置启动，一键部署
- ✅ 内置 WARP 代理，自动避免 IP 封禁
- ✅ SQLite 数据库，无需额外数据库服务
- ✅ Web 管理界面，实时监控
- ❌ 单实例，无高可用

### 部署步骤

```bash
# 1. 下载快速部署文件
cd deployments/quick

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，填入你的 GitHub Token

# 3. 一键部署
bash deploy.sh
```

### 访问服务
- **Web界面**: http://localhost:8080
- **健康检查**: http://localhost:8080/health

### 管理命令
```bash
# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 停止服务
docker-compose down
```

---

## 🏢 方式二：生产级部署

**适合场景**：企业环境、高负载、7x24运行

**特点**：
- ✅ 高可用集群，负载均衡
- ✅ PostgreSQL + Redis，企业级数据库
- ✅ 多WARP代理，提高稳定性
- ✅ Nginx 反向代理，SSL支持
- ✅ 完整监控体系 (Prometheus + Grafana)
- ✅ 自动扩缩容支持

### 部署步骤

```bash
# 1. 进入生产部署目录
cd deployments/production

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env，配置数据库密码、域名等

# 3. 配置SSL证书 (可选)
mkdir -p config/ssl
# 将SSL证书文件放入 config/ssl/ 目录

# 4. 启动服务
docker-compose up -d

# 5. 初始化数据库
docker-compose exec postgres psql -U kooix -d kooix_hajimi -f /docker-entrypoint-initdb.d/init.sql

# 6. 验证部署
bash scripts/health-check.sh
```

### 访问服务
- **主应用**: https://your-domain.com
- **监控面板**: https://your-domain.com/grafana
- **Prometheus**: https://your-domain.com/prometheus

### 监控告警
- 集成 Slack/邮件告警
- 实时性能指标监控
- 自动日志聚合和分析

---

## 🔧 方式三：本机服务部署

**适合场景**：已有基础设施、需要集成现有数据库

**特点**：
- ✅ 灵活配置，适配现有环境
- ✅ 支持外部 PostgreSQL/Redis
- ✅ 可选组件，按需启用
- ✅ 资源优化，降低运行成本

### 部署步骤

```bash
# 1. 进入本机部署目录
cd deployments/local

# 2. 配置外部服务连接
cp .env.example .env
# 配置外部数据库和Redis连接信息

# 3. 选择要启用的组件
# 启动基础应用
docker-compose up -d

# 启动本地WARP代理 (可选)
docker-compose --profile warp-local up -d

# 启动本地监控 (可选)
docker-compose --profile monitoring up -d
```

### 外部服务配置

#### PostgreSQL 数据库
```sql
-- 在你的 PostgreSQL 中执行
CREATE DATABASE kooix_hajimi;
CREATE USER kooix WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE kooix_hajimi TO kooix;
```

#### Redis 配置
```bash
# 确保 Redis 允许外部连接
# 在 redis.conf 中配置
bind 0.0.0.0
requirepass your_redis_password
```

---

## ⚙️ 高级配置

### 代理配置
```bash
# 单个代理
PROXY=socks5://proxy-host:1080

# 多个代理轮换
PROXY=socks5://proxy1:1080,socks5://proxy2:1080,http://proxy3:8080

# 带认证的代理
PROXY=socks5://user:pass@proxy:1080
```

### 性能优化
```bash
# 高性能配置 (适合大内存服务器)
SCANNER_WORKER_COUNT=50
RATE_LIMIT_REQUESTS_PER_MINUTE=300
VALIDATOR_WORKER_COUNT=20

# 低资源配置 (适合小型VPS)
SCANNER_WORKER_COUNT=3
RATE_LIMIT_REQUESTS_PER_MINUTE=30
VALIDATOR_WORKER_COUNT=2
```

### 同步外部服务
```bash
# Gemini Balancer 同步
GEMINI_BALANCER_SYNC_ENABLED=true
GEMINI_BALANCER_URL=https://your-balancer.com
GEMINI_BALANCER_AUTH=your_password

# GPT Load Balancer 同步
GPT_LOAD_SYNC_ENABLED=true
GPT_LOAD_URL=https://your-gpt-load.com
GPT_LOAD_AUTH=your_token
GPT_LOAD_GROUP_NAME=group1,group2,group3
```

---

## 🛠️ 运维管理

### 日志管理
```bash
# 查看实时日志
docker-compose logs -f kooix-hajimi

# 查看特定时间段日志
docker-compose logs --since="1h" kooix-hajimi

# 导出日志
docker-compose logs --no-color kooix-hajimi > app.log
```

### 数据备份
```bash
# 数据库备份 (PostgreSQL)
docker-compose exec postgres pg_dump -U kooix kooix_hajimi > backup.sql

# 数据目录备份
tar -czf data-backup-$(date +%Y%m%d).tar.gz data/
```

### 更新升级
```bash
# 拉取最新镜像
docker-compose pull

# 滚动更新 (生产环境)
docker-compose up -d --no-deps kooix-hajimi-1
sleep 30
docker-compose up -d --no-deps kooix-hajimi-2

# 快速重启 (开发环境)
docker-compose restart
```

### 健康检查
```bash
# 检查服务状态
curl -f http://localhost:8080/health

# 检查数据库连接
curl -f http://localhost:8080/api/health/database

# 检查代理连接
curl --socks5-hostname localhost:1080 https://httpbin.org/ip
```

---

## 🚨 故障排除

### 常见问题

**1. WARP 代理连接失败**
```bash
# 重启WARP服务
docker-compose restart warp

# 检查WARP状态
docker-compose exec warp curl --socks5-hostname 127.0.0.1:1080 https://cloudflare.com/cdn-cgi/trace
```

**2. 数据库连接失败**
```bash
# 检查数据库状态
docker-compose exec postgres pg_isready -U kooix

# 查看数据库日志
docker-compose logs postgres
```

**3. GitHub API 频率限制**
```bash
# 检查Token状态
curl -H "Authorization: token YOUR_TOKEN" https://api.github.com/rate_limit

# 轮换更多Token
# 在 .env 中添加更多 GITHUB_TOKENS
```

**4. 内存不足**
```bash
# 查看资源使用
docker stats

# 调整配置
SCANNER_WORKER_COUNT=3
VALIDATOR_WORKER_COUNT=2
```

### 监控告警

**生产环境监控指标**：
- API 请求成功率 > 95%
- 响应时间 < 2s
- 内存使用率 < 80%
- 磁盘使用率 < 90%
- WARP 代理可用性 > 99%

---

## 📊 性能基准

### 快速部署性能
- **扫描速度**: ~100 文件/分钟
- **内存使用**: 500MB - 1GB
- **适合规模**: 个人使用，日扫描 < 10K 文件

### 生产级部署性能
- **扫描速度**: ~2000 文件/分钟
- **内存使用**: 4GB - 8GB
- **适合规模**: 企业使用，日扫描 > 100K 文件
- **高可用**: 99.9% 可用性

### 资源规划建议
```
小型部署 (< 1万文件/天):
- CPU: 2核心
- 内存: 2GB
- 磁盘: 20GB

中型部署 (1-10万文件/天):
- CPU: 4核心
- 内存: 8GB
- 磁盘: 100GB

大型部署 (> 10万文件/天):
- CPU: 8核心+
- 内存: 16GB+
- 磁盘: 500GB+
```

---

## 🔒 安全最佳实践

1. **Token 安全**
   - 使用专用的 GitHub Token，权限最小化
   - 定期轮换 Token (建议每30天)
   - 不要在日志中记录 Token

2. **网络安全**
   - 生产环境启用 SSL/TLS
   - 使用防火墙限制访问
   - 定期更新系统和依赖

3. **数据安全**
   - 定期备份数据
   - 加密敏感配置文件
   - 限制数据库访问权限

4. **监控安全**
   - 启用访问日志
   - 设置异常告警
   - 定期安全审计

---

## 📞 技术支持

- **文档**: [项目Wiki](https://github.com/your-org/kooix-hajimi/wiki)
- **问题反馈**: [GitHub Issues](https://github.com/your-org/kooix-hajimi/issues)
- **讨论交流**: [GitHub Discussions](https://github.com/your-org/kooix-hajimi/discussions)

选择适合你的部署方式，享受 Kooix Hajimi 带来的高效 API 密钥发现体验！ 🎉