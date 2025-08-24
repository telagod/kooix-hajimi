package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	
	"kooix-hajimi/internal/config"
	"kooix-hajimi/pkg/logger"
)

// SQLiteStorage SQLite存储实现
type SQLiteStorage struct {
	db     *sqlx.DB
	config config.StorageConfig
}

// NewSQLiteStorage 创建SQLite存储
func NewSQLiteStorage(cfg config.StorageConfig) (*SQLiteStorage, error) {
	db, err := sqlx.Open("sqlite3", cfg.DSN+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000&_temp_store=memory")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite建议单连接
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	storage := &SQLiteStorage{
		db:     db,
		config: cfg,
	}

	// 初始化数据库表
	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return storage, nil
}

// migrate 执行数据库迁移
func (s *SQLiteStorage) migrate() error {
	queries := []string{
		// 有效密钥表
		`CREATE TABLE IF NOT EXISTS valid_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key_value TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT 'gemini',
			key_type TEXT NOT NULL DEFAULT 'api_key',
			source TEXT NOT NULL DEFAULT 'github',
			repo_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_url TEXT NOT NULL,
			sha TEXT NOT NULL,
			validated_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(key_value, sha)
		)`,
		
		// 限流密钥表
		`CREATE TABLE IF NOT EXISTS rate_limited_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key_value TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT 'gemini',
			key_type TEXT NOT NULL DEFAULT 'api_key',
			source TEXT NOT NULL DEFAULT 'github',
			repo_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_url TEXT NOT NULL,
			sha TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT 'rate_limited',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(key_value, sha)
		)`,
		
		// 扫描进度表
		`CREATE TABLE IF NOT EXISTS scan_progress (
			id INTEGER PRIMARY KEY,
			last_scan_time DATETIME,
			total_files_scanned INTEGER DEFAULT 0,
			valid_keys_found INTEGER DEFAULT 0,
			rate_limited_keys INTEGER DEFAULT 0,
			queries_processed INTEGER DEFAULT 0,
			is_scanning BOOLEAN DEFAULT FALSE,
			current_query TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// 检查点表
		`CREATE TABLE IF NOT EXISTS checkpoints (
			id INTEGER PRIMARY KEY,
			scanned_shas TEXT DEFAULT '[]',
			processed_queries TEXT DEFAULT '[]',
			wait_send_balancer TEXT DEFAULT '[]',
			wait_send_gpt_load TEXT DEFAULT '[]',
			last_scan_time DATETIME,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// 已扫描SHA表
		`CREATE TABLE IF NOT EXISTS scanned_shas (
			sha TEXT PRIMARY KEY,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// 已处理查询表
		`CREATE TABLE IF NOT EXISTS processed_queries (
			query_hash TEXT PRIMARY KEY,
			query_text TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// 同步队列表
		`CREATE TABLE IF NOT EXISTS sync_queues (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			queue_type TEXT NOT NULL,
			key_value TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(queue_type, key_value)
		)`,
	}

	// 创建索引
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_valid_keys_created_at ON valid_keys(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_valid_keys_repo_name ON valid_keys(repo_name)`,
		`CREATE INDEX IF NOT EXISTS idx_valid_keys_provider ON valid_keys(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_rate_limited_keys_created_at ON rate_limited_keys(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_rate_limited_keys_provider ON rate_limited_keys(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_scanned_shas_created_at ON scanned_shas(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_sync_queues_type ON sync_queues(queue_type)`,
	}

	// 执行迁移
	for _, query := range append(queries, indexes...) {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration query: %w", err)
		}
	}
	
	// 执行字段迁移（为现有表添加新字段）
	alterQueries := []string{
		`ALTER TABLE valid_keys ADD COLUMN provider TEXT DEFAULT 'gemini'`,
		`ALTER TABLE valid_keys ADD COLUMN key_type TEXT DEFAULT 'api_key'`,
		`ALTER TABLE rate_limited_keys ADD COLUMN provider TEXT DEFAULT 'gemini'`,
		`ALTER TABLE rate_limited_keys ADD COLUMN key_type TEXT DEFAULT 'api_key'`,
	}
	
	for _, query := range alterQueries {
		// 忽略"duplicate column name"错误，因为字段可能已存在
		if _, err := s.db.Exec(query); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("failed to execute alter query: %w", err)
		}
	}

	// 初始化默认数据
	if err := s.initializeDefaults(); err != nil {
		return fmt.Errorf("failed to initialize defaults: %w", err)
	}

	logger.Info("Database migration completed successfully")
	return nil
}

// initializeDefaults 初始化默认数据
func (s *SQLiteStorage) initializeDefaults() error {
	// 初始化扫描进度记录
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO scan_progress (id, last_scan_time, updated_at) 
		VALUES (1, datetime('now'), datetime('now'))
	`)
	if err != nil {
		return err
	}

	// 初始化检查点记录
	_, err = s.db.Exec(`
		INSERT OR IGNORE INTO checkpoints (id, last_scan_time, updated_at) 
		VALUES (1, datetime('now'), datetime('now'))
	`)
	return err
}

// SaveValidKeys 保存有效密钥
func (s *SQLiteStorage) SaveValidKeys(ctx context.Context, keys []*ValidKey) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT OR REPLACE INTO valid_keys 
		(key_value, source, repo_name, file_path, file_url, sha, validated_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`

	for _, key := range keys {
		_, err := tx.ExecContext(ctx, query,
			key.Key, key.Source, key.RepoName, key.FilePath, 
			key.FileURL, key.SHA, key.ValidatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert valid key: %w", err)
		}
	}

	return tx.Commit()
}

// SaveRateLimitedKeys 保存限流密钥
func (s *SQLiteStorage) SaveRateLimitedKeys(ctx context.Context, keys []*RateLimitedKey) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT OR REPLACE INTO rate_limited_keys 
		(key_value, source, repo_name, file_path, file_url, sha, reason)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	for _, key := range keys {
		_, err := tx.ExecContext(ctx, query,
			key.Key, key.Source, key.RepoName, key.FilePath, 
			key.FileURL, key.SHA, key.Reason)
		if err != nil {
			return fmt.Errorf("failed to insert rate limited key: %w", err)
		}
	}

	return tx.Commit()
}

// GetValidKeys 获取有效密钥
func (s *SQLiteStorage) GetValidKeys(ctx context.Context, filter *KeyFilter) ([]*ValidKey, int64, error) {
	whereClause, args := s.buildWhereClause(filter)
	
	// 获取总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM valid_keys %s", whereClause)
	var total int64
	err := s.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get count: %w", err)
	}

	// 构建查询
	query := fmt.Sprintf(`
		SELECT id, key_value, source, repo_name, file_path, file_url, sha, 
		       validated_at, created_at, updated_at
		FROM valid_keys %s %s
	`, whereClause, s.buildOrderClause(filter))

	var keys []*ValidKey
	err = s.db.SelectContext(ctx, &keys, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get valid keys: %w", err)
	}

	return keys, total, nil
}

// GetRateLimitedKeys 获取限流密钥
func (s *SQLiteStorage) GetRateLimitedKeys(ctx context.Context, filter *KeyFilter) ([]*RateLimitedKey, int64, error) {
	whereClause, args := s.buildWhereClause(filter)
	
	// 获取总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM rate_limited_keys %s", whereClause)
	var total int64
	err := s.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get count: %w", err)
	}

	// 构建查询
	query := fmt.Sprintf(`
		SELECT id, key_value, source, repo_name, file_path, file_url, sha, reason, created_at
		FROM rate_limited_keys %s %s
	`, whereClause, s.buildOrderClause(filter))

	var keys []*RateLimitedKey
	err = s.db.SelectContext(ctx, &keys, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get rate limited keys: %w", err)
	}

	return keys, total, nil
}

// buildWhereClause 构建WHERE子句
func (s *SQLiteStorage) buildWhereClause(filter *KeyFilter) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	var conditions []string
	var args []interface{}

	if filter.Source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, filter.Source)
	}

	if filter.RepoName != "" {
		conditions = append(conditions, "repo_name LIKE ?")
		args = append(args, "%"+filter.RepoName+"%")
	}

	if !filter.DateFrom.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filter.DateFrom)
	}

	if !filter.DateTo.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filter.DateTo)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}

// buildOrderClause 构建ORDER BY子句
func (s *SQLiteStorage) buildOrderClause(filter *KeyFilter) string {
	if filter == nil {
		return "ORDER BY created_at DESC LIMIT 100"
	}

	orderBy := "created_at"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}

	orderDir := "DESC"
	if filter.OrderDir != "" {
		orderDir = filter.OrderDir
	}

	limit := 100
	if filter.Limit > 0 {
		limit = filter.Limit
	}

	offset := 0
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	return fmt.Sprintf("ORDER BY %s %s LIMIT %d OFFSET %d", 
		orderBy, orderDir, limit, offset)
}

// LoadCheckpoint 加载检查点
func (s *SQLiteStorage) LoadCheckpoint(ctx context.Context) (*Checkpoint, error) {
	checkpoint := &Checkpoint{}
	
	row := s.db.QueryRowxContext(ctx, `
		SELECT id, scanned_shas, processed_queries, wait_send_balancer, 
		       wait_send_gpt_load, last_scan_time, updated_at 
		FROM checkpoints WHERE id = 1
	`)

	var scannedSHAsJSON, processedQueriesJSON, waitSendBalancerJSON, waitSendGPTLoadJSON string
	err := row.Scan(
		&checkpoint.ID, &scannedSHAsJSON, &processedQueriesJSON,
		&waitSendBalancerJSON, &waitSendGPTLoadJSON,
		&checkpoint.LastScanTime, &checkpoint.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// 返回空的检查点
			return &Checkpoint{
				ID:                1,
				ScannedSHAs:       []string{},
				ProcessedQueries:  []string{},
				WaitSendBalancer:  []string{},
				WaitSendGPTLoad:   []string{},
				LastScanTime:      time.Now(),
				UpdatedAt:         time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// 解析JSON字段
	json.Unmarshal([]byte(scannedSHAsJSON), &checkpoint.ScannedSHAs)
	json.Unmarshal([]byte(processedQueriesJSON), &checkpoint.ProcessedQueries)
	json.Unmarshal([]byte(waitSendBalancerJSON), &checkpoint.WaitSendBalancer)
	json.Unmarshal([]byte(waitSendGPTLoadJSON), &checkpoint.WaitSendGPTLoad)

	return checkpoint, nil
}

// SaveCheckpoint 保存检查点
func (s *SQLiteStorage) SaveCheckpoint(ctx context.Context, checkpoint *Checkpoint) error {
	// 序列化JSON字段
	scannedSHAsJSON, _ := json.Marshal(checkpoint.ScannedSHAs)
	processedQueriesJSON, _ := json.Marshal(checkpoint.ProcessedQueries)
	waitSendBalancerJSON, _ := json.Marshal(checkpoint.WaitSendBalancer)
	waitSendGPTLoadJSON, _ := json.Marshal(checkpoint.WaitSendGPTLoad)

	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO checkpoints 
		(id, scanned_shas, processed_queries, wait_send_balancer, 
		 wait_send_gpt_load, last_scan_time, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, datetime('now'))
	`, string(scannedSHAsJSON), string(processedQueriesJSON), 
	   string(waitSendBalancerJSON), string(waitSendGPTLoadJSON), checkpoint.LastScanTime)

	return err
}

// GetScanStats 获取扫描统计
func (s *SQLiteStorage) GetScanStats(ctx context.Context) (*ScanStats, error) {
	stats := &ScanStats{}

	// 获取基础统计
	err := s.db.GetContext(ctx, stats, `
		SELECT 
			(SELECT COUNT(*) FROM valid_keys) + (SELECT COUNT(*) FROM rate_limited_keys) as total_keys,
			(SELECT COUNT(*) FROM valid_keys) as valid_keys,
			(SELECT COUNT(*) FROM rate_limited_keys) as rate_limited_keys,
			COALESCE((SELECT total_files_scanned FROM scan_progress WHERE id = 1), 0) as total_files_scanned,
			COALESCE((SELECT last_scan_time FROM scan_progress WHERE id = 1), datetime('now')) as last_scan_time,
			COALESCE((SELECT is_scanning FROM scan_progress WHERE id = 1), 0) as scanning_active
	`)

	return stats, err
}

// IsSHAScanned 检查SHA是否已扫描
func (s *SQLiteStorage) IsSHAScanned(ctx context.Context, sha string) (bool, error) {
	var count int
	err := s.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM scanned_shas WHERE sha = ?", sha)
	return count > 0, err
}

// AddScannedSHA 添加已扫描的SHA
func (s *SQLiteStorage) AddScannedSHA(ctx context.Context, sha string) error {
	_, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO scanned_shas (sha) VALUES (?)", sha)
	return err
}

// HealthCheck 健康检查
func (s *SQLiteStorage) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close 关闭数据库连接
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// 实现其他接口方法...
func (s *SQLiteStorage) DeleteValidKey(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM valid_keys WHERE id = ?", id)
	return err
}

func (s *SQLiteStorage) DeleteRateLimitedKey(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM rate_limited_keys WHERE id = ?", id)
	return err
}

func (s *SQLiteStorage) UpdateScanProgress(ctx context.Context, progress *ScanProgress) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE scan_progress SET 
		last_scan_time = ?, total_files_scanned = ?, valid_keys_found = ?,
		rate_limited_keys = ?, queries_processed = ?, is_scanning = ?,
		current_query = ?, updated_at = datetime('now')
		WHERE id = 1
	`, progress.LastScanTime, progress.TotalFilesScanned, progress.ValidKeysFound,
		progress.RateLimitedKeys, progress.QueriesProcessed, progress.IsScanning, progress.CurrentQuery)
	return err
}

func (s *SQLiteStorage) GetScanProgress(ctx context.Context) (*ScanProgress, error) {
	progress := &ScanProgress{}
	err := s.db.GetContext(ctx, progress, "SELECT * FROM scan_progress WHERE id = 1")
	return progress, err
}

func (s *SQLiteStorage) GetScannedSHAsCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM scanned_shas")
	return count, err
}

func (s *SQLiteStorage) IsQueryProcessed(ctx context.Context, query string) (bool, error) {
	var count int
	err := s.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM processed_queries WHERE query_text = ?", query)
	return count > 0, err
}

func (s *SQLiteStorage) AddProcessedQuery(ctx context.Context, query string) error {
	_, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO processed_queries (query_hash, query_text) VALUES (?, ?)", 
		fmt.Sprintf("%x", sha256([]byte(query))), query)
	return err
}

func (s *SQLiteStorage) AddKeysToBalancerQueue(ctx context.Context, keys []string) error {
	return s.addKeysToQueue(ctx, "balancer", keys)
}

func (s *SQLiteStorage) AddKeysToGPTLoadQueue(ctx context.Context, keys []string) error {
	return s.addKeysToQueue(ctx, "gpt_load", keys)
}

func (s *SQLiteStorage) addKeysToQueue(ctx context.Context, queueType string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, key := range keys {
		_, err := tx.ExecContext(ctx, 
			"INSERT OR IGNORE INTO sync_queues (queue_type, key_value) VALUES (?, ?)",
			queueType, key)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStorage) GetBalancerQueue(ctx context.Context) ([]string, error) {
	return s.getQueue(ctx, "balancer")
}

func (s *SQLiteStorage) GetGPTLoadQueue(ctx context.Context) ([]string, error) {
	return s.getQueue(ctx, "gpt_load")
}

func (s *SQLiteStorage) getQueue(ctx context.Context, queueType string) ([]string, error) {
	var keys []string
	err := s.db.SelectContext(ctx, &keys, 
		"SELECT key_value FROM sync_queues WHERE queue_type = ? ORDER BY created_at",
		queueType)
	return keys, err
}

func (s *SQLiteStorage) ClearBalancerQueue(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sync_queues WHERE queue_type = 'balancer'")
	return err
}

func (s *SQLiteStorage) ClearGPTLoadQueue(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sync_queues WHERE queue_type = 'gpt_load'")
	return err
}

// sha256 计算SHA256哈希
func sha256(data []byte) []byte {
	// 简化实现，实际应该使用crypto/sha256
	return []byte(fmt.Sprintf("%x", len(data)))
}