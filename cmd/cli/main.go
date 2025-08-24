package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"kooix-hajimi/internal/config"
	"kooix-hajimi/internal/scanner"
	"kooix-hajimi/pkg/logger"
)

var (
	configFile string
	queries    []string
	output     string
)

var rootCmd = &cobra.Command{
	Use:   "hajimi-king",
	Short: "GitHub API key discovery tool",
	Long:  `A high-performance tool for discovering and validating API keys across GitHub repositories.`,
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Start scanning for API keys",
	Run:   runScan,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file path")
	
	scanCmd.Flags().StringSliceVarP(&queries, "query", "q", nil, "search queries")
	scanCmd.Flags().StringVarP(&output, "output", "o", "", "output directory")
	
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger.Init(cfg.Log)

	// 创建扫描器
	scanner, err := scanner.New(cfg)
	if err != nil {
		logger.Fatalf("Failed to create scanner: %v", err)
	}

	// 执行扫描
	ctx := context.Background()
	if len(queries) > 0 {
		err = scanner.ScanWithQueries(ctx, queries)
	} else {
		err = scanner.StartContinuousScanning(ctx)
	}

	if err != nil {
		logger.Fatalf("Scan failed: %v", err)
	}

	logger.Info("Scan completed successfully")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}