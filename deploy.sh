#!/bin/bash

# Kooix Hajimi 一键部署脚本
# 支持三种部署模式的自动化部署

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
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

# 显示横幅
show_banner() {
    echo -e "${BLUE}"
    echo "🎪======================================🎪"
    echo "    Kooix Hajimi - 一键部署脚本"
    echo "         GitHub API Key Discovery"
    echo "🎪======================================🎪"
    echo -e "${NC}"
}

# 显示部署模式菜单
show_deployment_menu() {
    echo ""
    echo "请选择部署模式:"
    echo ""
    echo -e "${GREEN}1. 快速部署${NC} - 适合个人使用，零配置启动"
    echo "   ✅ SQLite数据库 + 内置WARP代理"
    echo "   ✅ 单实例，资源占用低"
    echo "   ✅ 一键启动，5分钟部署完成"
    echo ""
    echo -e "${BLUE}2. 生产级部署${NC} - 适合企业环境，高可用集群"
    echo "   ✅ PostgreSQL + Redis + 负载均衡"
    echo "   ✅ 多实例高可用 + 完整监控"
    echo "   ✅ SSL支持 + 自动扩缩容"
    echo ""
    echo -e "${YELLOW}3. 本机服务部署${NC} - 适合现有基础设施集成"
    echo "   ✅ 连接外部数据库和服务"
    echo "   ✅ 灵活配置，按需启用组件"
    echo "   ✅ 资源优化，成本可控"
    echo ""
    echo -e "${RED}4. 退出${NC}"
    echo ""
}

# 检查系统依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        echo "安装指令: curl -fsSL https://get.docker.com | bash"
        exit 1
    fi
    
    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        echo "安装指令: pip install docker-compose 或 下载二进制文件"
        exit 1
    fi
    
    # 检查 Docker 服务状态
    if ! docker info &> /dev/null; then
        log_error "Docker 服务未运行，请启动 Docker 服务"
        echo "启动指令: sudo systemctl start docker"
        exit 1
    fi
    
    log_success "系统依赖检查通过"
}

# 检查GitHub Token
check_github_token() {
    local env_file=$1
    
    if [ ! -f "$env_file" ]; then
        log_warning ".env 文件不存在，将从示例创建"
        return 1
    fi
    
    source "$env_file"
    if [ -z "$GITHUB_TOKENS" ] || [ "$GITHUB_TOKENS" = "ghp_your_token_1,ghp_your_token_2" ]; then
        log_warning "GitHub Token 未配置或使用默认值"
        return 1
    fi
    
    log_success "GitHub Token 配置检查通过"
    return 0
}

# 配置GitHub Token
configure_github_token() {
    local env_file=$1
    
    echo ""
    log_info "配置 GitHub Token"
    echo "请访问 https://github.com/settings/tokens 创建新的 Token"
    echo "权限选择: public_repo (读取公共仓库)"
    echo ""
    
    read -p "请输入你的 GitHub Token (ghp_开头): " github_token
    
    if [[ ! $github_token =~ ^ghp_ ]]; then
        log_error "GitHub Token 格式错误，应该以 ghp_ 开头"
        return 1
    fi
    
    # 验证Token有效性
    log_info "验证 Token 有效性..."
    if curl -s -H "Authorization: token $github_token" https://api.github.com/user | grep -q "login"; then
        log_success "Token 验证成功"
    else
        log_error "Token 验证失败，请检查Token是否正确"
        return 1
    fi
    
    # 更新环境变量文件
    if [ -f "$env_file" ]; then
        sed -i "s/GITHUB_TOKENS=.*/GITHUB_TOKENS=$github_token/" "$env_file"
    else
        echo "GITHUB_TOKENS=$github_token" > "$env_file"
    fi
    
    log_success "GitHub Token 配置完成"
}

# 快速部署
deploy_quick() {
    log_info "开始快速部署..."
    
    cd deployments/quick
    
    # 检查配置
    if ! check_github_token ".env"; then
        cp .env.example .env
        if ! configure_github_token ".env"; then
            log_error "GitHub Token 配置失败"
            return 1
        fi
    fi
    
    # 创建数据目录
    mkdir -p data/{app,warp} config
    
    # 确保查询文件存在
    if [ ! -f "config/queries.txt" ]; then
        log_info "创建默认查询文件..."
        cp config/queries.txt queries.txt 2>/dev/null || {
            echo "AIzaSy in:file" > queries.txt
        }
    fi
    
    # 构建镜像
    log_info "构建应用镜像..."
    cd ../..
    docker build -t kooix-hajimi:latest .
    cd deployments/quick
    
    # 启动服务
    log_info "启动服务..."
    docker-compose up -d
    
    # 等待服务启动
    log_info "等待服务启动..."
    sleep 30
    
    # 检查服务状态
    if curl -f http://localhost:8080/health &> /dev/null; then
        log_success "快速部署完成！"
        echo ""
        echo "🎉 服务访问地址:"
        echo "   Web界面: http://localhost:8080"
        echo "   健康检查: http://localhost:8080/health"
        echo ""
        echo "📊 管理命令:"
        echo "   查看状态: docker-compose ps"
        echo "   查看日志: docker-compose logs -f"
        echo "   重启服务: docker-compose restart"
        echo "   停止服务: docker-compose down"
    else
        log_error "服务启动失败，请检查日志"
        docker-compose logs
        return 1
    fi
}

# 生产级部署
deploy_production() {
    log_info "开始生产级部署..."
    
    cd deployments/production
    
    # 检查配置文件
    if [ ! -f ".env" ]; then
        log_info "创建生产环境配置文件..."
        cp .env.example .env
        log_warning "请编辑 .env 文件，配置数据库密码、域名等信息"
        echo "配置文件位置: $(pwd)/.env"
        read -p "配置完成后按回车继续..."
    fi
    
    # 生成随机密码
    generate_password() {
        openssl rand -base64 32 | tr -d "=+/" | cut -c1-25
    }
    
    source .env
    if [ -z "$POSTGRES_PASSWORD" ] || [ "$POSTGRES_PASSWORD" = "your_secure_postgres_password_here" ]; then
        log_info "生成随机数据库密码..."
        postgres_password=$(generate_password)
        sed -i "s/POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$postgres_password/" .env
        log_success "数据库密码已生成: $postgres_password"
    fi
    
    # 检查GitHub Token
    if ! check_github_token ".env"; then
        if ! configure_github_token ".env"; then
            log_error "GitHub Token 配置失败"
            return 1
        fi
    fi
    
    # 创建SSL目录
    mkdir -p config/ssl config/monitoring config/postgres
    
    # 生成自签名证书 (如果不存在)
    if [ ! -f "config/ssl/fullchain.pem" ]; then
        log_info "生成自签名SSL证书..."
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout config/ssl/privkey.pem \
            -out config/ssl/fullchain.pem \
            -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
        log_success "SSL证书生成完成"
    fi
    
    # 构建镜像
    log_info "构建应用镜像..."
    cd ../..
    docker build -t kooix-hajimi:latest .
    cd deployments/production
    
    # 启动服务
    log_info "启动生产级服务..."
    docker-compose up -d
    
    # 等待数据库初始化
    log_info "等待数据库初始化..."
    sleep 60
    
    # 检查服务状态
    log_info "检查服务状态..."
    if docker-compose ps | grep -q "Up"; then
        log_success "生产级部署完成！"
        echo ""
        echo "🎉 服务访问地址:"
        echo "   主应用: https://localhost (如果配置了域名)"
        echo "   Grafana监控: https://localhost/grafana"
        echo "   Prometheus: https://localhost/prometheus"
        echo ""
        echo "📊 管理命令:"
        echo "   查看状态: docker-compose ps"
        echo "   查看日志: docker-compose logs -f"
        echo "   扩容应用: docker-compose scale kooix-hajimi-1=2"
        echo "   停止服务: docker-compose down"
        echo ""
        echo "🔒 默认凭据:"
        echo "   数据库: user=kooix, password=$(grep POSTGRES_PASSWORD .env | cut -d= -f2)"
        echo "   Grafana: admin/admin123 (首次登录需修改)"
    else
        log_error "服务启动失败，请检查日志"
        docker-compose logs
        return 1
    fi
}

# 本机服务部署
deploy_local() {
    log_info "开始本机服务部署..."
    
    cd deployments/local
    
    # 检查配置
    if [ ! -f ".env" ]; then
        cp .env.example .env
        log_warning "请编辑 .env 文件，配置外部数据库连接信息"
        echo "配置文件位置: $(pwd)/.env"
        read -p "配置完成后按回车继续..."
    fi
    
    # 检查GitHub Token
    if ! check_github_token ".env"; then
        if ! configure_github_token ".env"; then
            log_error "GitHub Token 配置失败"
            return 1
        fi
    fi
    
    # 询问要启用的组件
    echo ""
    echo "选择要启用的可选组件:"
    read -p "启用本地WARP代理? (y/n): " -n 1 -r enable_warp
    echo
    read -p "启用本地监控? (y/n): " -n 1 -r enable_monitoring
    echo
    
    # 构建镜像
    log_info "构建应用镜像..."
    cd ../..
    docker build -t kooix-hajimi:latest .
    cd deployments/local
    
    # 启动基础服务
    log_info "启动基础服务..."
    docker-compose up -d
    
    # 启动可选组件
    if [[ $enable_warp =~ ^[Yy]$ ]]; then
        log_info "启动本地WARP代理..."
        docker-compose --profile warp-local up -d
    fi
    
    if [[ $enable_monitoring =~ ^[Yy]$ ]]; then
        log_info "启动本地监控..."
        docker-compose --profile monitoring up -d
    fi
    
    # 等待服务启动
    sleep 20
    
    # 检查服务状态
    if curl -f http://localhost:8080/health &> /dev/null; then
        log_success "本机服务部署完成！"
        echo ""
        echo "🎉 服务访问地址:"
        echo "   Web界面: http://localhost:8080"
        if [[ $enable_monitoring =~ ^[Yy]$ ]]; then
            echo "   Grafana: http://localhost:3000 (admin/admin123)"
            echo "   Prometheus: http://localhost:9091"
        fi
        echo ""
        echo "📊 管理命令:"
        echo "   查看状态: docker-compose ps"
        echo "   查看完整状态: docker-compose --profile warp-local --profile monitoring ps"
        echo "   查看日志: docker-compose logs -f"
        echo "   停止服务: docker-compose down"
    else
        log_error "服务启动失败，请检查日志和外部服务连接"
        docker-compose logs
        return 1
    fi
}

# 显示后续操作指南
show_next_steps() {
    echo ""
    log_info "部署完成后的操作建议:"
    echo ""
    echo "🔧 配置优化:"
    echo "   - 编辑 queries.txt 文件，自定义搜索表达式"
    echo "   - 调整 .env 中的扫描参数和速率限制"
    echo "   - 配置外部同步服务 (可选)"
    echo ""
    echo "📊 监控建议:"
    echo "   - 定期查看 Web 界面的统计信息"
    echo "   - 监控日志文件，关注错误和警告"
    echo "   - 设置磁盘空间监控，防止存储满"
    echo ""
    echo "🔒 安全建议:"
    echo "   - 定期轮换 GitHub Token (建议30天)"
    echo "   - 备份重要数据和配置文件"
    echo "   - 启用防火墙，限制不必要的端口访问"
    echo ""
}

# 主函数
main() {
    show_banner
    check_dependencies
    
    while true; do
        show_deployment_menu
        read -p "请选择部署模式 (1-4): " choice
        
        case $choice in
            1)
                log_info "选择了快速部署模式"
                if deploy_quick; then
                    show_next_steps
                    break
                fi
                ;;
            2)
                log_info "选择了生产级部署模式"
                if deploy_production; then
                    show_next_steps
                    break
                fi
                ;;
            3)
                log_info "选择了本机服务部署模式"
                if deploy_local; then
                    show_next_steps
                    break
                fi
                ;;
            4)
                log_info "退出部署脚本"
                exit 0
                ;;
            *)
                log_error "无效选择，请输入 1-4"
                ;;
        esac
        
        echo ""
        read -p "部署失败，是否重试? (y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "退出部署脚本"
            exit 1
        fi
    done
}

# 错误处理
trap 'echo -e "\n${RED}[ERROR]${NC} 部署过程中发生错误，请检查上面的错误信息"; exit 1' ERR

# 运行主函数
main "$@"