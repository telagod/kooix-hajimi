# 安装指南

## 系统要求

- Go 1.21+
- Git
- SQLite3 或 PostgreSQL (可选)
- Docker (可选)

## 方式一：从源码安装

### 1. 克隆项目
```bash
git clone https://github.com/telagod/kooix-hajimi.git
cd kooix-hajimi
```

### 2. 安装依赖
```bash
go mod tidy
```

### 3. 构建项目
```bash
# 使用构建脚本（推荐）
./scripts/build.sh all

# 或手动构建
go build -o build/hajimi-server cmd/server/main.go
go build -o build/hajimi-cli cmd/cli/main.go
```

### 4. 配置环境
```bash
# 复制配置文件
cp configs/config.yaml.example configs/config.yaml

# 设置GitHub Token
export HAJIMI_GITHUB_TOKENS="your_token_1,your_token_2"
```

### 5. 运行服务
```bash
./build/hajimi-server
```

## 方式二：Docker安装

### 使用预构建镜像
```bash
# 从GitHub Container Registry拉取
docker pull ghcr.io/telagod/kooix-hajimi:latest

# 运行容器
docker run -d \
  --name kooix-hajimi \
  -p 8080:8080 \
  -e HAJIMI_GITHUB_TOKENS="your_tokens" \
  -v ./data:/app/data \
  ghcr.io/telagod/kooix-hajimi:latest
```

### 使用Docker Compose
```bash
# 设置环境变量
export GITHUB_TOKENS="your_token_1,your_token_2"

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

## 验证安装

1. 打开浏览器访问: http://localhost:8080
2. 检查仪表板是否正常显示
3. 在设置页面配置GitHub tokens
4. 运行一次测试扫描

## 下一步

- [配置指南](../setup/configuration.md)
- [GitHub权限设置](../security/github-permissions.md)
- [API文档](../api/README.md)