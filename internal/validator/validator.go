package validator

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	
	"kooix-hajimi/pkg/logger"
)

// ValidationResult 验证结果
type ValidationResult struct {
	Key       string        `json:"key"`
	Valid     bool          `json:"valid"`
	Status    string        `json:"status"` // valid, invalid, rate_limited, error
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

// ValidateBatch 批量验证Keys
func (v *Validator) ValidateBatch(ctx context.Context, keys []string) ([]*ValidationResult, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	logger.Infof("Starting batch validation of %d keys", len(keys))

	// 创建工作池
	jobCh := make(chan string, len(keys))
	resultCh := make(chan *ValidationResult, len(keys))

	// 启动worker
	var wg sync.WaitGroup
	for i := 0; i < v.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for key := range jobCh {
				result, err := v.ValidateKey(ctx, key)
				if err != nil {
					logger.Errorf("Worker %d validation error for key %s: %v", 
						workerID, key[:10]+"...", err)
				}
				resultCh <- result
			}
		}(i)
	}

	// 发送任务
	go func() {
		defer close(jobCh)
		for _, key := range keys {
			select {
			case jobCh <- key:
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