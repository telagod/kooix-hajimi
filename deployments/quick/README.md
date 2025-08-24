# Kooix Hajimi - 快速部署

一键启动 Kooix Hajimi，包含 WARP 代理支持，适合个人使用和快速测试。

## 特性

- 🚀 **零配置启动** - 一行命令完成部署
- 🌐 **WARP代理集成** - 自动配置 Cloudflare WARP，避免IP封禁
- 💾 **SQLite存储** - 无需额外数据库，数据持久化到本地文件
- 📱 **Web管理界面** - 实时监控扫描状态和结果
- 🔄 **自动重启** - 服务异常自动恢复

## 系统要求

- Docker 20.0+
- Docker Compose 2.0+
- 2GB+ 可用内存
- 10GB+ 可用磁盘空间

## 快速启动

### 1. 下载部署文件

```bash
# 克隆项目
git clone https://github.com/your-org/kooix-hajimi.git
cd kooix-hajimi/deployments/quick

# 或直接下载部署文件
wget -O deploy.sh https://raw.githubusercontent.com/your-org/kooix-hajimi/main/deployments/quick/deploy.sh
chmod +x deploy.sh
```

### 2. 配置GitHub Token

```bash
# 复制配置文件
cp .env.example .env

# 编辑配置文件
nano .env
```

在 `.env` 文件中配置你的 GitHub Token：

```bash
GITHUB_TOKENS=ghp_your_actual_token_here
```

> 💡 **获取GitHub Token**: 访问 [GitHub Settings > Tokens](https://github.com/settings/tokens)，创建具有 `public_repo` 权限的访问令牌

### 3. 一键部署

```bash
# 执行部署脚本
bash deploy.sh
```

部署脚本会自动：
- ✅ 检查系统依赖
- ✅ 创建必要目录
- ✅ 验证配置文件
- ✅ 构建应用镜像
- ✅ 启动所有服务
- ✅ 检查服务状态

## 访问服务

部署完成后，可通过以下方式访问：

- **Web管理界面**: http://localhost:8080
- **健康检查**: http://localhost:8080/health
- **API文档**: http://localhost:8080/swagger

## 服务管理

### 查看服务状态
```bash
docker-compose ps
```

### 查看日志
```bash
# 查看所有日志
docker-compose logs -f

# 查看应用日志
docker-compose logs -f kooix-hajimi

# 查看WARP代理日志
docker-compose logs -f warp
```

### 重启服务
```bash
# 重启所有服务
docker-compose restart

# 重启特定服务
docker-compose restart kooix-hajimi
docker-compose restart warp
```

### 停止服务
```bash
docker-compose down
```

### 更新服务
```bash
# 拉取最新镜像并重启
docker-compose pull
docker-compose up -d
```

## 数据目录

```
data/
├── app/
│   ├── keys/           # 发现的API密钥文件
│   ├── logs/           # 详细运行日志
│   └── hajimi.db       # SQLite数据库
├── warp/               # WARP代理配置
└── config/
    └── queries.txt     # 搜索查询配置
```

## 自定义配置

### 修改搜索查询

编辑 `config/queries.txt` 文件：

```bash
# 添加自定义搜索表达式
AIzaSy in:file language:python
AIzaSy in:file filename:config.json
```

### 调整运行参数

编辑 `.env` 文件：

```bash
# 扫描间隔（支持: 30m, 1h, 2h, 24h）
SCANNER_SCAN_INTERVAL=2h

# 并发工作线程数
SCANNER_WORKER_COUNT=3

# 日志级别
LOG_LEVEL=debug
```

## 故障排除

### WARP代理连接失败
```bash
# 检查WARP服务状态
docker-compose logs warp

# 重启WARP服务
docker-compose restart warp

# 测试WARP连接
docker-compose exec warp curl --socks5-hostname 127.0.0.1:1080 https://cloudflare.com/cdn-cgi/trace
```

### 应用启动失败
```bash
# 检查配置文件
cat .env

# 查看详细错误日志
docker-compose logs kooix-hajimi

# 重新构建镜像
docker-compose build --no-cache
```

### 磁盘空间不足
```bash
# 清理Docker缓存
docker system prune -a

# 查看数据目录大小
du -sh data/
```

## 性能调优

### 资源限制

编辑 `docker-compose.yml` 添加资源限制：

```yaml
services:
  kooix-hajimi:
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '1.0'
```

### 并发优化

根据机器配置调整 `.env` 中的参数：

```bash
# 高性能配置
SCANNER_WORKER_COUNT=10
RATE_LIMIT_REQUESTS_PER_MINUTE=60

# 低资源配置
SCANNER_WORKER_COUNT=2
RATE_LIMIT_REQUESTS_PER_MINUTE=20
```

## 安全建议

- 🔐 定期轮换 GitHub Token
- 🚫 不要将 `.env` 文件提交到版本控制
- 🔒 限制数据目录访问权限
- 📊 定期清理过期的密钥文件

## 升级指南

### 从旧版本升级
```bash
# 停止服务
docker-compose down

# 备份数据
cp -r data data.backup

# 拉取新版本
git pull

# 重新部署
bash deploy.sh
```

## 技术支持

- 📚 **文档**: [项目Wiki](https://github.com/your-org/kooix-hajimi/wiki)
- 🐛 **问题反馈**: [GitHub Issues](https://github.com/your-org/kooix-hajimi/issues)
- 💬 **社区讨论**: [GitHub Discussions](https://github.com/your-org/kooix-hajimi/discussions)