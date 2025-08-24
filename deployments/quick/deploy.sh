#!/bin/bash

# Kooix Hajimi 快速部署脚本
# 适用于单机快速启动，包含WARP代理支持

set -e

echo "🎪 Kooix Hajimi - 快速部署脚本"
echo "================================"

# 检查依赖
check_dependencies() {
    echo "📋 检查系统依赖..."
    
    if ! command -v docker &> /dev/null; then
        echo "❌ Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        echo "❌ Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    echo "✅ 系统依赖检查通过"
}

# 创建必要目录
create_directories() {
    echo "📁 创建数据目录..."
    mkdir -p data/app/{keys,logs}
    mkdir -p data/warp
    mkdir -p config
    echo "✅ 目录创建完成"
}

# 配置检查
check_config() {
    echo "🔧 检查配置文件..."
    
    if [ ! -f ".env" ]; then
        echo "📝 创建环境配置文件..."
        cp .env.example .env
        echo "⚠️  请编辑 .env 文件，填入你的 GitHub Token"
        echo "   配置路径: $(pwd)/.env"
        echo "   获取Token: https://github.com/settings/tokens"
        echo ""
        read -p "是否现在编辑配置文件? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            ${EDITOR:-nano} .env
        else
            echo "⚠️  请手动编辑 .env 文件后重新运行此脚本"
            exit 1
        fi
    fi
    
    # 检查GitHub Token是否已配置
    source .env
    if [ -z "$GITHUB_TOKENS" ] || [ "$GITHUB_TOKENS" = "ghp_your_token_1,ghp_your_token_2" ]; then
        echo "❌ GitHub Token 未配置或使用默认值"
        echo "   请在 .env 文件中配置 GITHUB_TOKENS"
        exit 1
    fi
    
    echo "✅ 配置检查通过"
}

# 构建镜像
build_image() {
    echo "🔨 构建应用镜像..."
    cd ../..
    docker build -t kooix-hajimi:latest .
    cd deployments/quick
    echo "✅ 镜像构建完成"
}

# 启动服务
start_services() {
    echo "🚀 启动服务..."
    docker-compose up -d
    echo "✅ 服务启动完成"
}

# 检查服务状态
check_services() {
    echo "🔍 检查服务状态..."
    sleep 10
    
    echo ""
    echo "📊 服务状态:"
    docker-compose ps
    
    echo ""
    echo "🌐 WARP代理测试:"
    if docker-compose exec -T warp curl --socks5-hostname 127.0.0.1:1080 https://cloudflare.com/cdn-cgi/trace 2>/dev/null | grep -q "warp=on"; then
        echo "✅ WARP代理工作正常"
    else
        echo "⚠️  WARP代理可能未就绪，等待几分钟后重试"
    fi
    
    echo ""
    echo "📱 Web界面:"
    echo "   本地访问: http://localhost:8080"
    echo "   健康检查: http://localhost:8080/health"
    
    echo ""
    echo "📋 查看日志:"
    echo "   应用日志: docker-compose logs -f kooix-hajimi"
    echo "   WARP日志: docker-compose logs -f warp"
    echo "   所有日志: docker-compose logs -f"
}

# 显示使用说明
show_usage() {
    echo ""
    echo "🎯 快速部署完成！"
    echo "=================="
    echo ""
    echo "📁 目录结构:"
    echo "   data/app/keys/     - 发现的API密钥"
    echo "   data/app/logs/     - 详细日志"
    echo "   data/warp/         - WARP代理数据"
    echo "   config/queries.txt - 搜索查询配置"
    echo ""
    echo "🛠️  常用命令:"
    echo "   查看状态: docker-compose ps"
    echo "   查看日志: docker-compose logs -f"
    echo "   重启服务: docker-compose restart"
    echo "   停止服务: docker-compose down"
    echo "   更新镜像: docker-compose pull && docker-compose up -d"
    echo ""
    echo "📚 更多配置:"
    echo "   修改查询: 编辑 config/queries.txt"
    echo "   调整设置: 编辑 .env 文件"
    echo ""
    echo "💡 提示:"
    echo "   - 首次启动需要等待WARP代理初始化（约2-5分钟）"
    echo "   - 扫描频率可在.env中调整SCANNER_SCAN_INTERVAL"
    echo "   - Web界面提供实时监控和统计信息"
}

# 主函数
main() {
    check_dependencies
    create_directories
    check_config
    build_image
    start_services
    check_services
    show_usage
}

# 错误处理
trap 'echo "❌ 部署失败，请检查错误信息"; exit 1' ERR

# 运行主函数
main