package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"kooix-hajimi/internal/config"
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
	upgrader websocket.Upgrader
}

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
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
		router:  router,
		scanner: scanner,
		storage: store,
		config:  cfg.Web,
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
	// 返回安全的配置信息（不包含敏感信息）
	config := map[string]interface{}{
		"scanner": map[string]interface{}{
			"worker_count":   10, // TODO: 从实际配置获取
			"scan_interval":  "10s",
			"auto_start":     false,
			"date_range_days": 730,
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
	// TODO: 实现配置更新
	c.JSON(http.StatusNotImplemented, Response{
		Code:    1,
		Message: "Configuration update not implemented yet",
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