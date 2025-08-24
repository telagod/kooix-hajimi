package storage

import (
	"context"
	"time"
)

// ValidKey 有效密钥
type ValidKey struct {
	ID          int64     `json:"id" db:"id"`
	Key         string    `json:"key" db:"key_value"`
	Source      string    `json:"source" db:"source"`
	RepoName    string    `json:"repo_name" db:"repo_name"`
	FilePath    string    `json:"file_path" db:"file_path"`
	FileURL     string    `json:"file_url" db:"file_url"`
	SHA         string    `json:"sha" db:"sha"`
	ValidatedAt time.Time `json:"validated_at" db:"validated_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// RateLimitedKey 被限流的密钥
type RateLimitedKey struct {
	ID          int64     `json:"id" db:"id"`
	Key         string    `json:"key" db:"key_value"`
	Source      string    `json:"source" db:"source"`
	RepoName    string    `json:"repo_name" db:"repo_name"`
	FilePath    string    `json:"file_path" db:"file_path"`
	FileURL     string    `json:"file_url" db:"file_url"`
	SHA         string    `json:"sha" db:"sha"`
	Reason      string    `json:"reason" db:"reason"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ScanProgress 扫描进度
type ScanProgress struct {
	ID                  int64     `json:"id" db:"id"`
	LastScanTime        time.Time `json:"last_scan_time" db:"last_scan_time"`
	TotalFilesScanned   int64     `json:"total_files_scanned" db:"total_files_scanned"`
	ValidKeysFound      int64     `json:"valid_keys_found" db:"valid_keys_found"`
	RateLimitedKeys     int64     `json:"rate_limited_keys" db:"rate_limited_keys"`
	QueriesProcessed    int       `json:"queries_processed" db:"queries_processed"`
	IsScanning          bool      `json:"is_scanning" db:"is_scanning"`
	CurrentQuery        string    `json:"current_query" db:"current_query"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// Checkpoint 检查点数据
type Checkpoint struct {
	ID                 int64     `json:"id" db:"id"`
	ScannedSHAs        []string  `json:"scanned_shas" db:"scanned_shas"`
	ProcessedQueries   []string  `json:"processed_queries" db:"processed_queries"`
	WaitSendBalancer   []string  `json:"wait_send_balancer" db:"wait_send_balancer"`
	WaitSendGPTLoad    []string  `json:"wait_send_gpt_load" db:"wait_send_gpt_load"`
	LastScanTime       time.Time `json:"last_scan_time" db:"last_scan_time"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// ScanStats 扫描统计
type ScanStats struct {
	TotalKeys          int64     `json:"total_keys"`
	ValidKeys          int64     `json:"valid_keys"`
	RateLimitedKeys    int64     `json:"rate_limited_keys"`
	TotalFilesScanned  int64     `json:"total_files_scanned"`
	LastScanTime       time.Time `json:"last_scan_time"`
	ScanningActive     bool      `json:"scanning_active"`
}

// KeyFilter 密钥过滤条件
type KeyFilter struct {
	Source     string    `json:"source,omitempty"`
	RepoName   string    `json:"repo_name,omitempty"`
	DateFrom   time.Time `json:"date_from,omitempty"`
	DateTo     time.Time `json:"date_to,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
	OrderBy    string    `json:"order_by,omitempty"`
	OrderDir   string    `json:"order_dir,omitempty"`
}

// Storage 存储接口
type Storage interface {
	// 密钥管理
	SaveValidKeys(ctx context.Context, keys []*ValidKey) error
	SaveRateLimitedKeys(ctx context.Context, keys []*RateLimitedKey) error
	GetValidKeys(ctx context.Context, filter *KeyFilter) ([]*ValidKey, int64, error)
	GetRateLimitedKeys(ctx context.Context, filter *KeyFilter) ([]*RateLimitedKey, int64, error)
	DeleteValidKey(ctx context.Context, id int64) error
	DeleteRateLimitedKey(ctx context.Context, id int64) error
	
	// 检查点管理
	SaveCheckpoint(ctx context.Context, checkpoint *Checkpoint) error
	LoadCheckpoint(ctx context.Context) (*Checkpoint, error)
	
	// 扫描进度
	UpdateScanProgress(ctx context.Context, progress *ScanProgress) error
	GetScanProgress(ctx context.Context) (*ScanProgress, error)
	
	// 统计信息
	GetScanStats(ctx context.Context) (*ScanStats, error)
	
	// SHA管理
	IsSHAScanned(ctx context.Context, sha string) (bool, error)
	AddScannedSHA(ctx context.Context, sha string) error
	GetScannedSHAsCount(ctx context.Context) (int64, error)
	
	// 查询管理
	IsQueryProcessed(ctx context.Context, query string) (bool, error)
	AddProcessedQuery(ctx context.Context, query string) error
	
	// 同步队列管理
	AddKeysToBalancerQueue(ctx context.Context, keys []string) error
	AddKeysToGPTLoadQueue(ctx context.Context, keys []string) error
	GetBalancerQueue(ctx context.Context) ([]string, error)
	GetGPTLoadQueue(ctx context.Context) ([]string, error)
	ClearBalancerQueue(ctx context.Context) error
	ClearGPTLoadQueue(ctx context.Context) error
	
	// 健康检查和清理
	HealthCheck(ctx context.Context) error
	Close() error
}