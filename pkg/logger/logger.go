package logger

import (
	"io"
	"os"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"

	"kooix-hajimi/internal/config"
)

var log *logrus.Logger

// Init 初始化日志
func Init(cfg config.LogConfig) {
	log = logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// 设置日志格式
	if cfg.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}

	// 设置输出目标
	switch cfg.Output {
	case "file":
		if cfg.Filename == "" {
			cfg.Filename = "hajimi-king.log"
		}
		
		logger := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,    // megabytes
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,     // days
			Compress:   cfg.Compress,
		}
		
		log.SetOutput(logger)
	case "both":
		if cfg.Filename == "" {
			cfg.Filename = "hajimi-king.log"
		}
		
		logger := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		
		multiWriter := io.MultiWriter(os.Stdout, logger)
		log.SetOutput(multiWriter)
	default:
		log.SetOutput(os.Stdout)
	}
}

// GetLogger 获取日志实例
func GetLogger() *logrus.Logger {
	if log == nil {
		log = logrus.New()
	}
	return log
}

// Debug 调试日志
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info 信息日志
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn 警告日志
func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error 错误日志
func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// Fatal 致命错误日志
func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}