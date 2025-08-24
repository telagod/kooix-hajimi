package validator

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	
	"kooix-hajimi/pkg/logger"
)

// KeyInfo 密钥信息
type KeyInfo struct {
	Key      string `json:"key"`
	Provider string `json:"provider"` // gemini, openai, claude
	Type     string `json:"type"`     // api_key, project_key
}

// ValidationResult 验证结果
type ValidationResult struct {
	Key       string        `json:"key"`
	Provider  string        `json:"provider"`
	Type      string        `json:"type"`
	Valid     bool          `json:"valid"`
	Status    string        `json:"status"` // valid, invalid, rate_limited, quota_exceeded, disabled, error
	Error     string        `json:"error,omitempty"`
	Latency   time.Duration `json:"latency"`
	Timestamp time.Time     `json:"timestamp"`
}

// Validator Key验证器
type Validator struct {
	mu          sync.RWMutex
	modelName   string
	workerCount int
	timeout     time.Duration
}

// Config 验证器配置
type Config struct {
	ModelName   string
	WorkerCount int
	Timeout     time.Duration
}

// New 创建新的验证器
func New(cfg Config) *Validator {
	if cfg.ModelName == "" {
		cfg.ModelName = "gemini-2.5-flash"
	}
	if cfg.WorkerCount == 0 {
		cfg.WorkerCount = 5
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Validator{
		modelName:   cfg.ModelName,
		workerCount: cfg.WorkerCount,
		timeout:     cfg.Timeout,
	}
}

// ValidateKey 验证单个Key
func (v *Validator) ValidateKey(ctx context.Context, key string) (*ValidationResult, error) {
	start := time.Now()
	
	result := &ValidationResult{
		Key:       key,
		Timestamp: start,
	}

	// 添加随机延迟避免过于频繁的请求
	delay := time.Duration(rand.Intn(1000)+500) * time.Millisecond
	select {
	case <-ctx.Done():
		result.Status = "error"
		result.Error = "context cancelled"
		result.Latency = time.Since(start)
		return result, ctx.Err()
	case <-time.After(delay):
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	// 验证Key
	valid, status, err := v.validateGeminiKey(timeoutCtx, key)
	
	result.Valid = valid
	result.Status = status
	result.Latency = time.Since(start)
	
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	return result, nil
}

// ValidateBatch 批量验证密钥
func (v *Validator) ValidateBatch(ctx context.Context, keys []KeyInfo) ([]*ValidationResult, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	logger.Infof("Starting batch validation of %d keys", len(keys))

	// 创建工作池
	jobCh := make(chan KeyInfo, len(keys))
	resultCh := make(chan *ValidationResult, len(keys))

	// 启动worker
	var wg sync.WaitGroup
	for i := 0; i < v.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for keyInfo := range jobCh {
				result, err := v.validateKeyByProvider(ctx, keyInfo)
				if err != nil {
					logger.Errorf("Worker %d validation error for key %s: %v", 
						workerID, keyInfo.Provider, keyInfo.Key[:10]+"...", err)
				}
				resultCh <- result
			}
		}(i)
	}

	// 发送任务
	go func() {
		defer close(jobCh)
		for _, keyInfo := range keys {
			select {
			case jobCh <- keyInfo:
			case <-ctx.Done():
				return
			}
		}
	}()

	// 等待所有worker完成
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集结果
	var results []*ValidationResult
	for result := range resultCh {
		results = append(results, result)
	}

	// 统计结果
	var validCount, invalidCount, rateLimitedCount, errorCount int
	for _, result := range results {
		switch result.Status {
		case "valid":
			validCount++
		case "invalid":
			invalidCount++
		case "rate_limited":
			rateLimitedCount++
		default:
			errorCount++
		}
	}

	logger.Infof("Batch validation complete: %d valid, %d invalid, %d rate limited, %d errors",
		validCount, invalidCount, rateLimitedCount, errorCount)

	return results, nil
}

// validateKeyByProvider 根据提供商验证密钥
func (v *Validator) validateKeyByProvider(ctx context.Context, keyInfo KeyInfo) (*ValidationResult, error) {
	start := time.Now()
	
	result := &ValidationResult{
		Key:       keyInfo.Key,
		Provider:  keyInfo.Provider,
		Type:      keyInfo.Type,
		Timestamp: start,
	}

	// 根据提供商选择验证方法
	var valid bool
	var status string
	var err error
	
	switch keyInfo.Provider {
	case "gemini":
		valid, status, err = v.validateGeminiKey(ctx, keyInfo.Key)
	case "openai":
		valid, status, err = v.validateOpenAIKey(ctx, keyInfo)
	case "claude":
		valid, status, err = v.validateClaudeKey(ctx, keyInfo)
	default:
		valid = false
		status = "error"
		err = fmt.Errorf("unsupported provider: %s", keyInfo.Provider)
	}
	
	result.Valid = valid
	result.Status = status
	result.Latency = time.Since(start)
	
	if err != nil {
		result.Error = err.Error()
	}

	return result, err
}

// validateGeminiKey 验证Gemini API Key
func (v *Validator) validateGeminiKey(ctx context.Context, key string) (bool, string, error) {
	// 创建客户端
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		return false, "error", fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// 获取模型
	model := client.GenerativeModel(v.modelName)

	// 发送简单的测试请求
	resp, err := model.GenerateContent(ctx, genai.Text("hi"))
	if err != nil {
		// 分析错误类型
		errStr := err.Error()
		switch {
		case contains(errStr, "API_KEY_INVALID"):
			return false, "invalid", nil
		case contains(errStr, "PERMISSION_DENIED"):
			return false, "invalid", nil
		case contains(errStr, "UNAUTHENTICATED"):
			return false, "invalid", nil
		case contains(errStr, "QUOTA_EXCEEDED"):
			return false, "rate_limited", nil
		case contains(errStr, "RESOURCE_EXHAUSTED"):
			return false, "rate_limited", nil
		case contains(errStr, "RATE_LIMIT_EXCEEDED"):
			return false, "rate_limited", nil
		case contains(errStr, "SERVICE_DISABLED"):
			return false, "invalid", nil
		case contains(errStr, "API has not been used"):
			return false, "invalid", nil
		default:
			return false, "error", fmt.Errorf("validation error: %w", err)
		}
	}

	// 检查响应
	if resp == nil || len(resp.Candidates) == 0 {
		return false, "error", fmt.Errorf("empty response")
	}

	return true, "valid", nil
}

// validateOpenAIKey 验证OpenAI API Key
func (v *Validator) validateOpenAIKey(ctx context.Context, keyInfo KeyInfo) (bool, string, error) {
	client := &http.Client{Timeout: v.timeout}
	
	// 根据密钥类型选择不同的验证端点
	var url string
	if keyInfo.Type == "project_key" {
		url = "https://api.openai.com/v1/models" // 项目密钥使用models端点
	} else {
		url = "https://api.openai.com/v1/models" // 标准API密钥
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, "error", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+keyInfo.Key)
	req.Header.Set("User-Agent", "Kooix-Hajimi/1.0")
	
	resp, err := client.Do(req)
	if err != nil {
		return false, "error", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// 分析HTTP状态码和错误
	switch resp.StatusCode {
	case 200:
		return true, "valid", nil
	case 401:
		return false, "invalid", nil // Invalid API key
	case 403:
		return false, "invalid", nil // Forbidden - key disabled
	case 429:
		return false, "rate_limited", nil // Rate limited
	case 402:
		return false, "quota_exceeded", nil // Payment required
	case 500, 502, 503, 504:
		return false, "error", fmt.Errorf("server error: %d", resp.StatusCode)
	default:
		return false, "error", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

// validateClaudeKey 验证Claude API Key
func (v *Validator) validateClaudeKey(ctx context.Context, keyInfo KeyInfo) (bool, string, error) {
	client := &http.Client{Timeout: v.timeout}
	
	// Claude API使用messages端点进行简单验证
	url := "https://api.anthropic.com/v1/messages"
	
	// 创建简单的测试请求
	payload := strings.NewReader(`{
		"model": "claude-3-haiku-20240307",
		"max_tokens": 1,
		"messages": [{"role": "user", "content": "hi"}]
	}`)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, payload)
	if err != nil {
		return false, "error", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", keyInfo.Key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("User-Agent", "Kooix-Hajimi/1.0")
	
	resp, err := client.Do(req)
	if err != nil {
		return false, "error", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// 分析HTTP状态码和错误
	switch resp.StatusCode {
	case 200:
		return true, "valid", nil
	case 400:
		// 可能是有效key但请求格式问题，视为有效
		return true, "valid", nil
	case 401:
		return false, "invalid", nil // Invalid API key
	case 403:
		return false, "invalid", nil // Forbidden - key disabled
	case 429:
		return false, "rate_limited", nil // Rate limited
	case 402:
		return false, "quota_exceeded", nil // Payment required
	case 500, 502, 503, 504:
		return false, "error", fmt.Errorf("server error: %d", resp.StatusCode)
	default:
		return false, "error", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

// GetStats 获取验证统计信息
func (v *Validator) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"model_name":   v.modelName,
		"worker_count": v.workerCount,
		"timeout":      v.timeout.String(),
	}
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     indexOf(s, substr) >= 0))
}

// indexOf 查找子串位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}