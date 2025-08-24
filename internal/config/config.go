package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	GitHub    GitHubConfig    `mapstructure:"github"`
	Scanner   ScannerConfig   `mapstructure:"scanner"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Web       WebConfig       `mapstructure:"web"`
	Sync      SyncConfig      `mapstructure:"sync"`
	Log       LogConfig       `mapstructure:"log"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

// GitHubConfig GitHub相关配置
type GitHubConfig struct {
	Tokens     []string      `mapstructure:"tokens"`
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
	UserAgent  string        `mapstructure:"user_agent"`
}

// ScannerConfig 扫描器配置
type ScannerConfig struct {
	WorkerCount    int           `mapstructure:"worker_count"`
	BatchSize      int           `mapstructure:"batch_size"`
	ScanInterval   time.Duration `mapstructure:"scan_interval"`
	AutoStart      bool          `mapstructure:"auto_start"`
	QueryFile      string        `mapstructure:"query_file"`
	DateRangeDays  int           `mapstructure:"date_range_days"`
	FileBlacklist  []string      `mapstructure:"file_blacklist"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type     string `mapstructure:"type"` // sqlite, postgres, file
	DSN      string `mapstructure:"dsn"`
	DataPath string `mapstructure:"data_path"`
	
	// 文件存储配置
	ValidKeyPrefix        string `mapstructure:"valid_key_prefix"`
	RateLimitedKeyPrefix  string `mapstructure:"rate_limited_key_prefix"`
	KeysSendPrefix        string `mapstructure:"keys_send_prefix"`
	
	// 数据库配置
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// WebConfig Web服务配置
type WebConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	
	// 静态文件配置
	StaticDir     string `mapstructure:"static_dir"`
	TemplateDir   string `mapstructure:"template_dir"`
	
	// CORS配置
	CORSEnabled     bool     `mapstructure:"cors_enabled"`
	CORSOrigins     []string `mapstructure:"cors_origins"`
	
	// 认证配置
	AuthEnabled bool   `mapstructure:"auth_enabled"`
	AuthToken   string `mapstructure:"auth_token"`
}

// SyncConfig 外部同步配置
type SyncConfig struct {
	Enabled bool `mapstructure:"enabled"`
	
	// Gemini Balancer
	GeminiBalancer GeminiBalancerConfig `mapstructure:"gemini_balancer"`
	
	// GPT Load Balancer
	GPTLoadBalancer GPTLoadBalancerConfig `mapstructure:"gpt_load_balancer"`
}

// GeminiBalancerConfig Gemini Balancer配置
type GeminiBalancerConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
	Auth    string `mapstructure:"auth"`
}

// GPTLoadBalancerConfig GPT Load Balancer配置
type GPTLoadBalancerConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	URL        string   `mapstructure:"url"`
	Auth       string   `mapstructure:"auth"`
	GroupNames []string `mapstructure:"group_names"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // json, text
	Output     string `mapstructure:"output"` // stdout, file
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	RequestsPerMinute int           `mapstructure:"requests_per_minute"`
	BurstSize         int           `mapstructure:"burst_size"`
	CooldownDuration  time.Duration `mapstructure:"cooldown_duration"`
	
	// 智能限流配置
	AdaptiveEnabled   bool    `mapstructure:"adaptive_enabled"`
	SuccessThreshold  float64 `mapstructure:"success_threshold"`
	BackoffMultiplier float64 `mapstructure:"backoff_multiplier"`
}

// Load 加载配置
func Load() (*Config, error) {
	viper.SetDefault("github.timeout", "30s")
	viper.SetDefault("github.max_retries", 5)
	viper.SetDefault("github.user_agent", "HajimiKing/2.0")
	
	viper.SetDefault("scanner.worker_count", 20)
	viper.SetDefault("scanner.batch_size", 100)
	viper.SetDefault("scanner.scan_interval", "10s")
	viper.SetDefault("scanner.auto_start", false)
	viper.SetDefault("scanner.query_file", "queries.txt")
	viper.SetDefault("scanner.date_range_days", 730)
	viper.SetDefault("scanner.file_blacklist", []string{
		"readme", "docs", "doc/", ".md", "example", "sample", 
		"tutorial", "test", "spec", "demo", "mock",
	})
	
	viper.SetDefault("storage.type", "sqlite")
	viper.SetDefault("storage.data_path", "./data")
	viper.SetDefault("storage.max_open_conns", 10)
	viper.SetDefault("storage.max_idle_conns", 5)
	viper.SetDefault("storage.conn_max_lifetime", "1h")
	
	viper.SetDefault("web.enabled", true)
	viper.SetDefault("web.host", "0.0.0.0")
	viper.SetDefault("web.port", 8080)
	viper.SetDefault("web.static_dir", "web/static")
	viper.SetDefault("web.template_dir", "web/templates")
	viper.SetDefault("web.cors_enabled", true)
	viper.SetDefault("web.cors_origins", []string{"*"})
	
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)
	
	viper.SetDefault("rate_limit.enabled", true)
	viper.SetDefault("rate_limit.requests_per_minute", 30)
	viper.SetDefault("rate_limit.burst_size", 10)
	viper.SetDefault("rate_limit.cooldown_duration", "5m")
	viper.SetDefault("rate_limit.adaptive_enabled", true)
	viper.SetDefault("rate_limit.success_threshold", 0.8)
	viper.SetDefault("rate_limit.backoff_multiplier", 1.5)

	// 设置配置文件名和路径
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/hajimi-king")

	// 环境变量支持
	viper.AutomaticEnv()
	viper.SetEnvPrefix("HAJIMI")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 解析配置
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// validate 验证配置
func validate(cfg *Config) error {
	if len(cfg.GitHub.Tokens) == 0 {
		return fmt.Errorf("github tokens are required")
	}

	if cfg.Scanner.WorkerCount <= 0 {
		return fmt.Errorf("scanner worker count must be greater than 0")
	}

	if cfg.Web.Enabled && cfg.Web.Port <= 0 {
		return fmt.Errorf("web port must be greater than 0")
	}

	return nil
}