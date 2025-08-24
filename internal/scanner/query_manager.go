package scanner

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// QueryManager 查询管理器
type QueryManager struct {
	queries         []string
	currentIndex    int
	lastRefreshTime time.Time
	queryFilePath   string
}

// NewQueryManager 创建查询管理器
func NewQueryManager(queryFilePath string) *QueryManager {
	return &QueryManager{
		queryFilePath: queryFilePath,
	}
}

// GetOptimizedQueries 获取优化的查询列表
func (qm *QueryManager) GetOptimizedQueries() ([]string, error) {
	// 如果查询列表为空或超过1小时，重新加载
	if len(qm.queries) == 0 || time.Since(qm.lastRefreshTime) > time.Hour {
		if err := qm.loadQueries(); err != nil {
			return nil, err
		}
		qm.lastRefreshTime = time.Now()
	}
	
	return qm.queries, nil
}

// GetNextQuery 获取下一个查询
func (qm *QueryManager) GetNextQuery() (string, error) {
	queries, err := qm.GetOptimizedQueries()
	if err != nil {
		return "", err
	}
	
	if len(queries) == 0 {
		return "", fmt.Errorf("no queries available")
	}
	
	query := queries[qm.currentIndex]
	qm.currentIndex = (qm.currentIndex + 1) % len(queries)
	
	return query, nil
}

// loadQueries 从文件加载查询
func (qm *QueryManager) loadQueries() error {
	file, err := os.Open(qm.queryFilePath)
	if err != nil {
		return fmt.Errorf("failed to open query file: %w", err)
	}
	defer file.Close()

	var queries []string
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		queries = append(queries, line)
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read query file: %w", err)
	}
	
	qm.queries = queries
	return nil
}

// GetQueryStats 获取查询统计
func (qm *QueryManager) GetQueryStats() map[string]interface{} {
	return map[string]interface{}{
		"total_queries":     len(qm.queries),
		"current_index":     qm.currentIndex,
		"last_refresh_time": qm.lastRefreshTime,
		"query_file_path":   qm.queryFilePath,
	}
}