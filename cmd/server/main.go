package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kooix-hajimi/internal/config"
	"kooix-hajimi/internal/scanner"
	"kooix-hajimi/internal/web"
	"kooix-hajimi/pkg/logger"
)

func main() {
	// 初始化配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger.Init(cfg.Log)

	// 创建扫描器
	scannerInstance, err := scanner.New(cfg)
	if err != nil {
		logger.Fatalf("Failed to create scanner: %v", err)
	}

	// 创建Web服务器
	webServer, err := web.New(cfg, scannerInstance)
	if err != nil {
		logger.Fatalf("Failed to create web server: %v", err)
	}

	// 启动Web服务器
	go func() {
		if err := webServer.Start(); err != nil {
			logger.Fatalf("Failed to start web server: %v", err)
		}
	}()

	// 如果启用了自动扫描，启动扫描器
	if cfg.Scanner.AutoStart {
		go func() {
			ctx := context.Background()
			if err := scannerInstance.StartContinuousScanning(ctx); err != nil {
				logger.Errorf("Scanner error: %v", err)
			}
		}()
	}

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := webServer.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}