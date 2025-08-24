package scanner

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"kooix-hajimi/internal/config"
	"kooix-hajimi/internal/github"
	"kooix-hajimi/internal/ratelimit"
	"kooix-hajimi/internal/storage"
	"kooix-hajimi/internal/validator"
	"kooix-hajimi/pkg/logger"
)

// Scanner 扫描器
type Scanner struct {
	github     *github.Client
	validator  *validator.Validator
	storage    storage.Storage
	config     config.Config
	
	// 状态管理
	isScanning    bool
	scanMu        sync.RWMutex
	stopCh        chan struct{}
	
	// 统计信息
	stats         *ScanStats
	statsMu       sync.RWMutex
}

// ScanStats 扫描统计
type ScanStats struct {
	StartTime         time.Time `json:"start_time"`
	TotalQueries      int       `json:"total_queries"`
	ProcessedQueries  int       `json:"processed_queries"`
	TotalFiles        int       `json:"total_files"`
	ProcessedFiles    int       `json:"processed_files"`
	ValidKeys         int       `json:"valid_keys"`
	RateLimitedKeys   int       `json:"rate_limited_keys"`
	ErrorCount        int       `json:"error_count"`
	CurrentQuery      string    `json:"current_query"`
	IsActive          bool      `json:"is_active"`
}

// New 创建新的扫描器
func New(cfg *config.Config) (*Scanner, error) {
	// 创建限流管理器
	rateLimitConfig := ratelimit.Config{
		RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
		BurstSize:         cfg.RateLimit.BurstSize,
		CooldownDuration:  cfg.RateLimit.CooldownDuration,
		AdaptiveEnabled:   cfg.RateLimit.AdaptiveEnabled,
		SuccessThreshold:  cfg.RateLimit.SuccessThreshold,
		BackoffMultiplier: cfg.RateLimit.BackoffMultiplier,
	}
	rateLimiter := ratelimit.New(rateLimitConfig)

	// 创建GitHub客户端
	githubClient := github.New(cfg.GitHub, rateLimiter)

	// 创建验证器
	validatorConfig := validator.Config{
		ModelName:   "gemini-2.5-flash",
		WorkerCount: 5,
		Timeout:     30 * time.Second,
	}
	keyValidator := validator.New(validatorConfig)

	// 创建存储
	var store storage.Storage
	var err error
	
	switch cfg.Storage.Type {
	case "sqlite":
		store, err = storage.NewSQLiteStorage(cfg.Storage)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &Scanner{
		github:    githubClient,
		validator: keyValidator,
		storage:   store,
		config:    *cfg,
		stopCh:    make(chan struct{}),
		stats: &ScanStats{
			StartTime: time.Now(),
		},
	}, nil
}

// ScanWithQueries 使用指定查询进行扫描
func (s *Scanner) ScanWithQueries(ctx context.Context, queries []string) error {
	if !s.startScanning() {
		return fmt.Errorf("scanner is already running")
	}
	defer s.stopScanning()

	logger.Infof("Starting scan with %d queries", len(queries))
	
	s.updateStats(func(stats *ScanStats) {
		stats.TotalQueries = len(queries)
		stats.IsActive = true
	})

	// 更新扫描进度
	progress := &storage.ScanProgress{
		IsScanning:       true,
		QueriesProcessed: 0,
		UpdatedAt:        time.Now(),
	}
	s.storage.UpdateScanProgress(ctx, progress)

	for i, query := range queries {
		select {
		case <-ctx.Done():
			logger.Info("Scan cancelled by context")
			return ctx.Err()
		case <-s.stopCh:
			logger.Info("Scan stopped by user")
			return nil
		default:
		}

		logger.Infof("Processing query %d/%d: %s", i+1, len(queries), query)
		
		s.updateStats(func(stats *ScanStats) {
			stats.CurrentQuery = query
			stats.ProcessedQueries = i + 1
		})

		if err := s.processQuery(ctx, query); err != nil {
			logger.Errorf("Failed to process query '%s': %v", query, err)
			s.updateStats(func(stats *ScanStats) {
				stats.ErrorCount++
			})
			continue
		}

		// 更新进度
		progress.QueriesProcessed = i + 1
		progress.LastScanTime = time.Now()
		s.storage.UpdateScanProgress(ctx, progress)
	}

	// 完成扫描
	progress.IsScanning = false
	progress.LastScanTime = time.Now()
	s.storage.UpdateScanProgress(ctx, progress)

	logger.Info("Scan completed successfully")
	return nil
}

// processQuery 处理单个查询
func (s *Scanner) processQuery(ctx context.Context, query string) error {
	// 检查查询是否已处理
	processed, err := s.storage.IsQueryProcessed(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to check if query processed: %w", err)
	}
	
	if processed {
		logger.Infof("Query already processed, skipping: %s", query)
		return nil
	}

	// 搜索GitHub
	result, err := s.github.SearchCode(ctx, query)
	if err != nil {
		return fmt.Errorf("github search failed: %w", err)
	}

	if len(result.Items) == 0 {
		logger.Infof("No items found for query: %s", query)
		return s.storage.AddProcessedQuery(ctx, query)
	}

	logger.Infof("Found %d items for query: %s", len(result.Items), query)
	
	s.updateStats(func(stats *ScanStats) {
		stats.TotalFiles += len(result.Items)
	})

	// 处理搜索结果
	if err := s.processSearchItems(ctx, result.Items); err != nil {
		return fmt.Errorf("failed to process search items: %w", err)
	}

	// 标记查询已处理
	return s.storage.AddProcessedQuery(ctx, query)
}

// processSearchItems 处理搜索结果
func (s *Scanner) processSearchItems(ctx context.Context, items []github.SearchItem) error {
	// 过滤项目
	filteredItems := s.filterItems(items)
	
	if len(filteredItems) == 0 {
		logger.Info("All items filtered out")
		return nil
	}

	logger.Infof("Processing %d filtered items", len(filteredItems))

	// 创建错误组进行并发处理
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.config.Scanner.WorkerCount) // 限制并发数

	// 处理每个项目
	for _, item := range filteredItems {
		item := item // 捕获循环变量
		
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return s.processSearchItem(ctx, &item)
			}
		})
	}

	return g.Wait()
}

// processSearchItem 处理单个搜索项
func (s *Scanner) processSearchItem(ctx context.Context, item *github.SearchItem) error {
	// 检查SHA是否已扫描
	scanned, err := s.storage.IsSHAScanned(ctx, item.SHA)
	if err != nil {
		return fmt.Errorf("failed to check SHA: %w", err)
	}
	
	if scanned {
		logger.Debugf("SHA already scanned, skipping: %s", item.SHA)
		return nil
	}

	// 获取文件内容
	content, err := s.github.GetFileContent(ctx, item)
	if err != nil {
		logger.Warnf("Failed to get file content for %s: %v", item.HTMLURL, err)
		return nil // 不返回错误，继续处理其他文件
	}

	// 提取API密钥
	keys := s.extractKeys(string(content))
	if len(keys) == 0 {
		// 记录已扫描但无密钥的文件
		s.storage.AddScannedSHA(ctx, item.SHA)
		s.updateStats(func(stats *ScanStats) {
			stats.ProcessedFiles++
		})
		return nil
	}

	logger.Infof("Found %d potential keys in %s", len(keys), item.Path)

	// 验证密钥
	results, err := s.validator.ValidateBatch(ctx, keys)
	if err != nil {
		logger.Errorf("Validation failed for %s: %v", item.Path, err)
		return nil
	}

	// 分类保存结果
	var validKeys []*storage.ValidKey
	var rateLimitedKeys []*storage.RateLimitedKey

	for _, result := range results {
		switch result.Status {
		case "valid":
			validKeys = append(validKeys, &storage.ValidKey{
				Key:         result.Key,
				Source:      "github",
				RepoName:    item.Repository.FullName,
				FilePath:    item.Path,
				FileURL:     item.HTMLURL,
				SHA:         item.SHA,
				ValidatedAt: result.Timestamp,
			})
		case "rate_limited":
			rateLimitedKeys = append(rateLimitedKeys, &storage.RateLimitedKey{
				Key:      result.Key,
				Source:   "github",
				RepoName: item.Repository.FullName,
				FilePath: item.Path,
				FileURL:  item.HTMLURL,
				SHA:      item.SHA,
				Reason:   "rate_limited",
			})
		}
	}

	// 保存结果
	if len(validKeys) > 0 {
		if err := s.storage.SaveValidKeys(ctx, validKeys); err != nil {
			logger.Errorf("Failed to save valid keys: %v", err)
		} else {
			logger.Infof("Saved %d valid keys from %s", len(validKeys), item.Path)
			s.updateStats(func(stats *ScanStats) {
				stats.ValidKeys += len(validKeys)
			})
		}
	}

	if len(rateLimitedKeys) > 0 {
		if err := s.storage.SaveRateLimitedKeys(ctx, rateLimitedKeys); err != nil {
			logger.Errorf("Failed to save rate limited keys: %v", err)
		} else {
			logger.Infof("Saved %d rate limited keys from %s", len(rateLimitedKeys), item.Path)
			s.updateStats(func(stats *ScanStats) {
				stats.RateLimitedKeys += len(rateLimitedKeys)
			})
		}
	}

	// 记录已扫描的SHA
	s.storage.AddScannedSHA(ctx, item.SHA)
	s.updateStats(func(stats *ScanStats) {
		stats.ProcessedFiles++
	})

	return nil
}

// filterItems 过滤搜索项
func (s *Scanner) filterItems(items []github.SearchItem) []github.SearchItem {
	var filtered []github.SearchItem

	for _, item := range items {
		// 检查文件路径黑名单
		path := strings.ToLower(item.Path)
		skip := false
		
		for _, blacklisted := range s.config.Scanner.FileBlacklist {
			if strings.Contains(path, strings.ToLower(blacklisted)) {
				skip = true
				break
			}
		}
		
		if skip {
			continue
		}

		// 检查仓库年龄
		if s.config.Scanner.DateRangeDays > 0 {
			cutoffDate := time.Now().AddDate(0, 0, -s.config.Scanner.DateRangeDays)
			if pushedAt, err := time.Parse("2006-01-02T15:04:05Z", item.Repository.PushedAt); err == nil {
				if pushedAt.Before(cutoffDate) {
					continue
				}
			}
		}

		filtered = append(filtered, item)
	}

	return filtered
}

// extractKeys 从内容中提取API密钥
func (s *Scanner) extractKeys(content string) []string {
	// Gemini API密钥正则表达式
	pattern := `AIzaSy[A-Za-z0-9\-_]{33}`
	regex := regexp.MustCompile(pattern)
	
	matches := regex.FindAllString(content, -1)
	
	// 去重和过滤占位符
	keySet := make(map[string]bool)
	var keys []string
	
	for _, match := range matches {
		if keySet[match] {
			continue
		}
		
		// 过滤明显的占位符
		if s.isPlaceholderKey(match, content) {
			continue
		}
		
		keySet[match] = true
		keys = append(keys, match)
	}
	
	return keys
}

// isPlaceholderKey 检查是否为占位符密钥
func (s *Scanner) isPlaceholderKey(key, content string) bool {
	keyIndex := strings.Index(content, key)
	if keyIndex == -1 {
		return false
	}
	
	// 检查密钥周围的上下文
	start := keyIndex - 50
	if start < 0 {
		start = 0
	}
	
	end := keyIndex + len(key) + 50
	if end > len(content) {
		end = len(content)
	}
	
	context := strings.ToUpper(content[start:end])
	
	// 检查占位符关键词
	placeholderKeywords := []string{
		"YOUR_", "EXAMPLE", "PLACEHOLDER", "REPLACE", "...", 
		"TODO", "FIXME", "XXX", "SAMPLE",
	}
	
	for _, keyword := range placeholderKeywords {
		if strings.Contains(context, keyword) {
			return true
		}
	}
	
	return false
}

// StartContinuousScanning 开始持续扫描
func (s *Scanner) StartContinuousScanning(ctx context.Context) error {
	if !s.startScanning() {
		return fmt.Errorf("scanner is already running")
	}
	defer s.stopScanning()

	// 加载查询列表
	queries, err := s.loadQueries()
	if err != nil {
		return fmt.Errorf("failed to load queries: %w", err)
	}

	logger.Infof("Starting continuous scanning with %d queries", len(queries))

	for {
		select {
		case <-ctx.Done():
			logger.Info("Continuous scanning cancelled")
			return ctx.Err()
		case <-s.stopCh:
			logger.Info("Continuous scanning stopped")
			return nil
		default:
		}

		// 执行一轮扫描
		if err := s.ScanWithQueries(ctx, queries); err != nil {
			logger.Errorf("Scan round failed: %v", err)
		}

		// 等待下一轮
		logger.Infof("Waiting %v before next scan round", s.config.Scanner.ScanInterval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopCh:
			return nil
		case <-time.After(s.config.Scanner.ScanInterval):
		}
	}
}

// loadQueries 加载查询列表
func (s *Scanner) loadQueries() ([]string, error) {
	file, err := os.Open(s.config.Scanner.QueryFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open query file: %w", err)
	}
	defer file.Close()

	var queries []string
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		queries = append(queries, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read query file: %w", err)
	}

	return queries, nil
}

// Stop 停止扫描
func (s *Scanner) Stop() {
	close(s.stopCh)
}

// GetStats 获取扫描统计
func (s *Scanner) GetStats() *ScanStats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()
	
	// 返回副本
	stats := *s.stats
	return &stats
}

// IsScanning 检查是否正在扫描
func (s *Scanner) IsScanning() bool {
	s.scanMu.RLock()
	defer s.scanMu.RUnlock()
	return s.isScanning
}

// startScanning 开始扫描
func (s *Scanner) startScanning() bool {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	
	if s.isScanning {
		return false
	}
	
	s.isScanning = true
	s.updateStats(func(stats *ScanStats) {
		stats.IsActive = true
		stats.StartTime = time.Now()
	})
	
	return true
}

// stopScanning 停止扫描
func (s *Scanner) stopScanning() {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	
	s.isScanning = false
	s.updateStats(func(stats *ScanStats) {
		stats.IsActive = false
	})
}

// updateStats 更新统计信息
func (s *Scanner) updateStats(fn func(*ScanStats)) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	fn(s.stats)
}

// GetTokenStates 获取GitHub token状态信息
func (s *Scanner) GetTokenStates() map[string]interface{} {
	tokenInfo := s.github.GetRateLimitInfo()
	result := make(map[string]interface{})
	
	for token, state := range tokenInfo {
		// 只返回非敏感信息
		maskedToken := token[:8] + "***" + token[len(token)-4:]
		result[maskedToken] = map[string]interface{}{
			"remaining":     state.Remaining,
			"reset_time":    state.ResetTime,
			"cooldown_end":  state.CooldownEnd,
			"last_used":     state.LastUsed,
			"success_rate":  state.SuccessRate,
			"request_count": state.RequestCount,
		}
	}
	
	return result
}