# 部署指南

## 部署方式概览

Kooix Hajimi 支持多种部署方式，根据你的环境和需求选择合适的部署策略。

## Docker部署（推荐）

### 单容器部署

**步骤1: 拉取镜像**
```bash
docker pull ghcr.io/telagod/kooix-hajimi:latest
```

**步骤2: 准备环境变量**
```bash
# 创建环境变量文件
cat > .env << EOF
HAJIMI_GITHUB_TOKENS=token1,token2,token3
HAJIMI_LOG_LEVEL=info
HAJIMI_WEB_PORT=8080
EOF
```

**步骤3: 运行容器**
```bash
docker run -d \
  --name kooix-hajimi \
  --env-file .env \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  ghcr.io/telagod/kooix-hajimi:latest
```

### Docker Compose部署

**基础配置 (SQLite)**
```yaml
# docker-compose.yml
version: '3.8'

services:
  kooix-hajimi:
    image: ghcr.io/telagod/kooix-hajimi:latest
    container_name: kooix-hajimi
    ports:
      - "8080:8080"
    environment:
      - HAJIMI_GITHUB_TOKENS=${GITHUB_TOKENS}
      - HAJIMI_LOG_LEVEL=info
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
      - ./configs:/app/configs
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/status"]
      interval: 30s
      timeout: 10s
      retries: 3
```

**生产配置 (PostgreSQL)**
```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  kooix-hajimi:
    image: ghcr.io/telagod/kooix-hajimi:latest
    container_name: kooix-hajimi
    ports:
      - "8080:8080"
    environment:
      - HAJIMI_GITHUB_TOKENS=${GITHUB_TOKENS}
      - HAJIMI_STORAGE_TYPE=postgres
      - HAJIMI_STORAGE_DSN=postgres://hajimi:${DB_PASSWORD}@postgres:5432/hajimi_db
      - HAJIMI_LOG_LEVEL=info
    volumes:
      - ./logs:/app/logs
      - ./configs:/app/configs
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/status"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15-alpine
    container_name: kooix-hajimi-db
    environment:
      - POSTGRES_DB=hajimi_db
      - POSTGRES_USER=hajimi
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./deployments/production/config/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U hajimi"]
      interval: 10s
      timeout: 5s
      retries: 5

  nginx:
    image: nginx:alpine
    container_name: kooix-hajimi-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./deployments/production/config/nginx.conf:/etc/nginx/nginx.conf
      - ./certs:/etc/nginx/certs
    restart: unless-stopped
    depends_on:
      - kooix-hajimi

volumes:
  postgres_data:
```

## 生产环境部署

### 高可用架构

```yaml
# docker-compose.ha.yml
version: '3.8'

services:
  kooix-hajimi-1:
    image: ghcr.io/telagod/kooix-hajimi:latest
    environment:
      - HAJIMI_WEB_PORT=8080
      - HAJIMI_STORAGE_DSN=postgres://hajimi:${DB_PASSWORD}@postgres:5432/hajimi_db
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 1G
          cpus: '1.0'

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=hajimi_db
      - POSTGRES_USER=hajimi
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    deploy:
      replicas: 1
      placement:
        constraints:
          - node.role == manager

  redis:
    image: redis:alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    deploy:
      replicas: 1

volumes:
  postgres_data:
  redis_data:
```

### 监控集成

```yaml
# docker-compose.monitoring.yml
version: '3.8'

services:
  kooix-hajimi:
    # ... 主服务配置
    labels:
      - "prometheus.scrape=true"
      - "prometheus.port=8080"
      - "prometheus.path=/metrics"

  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/var/lib/grafana/dashboards

volumes:
  prometheus_data:
  grafana_data:
```

## Kubernetes部署

### 基础Deployment
```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kooix-hajimi
  labels:
    app: kooix-hajimi
spec:
  replicas: 2
  selector:
    matchLabels:
      app: kooix-hajimi
  template:
    metadata:
      labels:
        app: kooix-hajimi
    spec:
      containers:
      - name: kooix-hajimi
        image: ghcr.io/telagod/kooix-hajimi:latest
        ports:
        - containerPort: 8080
        env:
        - name: HAJIMI_GITHUB_TOKENS
          valueFrom:
            secretKeyRef:
              name: github-tokens
              key: tokens
        - name: HAJIMI_STORAGE_DSN
          valueFrom:
            secretKeyRef:
              name: database
              key: dsn
        resources:
          limits:
            memory: "1Gi"
            cpu: "1000m"
          requests:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /api/status
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/status
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: kooix-hajimi-service
spec:
  selector:
    app: kooix-hajimi
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Secret配置
```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-tokens
type: Opaque
data:
  tokens: <base64-encoded-tokens>
---
apiVersion: v1
kind: Secret
metadata:
  name: database
type: Opaque
data:
  dsn: <base64-encoded-database-dsn>
```

## 云服务部署

### AWS ECS
```json
{
  "family": "kooix-hajimi",
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "containerDefinitions": [
    {
      "name": "kooix-hajimi",
      "image": "ghcr.io/telagod/kooix-hajimi:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "HAJIMI_STORAGE_TYPE",
          "value": "postgres"
        }
      ],
      "secrets": [
        {
          "name": "HAJIMI_GITHUB_TOKENS",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:github-tokens"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/kooix-hajimi",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

### Google Cloud Run
```yaml
# cloudrun.yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: kooix-hajimi
  annotations:
    run.googleapis.com/ingress: all
    run.googleapis.com/execution-environment: gen2
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/maxScale: "10"
        run.googleapis.com/cpu-throttling: "false"
        run.googleapis.com/memory: "2Gi"
        run.googleapis.com/cpu: "2"
    spec:
      containerConcurrency: 1000
      timeoutSeconds: 300
      containers:
      - image: ghcr.io/telagod/kooix-hajimi:latest
        ports:
        - name: http1
          containerPort: 8080
        env:
        - name: HAJIMI_GITHUB_TOKENS
          valueFrom:
            secretKeyRef:
              name: github-tokens
              key: tokens
        resources:
          limits:
            memory: "2Gi"
            cpu: "2"
```

## 环境特定配置

### 开发环境
```bash
# .env.development
HAJIMI_LOG_LEVEL=debug
HAJIMI_STORAGE_TYPE=sqlite
HAJIMI_SCANNER_WORKER_COUNT=5
HAJIMI_SECURITY_NOTIFICATIONS_DRY_RUN=true
```

### 生产环境
```bash
# .env.production
HAJIMI_LOG_LEVEL=info
HAJIMI_STORAGE_TYPE=postgres
HAJIMI_SCANNER_WORKER_COUNT=50
HAJIMI_RATE_LIMIT_REQUESTS_PER_MINUTE=100
HAJIMI_SECURITY_NOTIFICATIONS_NOTIFY_ON_SEVERITY=high
```

## 部署检查清单

### 部署前检查
- [ ] GitHub tokens已准备并测试
- [ ] 数据库连接配置正确
- [ ] 环境变量已设置
- [ ] 存储卷已准备
- [ ] 网络端口已开放

### 部署后验证
- [ ] 服务健康检查通过
- [ ] Web界面可正常访问
- [ ] API接口响应正常
- [ ] WebSocket连接正常
- [ ] 数据库连接正常
- [ ] 日志输出正常

### 安全检查
- [ ] 敏感信息使用Secret管理
- [ ] 网络安全组配置正确
- [ ] HTTPS证书配置
- [ ] 访问日志启用
- [ ] 监控告警配置

## 故障排除

### 常见问题
1. **容器启动失败**: 检查环境变量和依赖服务
2. **数据库连接失败**: 验证数据库DSN和网络连接
3. **GitHub API限制**: 检查token有效性和权限
4. **内存不足**: 调整资源限制和worker数量

### 日志分析
```bash
# Docker日志
docker logs kooix-hajimi

# Kubernetes日志
kubectl logs -f deployment/kooix-hajimi

# 应用日志
tail -f /app/logs/hajimi-king.log
```

### 性能调优
- 根据负载调整worker数量
- 配置适当的内存限制
- 启用数据库连接池
- 使用Redis缓存（可选）