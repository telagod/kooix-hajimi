# 构建Go版本的Hajimi King - Multi-platform support
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

# 声明构建参数
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# 安装必要的包，包含跨平台构建工具
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev \
    sqlite-dev \
    build-base

WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 设置SQLite编译选项，解决多平台兼容性问题
ENV CGO_ENABLED=1
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

# 为不同架构设置不同的编译选项
RUN case $TARGETARCH in \
        amd64) \
            export CC=gcc; \
            export CGO_LDFLAGS="-static -w -s"; \
            ;; \
        arm64) \
            export CC=gcc; \
            export CGO_LDFLAGS="-static -w -s"; \
            ;; \
        *) \
            export CC=gcc; \
            export CGO_LDFLAGS="-static -w -s"; \
            ;; \
    esac && \
    go build -tags "sqlite_omit_load_extension netgo osusergo static_build" \
             -ldflags "-linkmode external -extldflags '-static' -s -w" \
             -a -installsuffix netgo \
             -o kooix-hajimi-server cmd/server/main.go && \
    go build -tags "sqlite_omit_load_extension netgo osusergo static_build" \
             -ldflags "-linkmode external -extldflags '-static' -s -w" \
             -a -installsuffix netgo \
             -o kooix-hajimi-cli cmd/cli/main.go

# 最终镜像 - 使用alpine获得良好的兼容性
FROM alpine:latest

# 安装必要的运行时包  
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1001 kooix && \
    adduser -D -s /bin/sh -u 1001 -G kooix kooix && \
    mkdir -p /app/data && \
    chown -R kooix:kooix /app

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/kooix-hajimi-server .
COPY --from=builder /app/kooix-hajimi-cli .

# 复制配置和静态文件
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/web ./web

# 更改文件所有权
RUN chown -R kooix:kooix /app

USER kooix

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./kooix-hajimi-server"]