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

// KeyTier key层级类型
type KeyTier string

const (
	TierUnknown KeyTier = "unknown"
	TierFree    KeyTier = "free"  
	TierPaid    KeyTier = "paid"
)

// TierDetectionResult 层级检测结果
type TierDetectionResult struct {
	Key        string        `json:"key"`
	Tier       KeyTier       `json:"tier"`
	Confidence float64       `json:"confidence"` // 0.0-1.0 置信度
	Method     string        `json:"method"`     // 检测方法
	Evidence   []string      `json:"evidence"`   // 证据列表
	Latency    time.Duration `json:"latency"`
	Timestamp  time.Time     `json:"timestamp"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Key       string              `json:"key"`
	Provider  string              `json:"provider"`
	Type      string              `json:"type"`
	Valid     bool                `json:"valid"`
	Status    string              `json:"status"` // valid, invalid, rate_limited, quota_exceeded, disabled, error
	Tier      *TierDetectionResult `json:"tier,omitempty"` // 可选的层级信息
	Error     string              `json:"error,omitempty"`
	Latency   time.Duration       `json:"latency"`
	Timestamp time.Time           `json:"timestamp"`
}

// Validator Key验证器
type Validator struct {
	mu                 sync.RWMutex
	modelName          string
	tierDetectionModel string
	workerCount        int
	timeout            time.Duration
	enableTierDetection bool
}

// Config 验证器配置
type Config struct {
	ModelName           string
	TierDetectionModel  string
	WorkerCount         int
	Timeout             time.Duration
	EnableTierDetection bool
}

// New 创建新的验证器
func New(cfg Config) *Validator {
	if cfg.ModelName == "" {
		cfg.ModelName = "gemini-2.5-flash"
	}
	if cfg.TierDetectionModel == "" {
		cfg.TierDetectionModel = "gemini-2.5-flash"
	}
	if cfg.WorkerCount == 0 {
		cfg.WorkerCount = 5
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Validator{
		modelName:          cfg.ModelName,
		tierDetectionModel: cfg.TierDetectionModel,
		workerCount:        cfg.WorkerCount,
		timeout:           cfg.Timeout,
		enableTierDetection: cfg.EnableTierDetection,
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

	// 如果是有效的Gemini key且启用了层级检测，进行层级检测
	if valid && keyInfo.Provider == "gemini" && v.enableTierDetection {
		tierResult, tierErr := v.DetectGeminiKeyTier(ctx, keyInfo.Key)
		if tierErr == nil {
			result.Tier = tierResult
			logger.Infof("Detected tier for key %s: %s (confidence: %.2f)", 
				keyInfo.Key[:10]+"...", tierResult.Tier, tierResult.Confidence)
		} else {
			logger.Warnf("Failed to detect tier for key %s: %v", keyInfo.Key[:10]+"...", tierErr)
		}
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

// DetectGeminiKeyTier 检测Gemini Key层级
func (v *Validator) DetectGeminiKeyTier(ctx context.Context, key string) (*TierDetectionResult, error) {
	start := time.Now()
	
	result := &TierDetectionResult{
		Key:        key,
		Tier:       TierUnknown,
		Confidence: 0.0,
		Timestamp:  start,
		Evidence:   make([]string, 0),
	}

	// 创建客户端
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		result.Latency = time.Since(start)
		return result, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// 方法1: Rate Limit探测
	tierFromRate, evidenceRate := v.detectTierByRateLimit(ctx, client)
	result.Evidence = append(result.Evidence, evidenceRate...)

	// 方法2: 模型访问测试（尝试付费模型特性）
	tierFromModel, evidenceModel := v.detectTierByModelAccess(ctx, client)
	result.Evidence = append(result.Evidence, evidenceModel...)

	// 综合判断
	result.Tier, result.Confidence = v.combineTierResults(tierFromRate, tierFromModel)
	result.Method = "rate_limit+model_access"
	result.Latency = time.Since(start)

	return result, nil
}

// detectTierByRateLimit 通过Rate Limit检测层级
func (v *Validator) detectTierByRateLimit(ctx context.Context, client *genai.Client) (KeyTier, []string) {
	evidence := make([]string, 0)
	model := client.GenerativeModel(v.tierDetectionModel)
	
	// 快速连续请求测试
	const testRequests = 3
	successCount := 0
	rateLimitCount := 0
	
	for i := 0; i < testRequests; i++ {
		testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := model.GenerateContent(testCtx, genai.Text("test"))
		cancel()
		
		if err != nil {
			errStr := err.Error()
			if contains(errStr, "RATE_LIMIT") || contains(errStr, "RESOURCE_EXHAUSTED") || contains(errStr, "QUOTA_EXCEEDED") {
				rateLimitCount++
				evidence = append(evidence, fmt.Sprintf("rate_limit_hit_request_%d", i+1))
			}
		} else {
			successCount++
		}
		
		// 短间隔延迟（模拟快速请求）
		if i < testRequests-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	// 判断逻辑：免费账户更容易触发rate limit
	if rateLimitCount >= 2 {
		evidence = append(evidence, fmt.Sprintf("high_rate_limit_ratio_%d/%d", rateLimitCount, testRequests))
		return TierFree, evidence
	} else if successCount >= 2 {
		evidence = append(evidence, fmt.Sprintf("low_rate_limit_ratio_%d/%d", rateLimitCount, testRequests))
		return TierPaid, evidence
	}
	
	return TierUnknown, evidence
}

// detectTierByModelAccess 通过模型访问检测层级
func (v *Validator) detectTierByModelAccess(ctx context.Context, client *genai.Client) (KeyTier, []string) {
	evidence := make([]string, 0)
	
	// 测试高级模型特性（如长上下文）
	model := client.GenerativeModel(v.tierDetectionModel)
	
	// 设置较大的上下文长度来测试
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	// 使用相对较长的输入测试上下文处理能力
	longInput := strings.Repeat("This is a test input to check context handling capabilities. ", 50)
	resp, err := model.GenerateContent(testCtx, genai.Text(longInput))
	
	if err != nil {
		errStr := err.Error()
		if contains(errStr, "context") || contains(errStr, "token") || contains(errStr, "length") {
			evidence = append(evidence, "context_limit_restriction")
			return TierFree, evidence
		} else if contains(errStr, "RATE_LIMIT") || contains(errStr, "QUOTA_EXCEEDED") {
			evidence = append(evidence, "rate_limit_on_complex_request")
			return TierFree, evidence
		}
	} else if resp != nil {
		evidence = append(evidence, "handled_complex_request_successfully")
		return TierPaid, evidence
	}
	
	return TierUnknown, evidence
}

// combineTierResults 综合多种检测结果
func (v *Validator) combineTierResults(tierFromRate, tierFromModel KeyTier) (KeyTier, float64) {
	// 权重评分
	rateScore := 0.0
	modelScore := 0.0
	
	switch tierFromRate {
	case TierFree:
		rateScore = -1.0
	case TierPaid:
		rateScore = 1.0
	}
	
	switch tierFromModel {
	case TierFree:
		modelScore = -1.0
	case TierPaid:
		modelScore = 1.0
	}
	
	// 加权平均（rate limit检测权重更高）
	finalScore := (rateScore * 0.7) + (modelScore * 0.3)
	confidence := (abs(finalScore) + abs(rateScore) + abs(modelScore)) / 3.0
	
	if finalScore < -0.3 {
		return TierFree, confidence
	} else if finalScore > 0.3 {
		return TierPaid, confidence
	}
	
	return TierUnknown, confidence * 0.5 // 降低未知结果的置信度
}

// abs 绝对值函数
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
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
		"model_name":            v.modelName,
		"tier_detection_model":  v.tierDetectionModel,
		"worker_count":          v.workerCount,
		"timeout":               v.timeout.String(),
		"enable_tier_detection": v.enableTierDetection,
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

// ValidateKeyWithTier 验证密钥并检测层级
func (v *Validator) ValidateKeyWithTier(ctx context.Context, key string) (*ValidationResult, error) {
	// 先进行基础验证
	result, err := v.ValidateKey(ctx, key)
	if err != nil || !result.Valid {
		return result, err
	}

	// 如果是Gemini key且启用了层级检测，进行层级检测
	if result.Provider == "gemini" && v.enableTierDetection {
		tierResult, tierErr := v.DetectGeminiKeyTier(ctx, key)
		if tierErr == nil {
			result.Tier = tierResult
		}
	}

	return result, nil
}

// SetTierDetectionEnabled 设置是否启用层级检测
func (v *Validator) SetTierDetectionEnabled(enabled bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enableTierDetection = enabled
}

// SetTierDetectionModel 设置层级检测使用的模型
func (v *Validator) SetTierDetectionModel(model string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if model != "" {
		v.tierDetectionModel = model
	}
}