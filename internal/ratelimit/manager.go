package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
	
	"kooix-hajimi/pkg/logger"
)

// TokenState 代表单个Token的状态
type TokenState struct {
	Token          string
	Remaining      int
	ResetTime      time.Time
	CooldownEnd    time.Time
	LastUsed       time.Time
	SuccessRate    float64
	RequestCount   int64
	SuccessCount   int64
	limiter        *rate.Limiter
}

// Manager 智能限流管理器
type Manager struct {
	tokenStates    map[string]*TokenState
	mu             sync.RWMutex
	adaptiveMode   bool
	successThresh  float64
	backoffMult    float64
	baseLimiter    *rate.Limiter
}

// Config 限流配置
type Config struct {
	RequestsPerMinute int
	BurstSize         int
	CooldownDuration  time.Duration
	AdaptiveEnabled   bool
	SuccessThreshold  float64
	BackoffMultiplier float64
}

// New 创建新的限流管理器
func New(cfg Config) *Manager {
	return &Manager{
		tokenStates:   make(map[string]*TokenState),
		adaptiveMode:  cfg.AdaptiveEnabled,
		successThresh: cfg.SuccessThreshold,
		backoffMult:   cfg.BackoffMultiplier,
		baseLimiter:   rate.NewLimiter(rate.Limit(cfg.RequestsPerMinute)/60, cfg.BurstSize),
	}
}

// GetBestToken 获取最佳可用Token
func (m *Manager) GetBestToken(tokens []string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(tokens) == 0 {
		return "", fmt.Errorf("no tokens available")
	}

	now := time.Now()
	var bestToken string
	var bestScore float64 = -1

	for _, token := range tokens {
		state, exists := m.tokenStates[token]
		if !exists {
			// 新token，直接返回
			return token, nil
		}

		// 检查是否在冷却期
		if now.Before(state.CooldownEnd) {
			continue
		}

		// 检查是否到了重置时间
		if now.After(state.ResetTime) && state.Remaining <= 0 {
			state.Remaining = 5000 // GitHub默认限制
		}

		// 如果没有剩余请求数，跳过
		if state.Remaining <= 0 {
			continue
		}

		// 计算token评分
		score := m.calculateTokenScore(state, now)
		if score > bestScore {
			bestScore = score
			bestToken = token
		}
	}

	if bestToken == "" {
		return "", fmt.Errorf("no available tokens (all in cooldown or exhausted)")
	}

	return bestToken, nil
}

// calculateTokenScore 计算token评分
func (m *Manager) calculateTokenScore(state *TokenState, now time.Time) float64 {
	score := float64(state.Remaining) / 5000.0 // 基于剩余请求数

	// 考虑成功率
	if state.RequestCount > 10 {
		score *= state.SuccessRate
	}

	// 考虑最后使用时间，越久没用分数越高
	timeSinceLastUse := now.Sub(state.LastUsed).Minutes()
	score *= (1.0 + timeSinceLastUse/60.0) // 每小时加权+1

	return score
}

// UpdateTokenState 更新Token状态
func (m *Manager) UpdateTokenState(token string, headers http.Header, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.tokenStates[token]
	if !exists {
		state = &TokenState{
			Token:        token,
			Remaining:    5000,
			limiter:      rate.NewLimiter(rate.Limit(30)/60, 5), // 默认30/分钟
		}
		m.tokenStates[token] = state
	}

	now := time.Now()
	state.LastUsed = now
	state.RequestCount++

	if success {
		state.SuccessCount++
	}

	// 更新成功率
	if state.RequestCount > 0 {
		state.SuccessRate = float64(state.SuccessCount) / float64(state.RequestCount)
	}

	// 从响应头更新限流信息
	if remaining := headers.Get("X-RateLimit-Remaining"); remaining != "" {
		if r, err := strconv.Atoi(remaining); err == nil {
			state.Remaining = r
		}
	}

	if reset := headers.Get("X-RateLimit-Reset"); reset != "" {
		if r, err := strconv.ParseInt(reset, 10, 64); err == nil {
			state.ResetTime = time.Unix(r, 0)
		}
	}

	// 自适应调整
	if m.adaptiveMode {
		m.adjustTokenLimiter(state)
	}

	logger.Debugf("Token %s: remaining=%d, success_rate=%.2f", 
		token[:8], state.Remaining, state.SuccessRate)
}

// HandleRateLimit 处理限流情况
func (m *Manager) HandleRateLimit(token string, statusCode int, headers http.Header) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.tokenStates[token]
	if !exists {
		state = &TokenState{
			Token: token,
		}
		m.tokenStates[token] = state
	}

	var cooldownDuration time.Duration

	switch statusCode {
	case 403, 429:
		// 从响应头获取重置时间
		if reset := headers.Get("X-RateLimit-Reset"); reset != "" {
			if r, err := strconv.ParseInt(reset, 10, 64); err == nil {
				resetTime := time.Unix(r, 0)
				cooldownDuration = time.Until(resetTime)
			}
		}

		// 如果无法从响应头获取，使用默认值
		if cooldownDuration <= 0 {
			cooldownDuration = 5 * time.Minute
		}

		state.CooldownEnd = time.Now().Add(cooldownDuration)
		state.Remaining = 0

		logger.Warnf("Token %s rate limited, cooldown until %s", 
			token[:8], state.CooldownEnd.Format("15:04:05"))
	}

	return cooldownDuration
}

// adjustTokenLimiter 自适应调整token限流器
func (m *Manager) adjustTokenLimiter(state *TokenState) {
	if state.RequestCount < 20 {
		return // 数据不足，不调整
	}

	// 根据成功率调整限流速度
	if state.SuccessRate < m.successThresh {
		// 成功率低，降低请求频率
		currentRate := float64(state.limiter.Limit())
		newRate := currentRate / m.backoffMult
		if newRate < 10.0/60.0 { // 最低10/分钟
			newRate = 10.0 / 60.0
		}
		state.limiter.SetLimit(rate.Limit(newRate))
		
		logger.Debugf("Token %s: reduced rate to %.2f/min due to low success rate", 
			state.Token[:8], newRate*60)
	} else if state.SuccessRate > 0.95 {
		// 成功率高，可以提高请求频率
		currentRate := float64(state.limiter.Limit())
		newRate := currentRate * 1.2
		if newRate > 50.0/60.0 { // 最高50/分钟
			newRate = 50.0 / 60.0
		}
		state.limiter.SetLimit(rate.Limit(newRate))
		
		logger.Debugf("Token %s: increased rate to %.2f/min due to high success rate", 
			state.Token[:8], newRate*60)
	}
}

// Wait 等待直到可以发送请求
func (m *Manager) Wait(ctx context.Context, token string) error {
	m.mu.RLock()
	state, exists := m.tokenStates[token]
	m.mu.RUnlock()

	if exists && state.limiter != nil {
		return state.limiter.Wait(ctx)
	}

	return m.baseLimiter.Wait(ctx)
}

// GetTokenStates 获取所有token状态（用于监控）
func (m *Manager) GetTokenStates() map[string]*TokenState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*TokenState)
	for token, state := range m.tokenStates {
		// 复制状态，避免并发访问问题
		result[token] = &TokenState{
			Token:        state.Token,
			Remaining:    state.Remaining,
			ResetTime:    state.ResetTime,
			CooldownEnd:  state.CooldownEnd,
			LastUsed:     state.LastUsed,
			SuccessRate:  state.SuccessRate,
			RequestCount: state.RequestCount,
			SuccessCount: state.SuccessCount,
		}
	}

	return result
}