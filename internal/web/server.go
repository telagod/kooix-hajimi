package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"kooix-hajimi/internal/config"
	"kooix-hajimi/internal/github"
	"kooix-hajimi/internal/scanner"
	"kooix-hajimi/internal/storage"
	"kooix-hajimi/pkg/logger"
)

// Server Web服务器
type Server struct {
	router   *gin.Engine
	server   *http.Server
	scanner  *scanner.Scanner
	storage  storage.Storage
	config   config.WebConfig
	appConfig *config.Config // 添加完整配置引用
	upgrader websocket.Upgrader
}

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ConfigUpdateRequest 配置更新请求
type ConfigUpdateRequest struct {
	Scanner   *ScannerConfigUpdate   `json:"scanner,omitempty"`
	Validator *ValidatorConfigUpdate `json:"validator,omitempty"`
	RateLimit *RateLimitConfigUpdate `json:"rate_limit,omitempty"`
}

// ScannerConfigUpdate 扫描器配置更新
type ScannerConfigUpdate struct {
	WorkerCount   *int  `json:"worker_count,omitempty"`
	BatchSize     *int  `json:"batch_size,omitempty"`
	DateRangeDays *int  `json:"date_range_days,omitempty"`
	AutoStart     *bool `json:"auto_start,omitempty"`
}

// ValidatorConfigUpdate 验证器配置更新
type ValidatorConfigUpdate struct {
	ModelName           *string `json:"model_name,omitempty"`
	TierDetectionModel  *string `json:"tier_detection_model,omitempty"`
	WorkerCount         *int    `json:"worker_count,omitempty"`
	Timeout             *int    `json:"timeout,omitempty"` // 秒数
	EnableTierDetection *bool   `json:"enable_tier_detection,omitempty"`
}

// RateLimitConfigUpdate 限流配置更新
type RateLimitConfigUpdate struct {
	Enabled           *bool `json:"enabled,omitempty"`
	RequestsPerMinute *int  `json:"requests_per_minute,omitempty"`
	BurstSize         *int  `json:"burst_size,omitempty"`
	AdaptiveEnabled   *bool `json:"adaptive_enabled,omitempty"`
}

// New 创建Web服务器
func New(cfg *config.Config, scanner *scanner.Scanner) (*Server, error) {
	// 设置Gin模式
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// 创建存储实例
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

	server := &Server{
		router:    router,
		scanner:   scanner,
		storage:   store,
		config:    cfg.Web,
		appConfig: cfg, // 保存完整配置引用
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境需要严格控制
			},
		},
	}

	// 设置路由
	server.setupRoutes()

	// 创建HTTP服务器
	server.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Web.Host, cfg.Web.Port),
		Handler: router,
	}

	return server, nil
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// CORS中间件
	if s.config.CORSEnabled {
		s.router.Use(func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			
			c.Next()
		})
	}

	// 静态文件
	s.router.Static("/static", s.config.StaticDir)
	s.router.LoadHTMLGlob(s.config.TemplateDir + "/*")

	// 主页
	s.router.GET("/", s.handleHome)

	// API路由组
	api := s.router.Group("/api")
	{
		// 系统信息
		api.GET("/status", s.handleStatus)
		api.GET("/stats", s.handleStats)

		// 扫描控制
		api.POST("/scan/start", s.handleStartScan)
		api.POST("/scan/stop", s.handleStopScan)
		api.GET("/scan/status", s.handleScanStatus)

		// 密钥管理
		keys := api.Group("/keys")
		{
			keys.GET("/valid", s.handleGetValidKeys)
			keys.GET("/rate-limited", s.handleGetRateLimitedKeys)
			keys.DELETE("/valid/:id", s.handleDeleteValidKey)
			keys.DELETE("/rate-limited/:id", s.handleDeleteRateLimitedKey)
		}

		// 配置管理
		api.GET("/config", s.handleGetConfig)
		api.PUT("/config", s.handleUpdateConfig)

		// 查询规则管理
		api.GET("/queries", s.handleGetQueries)
		api.PUT("/queries", s.handleUpdateQueries)
		api.GET("/queries/default", s.handleGetDefaultQueries)
		
		// 安全审核管理
		security := api.Group("/security")
		{
			security.GET("/pending", s.handleGetPendingSecurityIssues)
			security.POST("/review/:id", s.handleReviewSecurityIssue)
			security.POST("/create-issue/:id", s.handleCreateSecurityIssue)
			security.GET("/issue/:id", s.handleGetSecurityIssue)
		}

		// 日志
		api.GET("/logs", s.handleGetLogs)

		// WebSocket
		api.GET("/ws", s.handleWebSocket)
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	logger.Infof("Starting web server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown 关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("Shutting down web server")
	return s.server.Shutdown(ctx)
}

// handleHome 主页处理
func (s *Server) handleHome(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Hajimi King Dashboard",
	})
}

// handleStatus 系统状态
func (s *Server) handleStatus(c *gin.Context) {
	status := map[string]interface{}{
		"status":    "running",
		"timestamp": time.Now(),
		"version":   "2.0.0",
		"uptime":    time.Since(time.Now()).String(), // TODO: 实际计算运行时间
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    status,
	})
}

// handleStats 获取统计信息
func (s *Server) handleStats(c *gin.Context) {
	// 获取扫描统计
	scanStats := s.scanner.GetStats()

	// 获取存储统计
	storageStats, err := s.storage.GetScanStats(c.Request.Context())
	if err != nil {
		logger.Errorf("Failed to get storage stats: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    1,
			Message: "Failed to get storage stats",
		})
		return
	}

	// 合并统计信息
	tokenStates := s.scanner.GetTokenStates()
	stats := map[string]interface{}{
		"scan":    scanStats,
		"storage": storageStats,
		"tokens":  len(tokenStates),
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    stats,
	})
}

// handleStartScan 开始扫描
func (s *Server) handleStartScan(c *gin.Context) {
	if s.scanner.IsScanning() {
		c.JSON(http.StatusBadRequest, Response{
			Code:    1,
			Message: "Scanner is already running",
		})
		return
	}

	// 在后台启动扫描
	go func() {
		ctx := context.Background()
		if err := s.scanner.StartContinuousScanning(ctx); err != nil {
			logger.Errorf("Continuous scanning failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "Scan started successfully",
	})
}

// handleStopScan 停止扫描
func (s *Server) handleStopScan(c *gin.Context) {
	if !s.scanner.IsScanning() {
		c.JSON(http.StatusBadRequest, Response{
			Code:    1,
			Message: "Scanner is not running",
		})
		return
	}

	s.scanner.Stop()

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "Scan stopped successfully",
	})
}

// handleScanStatus 获取扫描状态
func (s *Server) handleScanStatus(c *gin.Context) {
	stats := s.scanner.GetStats()
	
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    stats,
	})
}

// handleGetValidKeys 获取有效密钥列表
func (s *Server) handleGetValidKeys(c *gin.Context) {
	filter := &storage.KeyFilter{}

	// 解析查询参数
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Offset = o
		}
	}

	if source := c.Query("source"); source != "" {
		filter.Source = source
	}

	if repo := c.Query("repo"); repo != "" {
		filter.RepoName = repo
	}

	keys, total, err := s.storage.GetValidKeys(c.Request.Context(), filter)
	if err != nil {
		logger.Errorf("Failed to get valid keys: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    1,
			Message: "Failed to get valid keys",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"keys":  keys,
			"total": total,
		},
	})
}

// handleGetRateLimitedKeys 获取限流密钥列表
func (s *Server) handleGetRateLimitedKeys(c *gin.Context) {
	filter := &storage.KeyFilter{}

	// 解析查询参数
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Offset = o
		}
	}

	keys, total, err := s.storage.GetRateLimitedKeys(c.Request.Context(), filter)
	if err != nil {
		logger.Errorf("Failed to get rate limited keys: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    1,
			Message: "Failed to get rate limited keys",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"keys":  keys,
			"total": total,
		},
	})
}

// handleDeleteValidKey 删除有效密钥
func (s *Server) handleDeleteValidKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    1,
			Message: "Invalid key ID",
		})
		return
	}

	err = s.storage.DeleteValidKey(c.Request.Context(), id)
	if err != nil {
		logger.Errorf("Failed to delete valid key: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    1,
			Message: "Failed to delete key",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "Key deleted successfully",
	})
}

// handleDeleteRateLimitedKey 删除限流密钥
func (s *Server) handleDeleteRateLimitedKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    1,
			Message: "Invalid key ID",
		})
		return
	}

	err = s.storage.DeleteRateLimitedKey(c.Request.Context(), id)
	if err != nil {
		logger.Errorf("Failed to delete rate limited key: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    1,
			Message: "Failed to delete key",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "Key deleted successfully",
	})
}

// handleGetConfig 获取配置
func (s *Server) handleGetConfig(c *gin.Context) {
	// 返回完整的配置信息（不包含敏感信息如tokens）
	config := map[string]interface{}{
		"scanner": map[string]interface{}{
			"worker_count":    s.appConfig.Scanner.WorkerCount,
			"batch_size":      s.appConfig.Scanner.BatchSize,
			"scan_interval":   s.appConfig.Scanner.ScanInterval.Seconds(),
			"auto_start":      s.appConfig.Scanner.AutoStart,
			"date_range_days": s.appConfig.Scanner.DateRangeDays,
		},
		"validator": map[string]interface{}{
			"model_name":            s.appConfig.Scanner.Validator.ModelName,
			"tier_detection_model":  s.appConfig.Scanner.Validator.TierDetectionModel,
			"worker_count":          s.appConfig.Scanner.Validator.WorkerCount,
			"timeout":               s.appConfig.Scanner.Validator.Timeout.Seconds(),
			"enable_tier_detection": s.appConfig.Scanner.Validator.EnableTierDetection,
		},
		"rate_limit": map[string]interface{}{
			"enabled":            s.appConfig.RateLimit.Enabled,
			"requests_per_minute": s.appConfig.RateLimit.RequestsPerMinute,
			"burst_size":         s.appConfig.RateLimit.BurstSize,
			"adaptive_enabled":   s.appConfig.RateLimit.AdaptiveEnabled,
		},
		"web": map[string]interface{}{
			"port": s.config.Port,
			"host": s.config.Host,
		},
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    config,
	})
}

// handleUpdateConfig 更新配置
func (s *Server) handleUpdateConfig(c *gin.Context) {
	var req ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    1,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	logger.Info("Received configuration update request")

	// 更新扫描器配置
	if req.Scanner != nil {
		if req.Scanner.WorkerCount != nil {
			s.appConfig.Scanner.WorkerCount = *req.Scanner.WorkerCount
			logger.Infof("Updated scanner worker count to %d", *req.Scanner.WorkerCount)
		}
		if req.Scanner.BatchSize != nil {
			s.appConfig.Scanner.BatchSize = *req.Scanner.BatchSize
		}
		if req.Scanner.DateRangeDays != nil {
			s.appConfig.Scanner.DateRangeDays = *req.Scanner.DateRangeDays
		}
		if req.Scanner.AutoStart != nil {
			s.appConfig.Scanner.AutoStart = *req.Scanner.AutoStart
		}
	}

	// 更新验证器配置
	if req.Validator != nil {
		if req.Validator.ModelName != nil {
			s.appConfig.Scanner.Validator.ModelName = *req.Validator.ModelName
		}
		if req.Validator.TierDetectionModel != nil {
			s.appConfig.Scanner.Validator.TierDetectionModel = *req.Validator.TierDetectionModel
		}
		if req.Validator.WorkerCount != nil {
			s.appConfig.Scanner.Validator.WorkerCount = *req.Validator.WorkerCount
		}
		if req.Validator.Timeout != nil {
			s.appConfig.Scanner.Validator.Timeout = time.Duration(*req.Validator.Timeout) * time.Second
		}
		if req.Validator.EnableTierDetection != nil {
			s.appConfig.Scanner.Validator.EnableTierDetection = *req.Validator.EnableTierDetection
			logger.Infof("Updated tier detection enabled to %v", *req.Validator.EnableTierDetection)
		}
	}

	// 更新限流配置
	if req.RateLimit != nil {
		if req.RateLimit.Enabled != nil {
			s.appConfig.RateLimit.Enabled = *req.RateLimit.Enabled
		}
		if req.RateLimit.RequestsPerMinute != nil {
			s.appConfig.RateLimit.RequestsPerMinute = *req.RateLimit.RequestsPerMinute
		}
		if req.RateLimit.BurstSize != nil {
			s.appConfig.RateLimit.BurstSize = *req.RateLimit.BurstSize
		}
		if req.RateLimit.AdaptiveEnabled != nil {
			s.appConfig.RateLimit.AdaptiveEnabled = *req.RateLimit.AdaptiveEnabled
		}
	}

	// 通知Scanner更新配置（如果需要）
	// 注意：某些配置更改可能需要重启Scanner才能生效

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "Configuration updated successfully",
		Data: map[string]interface{}{
			"note": "Some changes may require service restart to take effect",
		},
	})
}

// handleGetLogs 获取日志
func (s *Server) handleGetLogs(c *gin.Context) {
	// TODO: 实现日志获取
	c.JSON(http.StatusNotImplemented, Response{
		Code:    1,
		Message: "Log retrieval not implemented yet",
	})
}

// handleWebSocket WebSocket处理
func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf("Failed to upgrade websocket: %v", err)
		return
	}
	defer conn.Close()

	logger.Info("WebSocket client connected")

	// 定期发送状态更新
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := s.scanner.GetStats()
			if err := conn.WriteJSON(map[string]interface{}{
				"type": "stats_update",
				"data": stats,
			}); err != nil {
				logger.Errorf("Failed to send websocket message: %v", err)
				return
			}
		}
	}
}

// handleGetQueries 获取查询规则
func (s *Server) handleGetQueries(c *gin.Context) {
	content, err := os.ReadFile(s.appConfig.Scanner.QueryFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": fmt.Sprintf("Failed to read query file: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"content": string(content),
		},
	})
}

// handleUpdateQueries 更新查询规则
func (s *Server) handleUpdateQueries(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// 备份原文件
	backupFile := s.appConfig.Scanner.QueryFile + ".backup"
	if err := os.Rename(s.appConfig.Scanner.QueryFile, backupFile); err != nil {
		logger.Warnf("Failed to create backup: %v", err)
	}

	// 写入新内容
	if err := os.WriteFile(s.appConfig.Scanner.QueryFile, []byte(req.Content), 0644); err != nil {
		// 恢复备份
		if backupErr := os.Rename(backupFile, s.appConfig.Scanner.QueryFile); backupErr != nil {
			logger.Errorf("Failed to restore backup: %v", backupErr)
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": fmt.Sprintf("Failed to save query file: %v", err),
		})
		return
	}

	// 删除备份文件
	os.Remove(backupFile)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Query rules updated successfully",
	})
}

// handleGetDefaultQueries 获取默认查询规则
func (s *Server) handleGetDefaultQueries(c *gin.Context) {
	// 读取默认查询规则（当前文件内容）
	defaultContent := `# 默认查询规则 - 基础模式
AIzaSy in:file
"sk-" in:file
"sk-ant-api03" in:file
"hf_" in:file
"ghp_" in:file fork:false
"AKIA" in:file fork:false

# 扩展查询
"AIzaSy" extension:js
"AIzaSy" extension:py
"AIzaSy" extension:env
"sk-" extension:env
"API_KEY" filename:.env`

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"content": defaultContent,
		},
	})
}

// handleGetPendingSecurityIssues 获取待审核的安全问题
func (s *Server) handleGetPendingSecurityIssues(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		status = "pending" // 默认只显示待审核的
	}
	
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	
	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}
	
	issues, total, err := s.storage.GetPendingSecurityIssues(c.Request.Context(), status, limit, offset)
	if err != nil {
		logger.Errorf("Failed to get pending security issues: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    2002,
			"message": "Failed to get pending security issues",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"issues": issues,
			"total":  total,
		},
	})
}

// handleReviewSecurityIssue 审核安全问题
func (s *Server) handleReviewSecurityIssue(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "Invalid issue ID",
		})
		return
	}
	
	var req struct {
		Action     string `json:"action"`     // approve, reject
		ReviewedBy string `json:"reviewed_by"`
		ReviewNote string `json:"review_note"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "Invalid request body",
		})
		return
	}
	
	// 验证action参数
	if req.Action != "approve" && req.Action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "Action must be 'approve' or 'reject'",
		})
		return
	}
	
	// 更新审核状态
	status := "approved"
	if req.Action == "reject" {
		status = "rejected"
	}
	
	err = s.storage.UpdateSecurityIssueStatus(c.Request.Context(), id, status, req.ReviewedBy, req.ReviewNote)
	if err != nil {
		logger.Errorf("Failed to update security issue status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    2002,
			"message": "Failed to update review status",
		})
		return
	}
	
	logger.Infof("Security issue %d %s by %s", id, status, req.ReviewedBy)
	
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Review submitted successfully",
	})
}

// handleCreateSecurityIssue 创建GitHub安全问题issue
func (s *Server) handleCreateSecurityIssue(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "Invalid issue ID",
		})
		return
	}
	
	// 获取安全问题详情
	issue, err := s.storage.GetSecurityIssueByID(c.Request.Context(), id)
	if err != nil {
		logger.Errorf("Failed to get security issue: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    2002,
			"message": "Failed to get security issue details",
		})
		return
	}
	
	if issue == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1002,
			"message": "Security issue not found",
		})
		return
	}
	
	// 检查是否已批准
	if issue.Status != "approved" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1003,
			"message": "Issue must be approved before creating GitHub issue",
		})
		return
	}
	
	// 检查是否已创建
	if issue.IssueURL != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1003,
			"message": "GitHub issue already created",
		})
		return
	}
	
	// 创建GitHub issue
	keyInfo := github.LeakedKeyInfo{
		KeyType:      issue.KeyType,
		Provider:     issue.Provider,
		Repository:   issue.RepoName,
		FilePath:     issue.FilePath,
		URL:          issue.FileURL,
		KeyPreview:   issue.KeyPreview,
		DiscoveredAt: issue.CreatedAt,
		Severity:     issue.Severity,
	}
	
	// 使用scanner的安全通知器创建issue
	if s.scanner != nil {
		securityNotifier := s.scanner.GetSecurityNotifier()
		if securityNotifier != nil {
			err = securityNotifier.CreateSecurityIssue(c.Request.Context(), keyInfo)
			if err != nil {
				logger.Errorf("Failed to create GitHub issue: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    2003,
					"message": "Failed to create GitHub issue: " + err.Error(),
				})
				return
			}
			
			// 更新issue URL（这里简化处理，实际应该从GitHub API获取issue URL）
			issueURL := fmt.Sprintf("https://github.com/%s/issues", issue.RepoName)
			err = s.storage.UpdateSecurityIssueURL(c.Request.Context(), id, issueURL)
			if err != nil {
				logger.Errorf("Failed to update issue URL: %v", err)
			}
			
			logger.Infof("✅ Created GitHub issue for security issue %d in %s", id, issue.RepoName)
			
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "GitHub issue created successfully",
				"data": gin.H{
					"issue_url": issueURL,
				},
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    2001,
				"message": "Security notifier not initialized",
			})
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    2001,
			"message": "Scanner not available",
		})
	}
}

// handleGetSecurityIssue 获取安全问题详情
func (s *Server) handleGetSecurityIssue(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "Invalid issue ID",
		})
		return
	}
	
	issue, err := s.storage.GetSecurityIssueByID(c.Request.Context(), id)
	if err != nil {
		logger.Errorf("Failed to get security issue: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    2002,
			"message": "Failed to get security issue details",
		})
		return
	}
	
	if issue == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1002,
			"message": "Security issue not found",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": issue,
	})
}