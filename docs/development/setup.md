# 开发环境搭建

## 系统要求

- Go 1.21+
- Git
- Node.js 16+ (可选，用于前端开发)
- SQLite3 或 PostgreSQL
- Docker (可选)

## 环境搭建

### 1. 克隆仓库
```bash
git clone https://github.com/telagod/kooix-hajimi.git
cd kooix-hajimi
```

### 2. 安装Go依赖
```bash
go mod tidy
```

### 3. 设置环境变量
```bash
# 复制环境变量模板
cp .env.example .env

# 编辑环境变量
export HAJIMI_GITHUB_TOKENS="your_token_1,your_token_2"
export HAJIMI_LOG_LEVEL="debug"
export HAJIMI_STORAGE_TYPE="sqlite"
```

### 4. 构建项目
```bash
# 使用构建脚本
./scripts/build.sh all

# 或手动构建
make build-server
make build-cli
```

### 5. 运行开发服务器
```bash
# 直接运行
go run cmd/server/main.go

# 或使用构建后的二进制
./build/hajimi-server
```

## 开发工具

### 推荐IDE配置

**VS Code配置** (`.vscode/settings.json`):
```json
{
  "go.lintTool": "golangci-lint",
  "go.formatTool": "goimports",
  "go.useLanguageServer": true,
  "go.buildOnSave": "package",
  "go.testOnSave": false
}
```

**GoLand配置**:
- 启用gofmt on save
- 配置golangci-lint
- 设置代码模板

### 代码质量工具

```bash
# 安装代码检查工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 运行代码检查
golangci-lint run

# 格式化代码
gofmt -s -w .
goimports -w .
```

## 调试配置

### VS Code调试配置 (`.vscode/launch.json`)
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Server",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/server/main.go",
      "env": {
        "HAJIMI_GITHUB_TOKENS": "your_tokens",
        "HAJIMI_LOG_LEVEL": "debug"
      }
    },
    {
      "name": "Debug CLI",
      "type": "go", 
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/cli/main.go",
      "args": ["scan", "--query", "AIzaSy"]
    }
  ]
}
```

### Delve调试器
```bash
# 安装delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试服务器
dlv debug cmd/server/main.go

# 调试CLI
dlv debug cmd/cli/main.go -- scan --query "AIzaSy"
```

## 测试

### 运行测试
```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/scanner/

# 运行测试并生成覆盖率报告
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 基准测试
```bash
# 运行基准测试
go test -bench=. ./internal/scanner/

# 内存分析
go test -bench=. -memprofile=mem.prof ./internal/scanner/
go tool pprof mem.prof
```

### 集成测试
```bash
# 运行集成测试
go test -tags=integration ./tests/

# Docker集成测试
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

## 前端开发

### 设置前端开发环境
```bash
# 安装Node.js依赖（如果有）
npm install

# 监控静态文件变化
make watch-static

# 或手动启动文件监控
fswatch -o web/static/ | xargs -n1 -I{} make build-static
```

### 前端资源
- **JavaScript**: `/web/static/js/`
- **CSS**: `/web/static/css/`
- **模板**: `/web/templates/`
- **图片**: `/web/static/images/`

## 数据库开发

### SQLite开发
```bash
# 查看数据库结构
sqlite3 data/hajimi-king.db ".schema"

# 查看表数据
sqlite3 data/hajimi-king.db "SELECT * FROM valid_keys LIMIT 5;"
```

### PostgreSQL开发
```bash
# 启动开发数据库
docker-compose -f docker-compose.dev.yml up -d postgres

# 连接数据库
psql postgres://hajimi:password@localhost:5432/hajimi_dev

# 运行迁移
make migrate-up
```

## 开发工作流

### Git工作流
```bash
# 创建功能分支
git checkout -b feature/new-feature

# 提交更改
git add .
git commit -m "feat: add new feature"

# 推送分支
git push origin feature/new-feature
```

### 代码提交规范
使用 Conventional Commits 格式：
- `feat:` 新功能
- `fix:` Bug修复
- `docs:` 文档更新
- `style:` 代码格式
- `refactor:` 重构
- `test:` 测试相关
- `chore:` 构建/工具相关

### 预提交检查
```bash
# 安装pre-commit hooks
make install-hooks

# 手动运行检查
make pre-commit
```

## 性能分析

### CPU性能分析
```bash
# 启动CPU分析
go run -cpuprofile=cpu.prof cmd/server/main.go

# 分析CPU使用
go tool pprof cpu.prof
```

### 内存分析
```bash
# 启动内存分析
go run -memprofile=mem.prof cmd/server/main.go

# 分析内存使用
go tool pprof mem.prof
```

### 实时监控
```bash
# 安装监控工具
go install github.com/google/gops@latest

# 监控运行中的进程
gops
```

## 常见开发问题

### 构建问题
1. **Go版本不兼容**: 确保使用Go 1.21+
2. **依赖下载失败**: 设置GOPROXY或使用VPN
3. **CGO编译错误**: 安装对应平台的编译工具

### 运行时问题
1. **端口被占用**: 修改配置文件中的端口设置
2. **数据库连接失败**: 检查数据库配置和网络连接
3. **GitHub API限制**: 验证token有效性和权限

### 调试技巧
1. **启用详细日志**: 设置`LOG_LEVEL=debug`
2. **使用调试器**: 在关键位置设置断点
3. **添加打印语句**: 临时添加fmt.Printf调试
4. **检查goroutine**: 使用`go tool trace`分析并发

## 开发最佳实践

### 代码组织
- 遵循Go项目布局标准
- 使用接口进行抽象
- 保持函数简洁和单一职责
- 添加适当的注释和文档

### 错误处理
- 使用标准的error类型
- 包装错误信息提供上下文
- 合理使用panic和recover
- 记录错误日志

### 并发编程
- 使用channel进行通信
- 避免共享内存
- 正确使用sync包
- 注意goroutine泄漏

### 测试编写
- 单元测试覆盖主要逻辑
- 使用mock进行隔离测试
- 编写集成测试验证整体流程
- 性能测试确保性能要求