package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	
	"kooix-hajimi/internal/config"
	"kooix-hajimi/internal/ratelimit"
	"kooix-hajimi/pkg/logger"
)

// SearchResult GitHub搜索结果
type SearchResult struct {
	TotalCount        int          `json:"total_count"`
	IncompleteResults bool         `json:"incomplete_results"`
	Items             []SearchItem `json:"items"`
}

// SearchItem 搜索结果项
type SearchItem struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	SHA        string     `json:"sha"`
	URL        string     `json:"url"`
	HTMLURL    string     `json:"html_url"`
	Repository Repository `json:"repository"`
}

// Repository 仓库信息
type Repository struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	PushedAt string `json:"pushed_at"`
	Private  bool   `json:"private"`
}

// FileContent 文件内容
type FileContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
}

// Client GitHub客户端
type Client struct {
	httpClient   *resty.Client
	tokens       []string
	rateLimiter  *ratelimit.Manager
	currentToken int32
	config       config.GitHubConfig
}

// New 创建新的GitHub客户端
func New(cfg config.GitHubConfig, rateLimiter *ratelimit.Manager) *Client {
	client := resty.New().
		SetTimeout(cfg.Timeout).
		SetRetryCount(cfg.MaxRetries).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		SetHeader("User-Agent", cfg.UserAgent).
		SetHeader("Accept", "application/vnd.github.v3+json")

	return &Client{
		httpClient:  client,
		tokens:      cfg.Tokens,
		rateLimiter: rateLimiter,
		config:      cfg,
	}
}

// SearchCode 搜索代码
func (c *Client) SearchCode(ctx context.Context, query string) (*SearchResult, error) {
	const maxPages = 10
	const perPage = 100

	var allItems []SearchItem
	totalCount := 0
	expectedTotal := 0

	for page := 1; page <= maxPages; page++ {
		pageResult, err := c.searchCodePage(ctx, query, page, perPage)
		if err != nil {
			if page == 1 {
				return nil, fmt.Errorf("first page search failed: %w", err)
			}
			logger.Warnf("Search page %d failed for query '%s': %v", page, query, err)
			break
		}

		if page == 1 {
			totalCount = pageResult.TotalCount
			expectedTotal = min(totalCount, 1000) // GitHub限制最多1000个结果
		}

		if len(pageResult.Items) == 0 {
			break
		}

		allItems = append(allItems, pageResult.Items...)

		// 如果获取到的结果已经达到预期总数，停止搜索
		if len(allItems) >= expectedTotal {
			break
		}

		// 页面间延迟
		if page < maxPages {
			delay := time.Duration(rand.Intn(1000)+500) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	logger.Infof("GitHub search complete for query '%s': %d/%d items retrieved",
		query, len(allItems), expectedTotal)

	return &SearchResult{
		TotalCount:        totalCount,
		IncompleteResults: len(allItems) < expectedTotal,
		Items:             allItems,
	}, nil
}

// searchCodePage 搜索单页结果
func (c *Client) searchCodePage(ctx context.Context, query string, page, perPage int) (*SearchResult, error) {
	token, err := c.rateLimiter.GetBestToken(c.tokens)
	if err != nil {
		return nil, fmt.Errorf("no available tokens: %w", err)
	}

	// 等待限流器允许
	if err := c.rateLimiter.Wait(ctx, token); err != nil {
		return nil, fmt.Errorf("rate limiter wait failed: %w", err)
	}

	searchURL := "https://api.github.com/search/code"
	params := url.Values{
		"q":        []string{query},
		"page":     []string{strconv.Itoa(page)},
		"per_page": []string{strconv.Itoa(perPage)},
	}

	var result SearchResult
	resp, err := c.httpClient.R().
		SetAuthToken(token).
		SetQueryParamsFromValues(params).
		SetResult(&result).
		SetContext(ctx).
		Get(searchURL)

	success := err == nil && resp.StatusCode() == 200
	c.rateLimiter.UpdateTokenState(token, resp.Header(), success)

	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	if resp.StatusCode() == 403 || resp.StatusCode() == 429 {
		cooldown := c.rateLimiter.HandleRateLimit(token, resp.StatusCode(), resp.Header())
		return nil, fmt.Errorf("rate limited (cooldown: %v)", cooldown)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", 
			resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// GetFileContent 获取文件内容
func (c *Client) GetFileContent(ctx context.Context, item *SearchItem) ([]byte, error) {
	token, err := c.rateLimiter.GetBestToken(c.tokens)
	if err != nil {
		return nil, fmt.Errorf("no available tokens: %w", err)
	}

	if err := c.rateLimiter.Wait(ctx, token); err != nil {
		return nil, fmt.Errorf("rate limiter wait failed: %w", err)
	}

	// 获取文件元数据
	metadataURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", 
		item.Repository.FullName, item.Path)

	var fileContent FileContent
	resp, err := c.httpClient.R().
		SetAuthToken(token).
		SetResult(&fileContent).
		SetContext(ctx).
		Get(metadataURL)

	success := err == nil && resp.StatusCode() == 200
	c.rateLimiter.UpdateTokenState(token, resp.Header(), success)

	if err != nil {
		return nil, fmt.Errorf("metadata request failed: %w", err)
	}

	if resp.StatusCode() == 403 || resp.StatusCode() == 429 {
		cooldown := c.rateLimiter.HandleRateLimit(token, resp.StatusCode(), resp.Header())
		return nil, fmt.Errorf("rate limited (cooldown: %v)", cooldown)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	// 如果有base64编码的内容，直接解码
	if fileContent.Encoding == "base64" && fileContent.Content != "" {
		content := strings.ReplaceAll(fileContent.Content, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err == nil {
			return decoded, nil
		}
		logger.Warnf("Failed to decode base64 content: %v, falling back to download_url", err)
	}

	// 使用download_url获取内容
	if fileContent.DownloadURL == "" {
		return nil, fmt.Errorf("no download URL available")
	}

	contentResp, err := c.httpClient.R().
		SetAuthToken(token).
		SetContext(ctx).
		Get(fileContent.DownloadURL)

	if err != nil {
		return nil, fmt.Errorf("content download failed: %w", err)
	}

	if contentResp.StatusCode() != 200 {
		return nil, fmt.Errorf("download failed with status: %d", contentResp.StatusCode())
	}

	return contentResp.Body(), nil
}

// GetRateLimitInfo 获取当前限流信息
func (c *Client) GetRateLimitInfo() map[string]*ratelimit.TokenState {
	return c.rateLimiter.GetTokenStates()
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}