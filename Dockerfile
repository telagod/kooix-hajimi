# 构建Go版本的Hajimi King
FROM golang:1.21-alpine AS builder

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev sqlite-dev

WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o kooix-hajimi-server cmd/server/main.go
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o kooix-hajimi-cli cmd/cli/main.go

# 最终镜像
FROM alpine:latest

# 安装必要的包
RUN apk --no-cache add ca-certificates tzdata sqlite

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/kooix-hajimi-server .
COPY --from=builder /app/kooix-hajimi-cli .

# 复制配置和静态文件
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/web ./web

# 创建数据目录
RUN mkdir -p /app/data

# 创建非root用户
RUN addgroup -g 1000 kooix && \
    adduser -D -s /bin/sh -u 1000 -G kooix kooix

# 更改文件所有权
RUN chown -R kooix:kooix /app

USER kooix

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/status || exit 1

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./kooix-hajimi-server"]