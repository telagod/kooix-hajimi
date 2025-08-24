package scanner

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// URLDeduplicator 智能URL去重器
type URLDeduplicator struct {
	seen     map[string]bool
	urlCache map[string]URLInfo
	mutex    sync.RWMutex
}

// URLInfo URL信息结构
type URLInfo struct {
	URL        string
	Repository string
	Path       string
	Hash       string
	Priority   int
}

// NewURLDeduplicator 创建新的URL去重器
func NewURLDeduplicator() *URLDeduplicator {
	return &URLDeduplicator{
		seen:     make(map[string]bool),
		urlCache: make(map[string]URLInfo),
	}
}

// AddURL 添加URL并检查是否重复
func (d *URLDeduplicator) AddURL(rawURL, repository, path string, priority int) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 规范化URL
	normalizedURL := d.normalizeURL(rawURL)
	
	// 生成内容哈希
	contentHash := d.generateContentHash(repository, path)
	
	// 检查是否已存在
	if d.seen[normalizedURL] || d.seen[contentHash] {
		return false
	}
	
	// 标记为已见
	d.seen[normalizedURL] = true
	d.seen[contentHash] = true
	
	// 存储URL信息
	d.urlCache[normalizedURL] = URLInfo{
		URL:        normalizedURL,
		Repository: repository,
		Path:       path,
		Hash:       contentHash,
		Priority:   priority,
	}
	
	return true
}

// normalizeURL 规范化URL格式
func (d *URLDeduplicator) normalizeURL(rawURL string) string {
	// 解析URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	
	// 移除查询参数和片段
	u.RawQuery = ""
	u.Fragment = ""
	
	// 规范化路径
	path := strings.TrimSuffix(u.Path, "/")
	u.Path = path
	
	return u.String()
}

// generateContentHash 生成内容哈希
func (d *URLDeduplicator) generateContentHash(repository, path string) string {
	content := fmt.Sprintf("%s:%s", repository, path)
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)[:16]
}

// GetUniqueURLs 获取去重后的URL列表，按优先级排序
func (d *URLDeduplicator) GetUniqueURLs() []URLInfo {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	urls := make([]URLInfo, 0, len(d.urlCache))
	for _, info := range d.urlCache {
		urls = append(urls, info)
	}
	
	// 按优先级排序（优先级越高，越靠前）
	sort.Slice(urls, func(i, j int) bool {
		return urls[i].Priority > urls[j].Priority
	})
	
	return urls
}

// GetStats 获取去重统计信息
func (d *URLDeduplicator) GetStats() map[string]int {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	return map[string]int{
		"total_seen":   len(d.seen),
		"unique_urls":  len(d.urlCache),
		"duplicates":   len(d.seen) - len(d.urlCache),
	}
}

// Clear 清空去重器
func (d *URLDeduplicator) Clear() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.seen = make(map[string]bool)
	d.urlCache = make(map[string]URLInfo)
}