#!/bin/bash

# Hajimi King Go - 构建和部署脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目信息
PROJECT_NAME="kooix-hajimi"
VERSION=${VERSION:-"latest"}
REGISTRY=${REGISTRY:-""}

# 函数定义
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    log_success "Dependencies check passed"
}

# 运行测试
run_tests() {
    log_info "Running tests..."
    
    # 单元测试
    go test -v ./...
    
    # 集成测试（如果存在）
    if [ -d "tests" ]; then
        go test -v ./tests/...
    fi
    
    log_success "Tests passed"
}

# 构建二进制文件
build_binary() {
    log_info "Building binary files..."
    
    # 清理之前的构建
    rm -rf build/
    mkdir -p build/
    
    # 构建服务器
    log_info "Building server..."
    CGO_ENABLED=1 go build -ldflags "-w -s" -o build/hajimi-king-server cmd/server/main.go
    
    # 构建CLI
    log_info "Building CLI..."
    CGO_ENABLED=1 go build -ldflags "-w -s" -o build/hajimi-king-cli cmd/cli/main.go
    
    # 复制配置文件
    cp -r configs build/
    cp -r web build/
    
    log_success "Binary build completed"
}

# 构建Docker镜像
build_docker() {
    log_info "Building Docker image..."
    
    local image_name="$PROJECT_NAME:$VERSION"
    
    if [ -n "$REGISTRY" ]; then
        image_name="$REGISTRY/$image_name"
    fi
    
    docker build -t "$image_name" .
    
    log_success "Docker image built: $image_name"
}

# 推送Docker镜像
push_docker() {
    if [ -z "$REGISTRY" ]; then
        log_warning "No registry specified, skipping push"
        return
    fi
    
    log_info "Pushing Docker image to registry..."
    
    local image_name="$REGISTRY/$PROJECT_NAME:$VERSION"
    docker push "$image_name"
    
    log_success "Docker image pushed: $image_name"
}

# 部署到本地
deploy_local() {
    log_info "Deploying to local environment..."
    
    # 检查环境变量
    if [ -z "$GITHUB_TOKENS" ]; then
        log_error "GITHUB_TOKENS environment variable is required"
        exit 1
    fi
    
    # 创建必要的目录
    mkdir -p data
    
    # 复制示例配置文件
    if [ ! -f "queries.txt" ]; then
        if [ -f "queries.example" ]; then
            cp queries.example queries.txt
            log_info "Created queries.txt from example"
        else
            echo "AIzaSy in:file" > queries.txt
            log_info "Created default queries.txt"
        fi
    fi
    
    # 启动服务
    docker-compose up -d
    
    log_success "Local deployment completed"
    log_info "Web interface available at: http://localhost:8080"
}

# 停止服务
stop_services() {
    log_info "Stopping services..."
    
    docker-compose down
    
    log_success "Services stopped"
}

# 清理
clean() {
    log_info "Cleaning up..."
    
    # 清理构建文件
    rm -rf build/
    
    # 清理Docker镜像（可选）
    if [ "$1" = "--docker" ]; then
        docker rmi $(docker images "$PROJECT_NAME" -q) 2>/dev/null || true
        log_info "Docker images cleaned"
    fi
    
    log_success "Cleanup completed"
}

# 显示帮助
show_help() {
    echo "Hajimi King Go - Build and Deploy Script"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  check       - Check dependencies"
    echo "  test        - Run tests"
    echo "  build       - Build binary files"
    echo "  docker      - Build Docker image"
    echo "  push        - Push Docker image to registry"
    echo "  deploy      - Deploy to local environment"
    echo "  stop        - Stop running services"
    echo "  clean       - Clean build artifacts"
    echo "  all         - Run check, test, build, docker"
    echo "  help        - Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  VERSION     - Version tag (default: latest)"
    echo "  REGISTRY    - Docker registry URL"
    echo "  GITHUB_TOKENS - GitHub API tokens (required for deployment)"
    echo ""
    echo "Examples:"
    echo "  $0 all                          # Build everything"
    echo "  $0 deploy                       # Deploy locally"
    echo "  VERSION=v2.0.0 $0 docker        # Build with version tag"
    echo "  REGISTRY=myregistry.com $0 push # Push to registry"
}

# 主逻辑
case "${1:-help}" in
    check)
        check_dependencies
        ;;
    test)
        check_dependencies
        run_tests
        ;;
    build)
        check_dependencies
        build_binary
        ;;
    docker)
        check_dependencies
        build_docker
        ;;
    push)
        push_docker
        ;;
    deploy)
        check_dependencies
        build_docker
        deploy_local
        ;;
    stop)
        stop_services
        ;;
    clean)
        clean $2
        ;;
    all)
        check_dependencies
        run_tests
        build_binary
        build_docker
        ;;
    help)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac