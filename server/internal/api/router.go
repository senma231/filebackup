package api

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"doc-scanner-server/internal/api/handlers"
	"doc-scanner-server/internal/api/middleware"

	"github.com/gin-gonic/gin"
)

// Server HTTP服务器
type Server struct {
	db *sql.DB
}

// NewServer 创建新的HTTP服务器
func NewServer(db *sql.DB) *Server {
	return &Server{db: db}
}

// Start 启动HTTP服务器
func (s *Server) Start(addr string) error {
	// 设置为发布模式
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// 添加中间件
	r.Use(middleware.CORS())

	// 创建处理器
	agentHandler := handlers.NewAgentHandler(s.db)
	configHandler := handlers.NewConfigHandler(s.db)
	fileHandler := handlers.NewFileHandler(s.db)
	reportHandler := handlers.NewReportHandler(s.db)
	downloadHandler := handlers.NewDownloadHandler(s.db)

	// 获取可执行文件所在目录
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("Warning: Error getting executable path: %v, using default", err)
		execPath, _ = os.Getwd()
	}
	execDir := filepath.Dir(execPath)
	agentBinPath := filepath.Join(execDir, "bin", "agent.exe")
	
	// 检查agent.exe是否存在，如果不存在尝试其他路径
	if _, err := os.Stat(agentBinPath); os.IsNotExist(err) {
		log.Printf("Warning: agent.exe not found at %s, trying alternative paths", agentBinPath)
		// 尝试相对路径
		alternativePaths := []string{
			"./bin/agent.exe",
			"../agent/agent.exe",
			"../../agent/agent.exe",
		}
		for _, altPath := range alternativePaths {
			if _, err := os.Stat(altPath); err == nil {
				agentBinPath = altPath
				break
			}
		}
	}

	downloadHandlerEnhanced := handlers.NewDownloadHandlerEnhanced(s.db, agentBinPath)

	// 公开API路由组
	v1 := r.Group("/api/v1")
	{
		// Agent下载和注册（增强版）
		download := v1.Group("/download")
		{
			download.POST("/enhanced/request", downloadHandlerEnhanced.RequestDownloadEnhanced)
			download.GET("/enhanced/package/:agent_id", downloadHandlerEnhanced.DownloadPackage)
			download.HEAD("/enhanced/package/:agent_id", downloadHandlerEnhanced.DownloadPackage)
			download.GET("/agent/:agent_id", downloadHandler.GetAgentConfig)
			download.GET("/agent/:agent_id/download", downloadHandler.DownloadAgent)
		}

		// Agent相关接口
		agents := v1.Group("/agents")
		{
			agents.POST("/register", agentHandler.Register)
			agents.POST("/:agent_id/heartbeat", agentHandler.Heartbeat)
			agents.GET("/:agent_id/config", agentHandler.GetConfig)
		}

		// 文件上传相关接口
		v1.POST("/agents/:agent_id/upload/progress", fileHandler.UploadProgress)
	}

	// 管理API路由组（需要认证）
	admin := r.Group("/api/v1/admin")
	{
		// Agent管理
		admin.GET("/agents", agentHandler.GetAll)
		admin.GET("/agents/:agent_id", agentHandler.GetByID)
		admin.GET("/agents/:agent_id/heartbeats", agentHandler.GetHeartbeats)
		admin.GET("/agents/:agent_id/files", fileHandler.GetByAgentID)

		// 配置管理
		admin.GET("/configs", configHandler.GetAll)
		admin.POST("/configs", configHandler.Create)
		admin.GET("/configs/:config_id", configHandler.GetByID)
		admin.GET("/agents/:agent_id/config", configHandler.GetByAgentID)
		admin.PUT("/configs/:config_id", configHandler.Update)
		admin.DELETE("/configs/:config_id", configHandler.Delete)

		// 文件管理
		admin.GET("/files", fileHandler.GetAll)
		admin.GET("/files/:file_id", fileHandler.GetByID)

		// 报告统计
		admin.GET("/reports/stats", reportHandler.GetStats)
		admin.GET("/reports/agents", reportHandler.GetAgentStats)
		admin.GET("/reports/files", reportHandler.GetFileStats)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// 根路径 - 重定向到下载页面
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/download")
	})

	// 静态文件服务
	webDir := filepath.Join(execDir, "web")
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		// 尝试相对路径
		for _, relPath := range []string{"./web", "../server/web", "../../server/web"} {
			if _, err := os.Stat(relPath); err == nil {
				webDir = relPath
				break
			}
		}
	}
	
	// 检查 web 目录是否存在
	if _, err := os.Stat(webDir); err == nil {
		log.Printf("Serving static files from: %s", webDir)
		r.Static("/static", filepath.Join(webDir, "static"))
	} else {
		log.Printf("Warning: web directory not found")
	}

	// 下载页面
	r.GET("/download", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		
		// 尝试多个可能的路径
		possiblePaths := []string{
			filepath.Join(webDir, "templates", "dashboard_enhanced.html"),
			filepath.Join(execDir, "web", "templates", "dashboard_enhanced.html"),
			filepath.Join(execDir, "templates", "dashboard_enhanced.html"),
			"./web/templates/dashboard_enhanced.html",
			"../server/web/templates/dashboard_enhanced.html",
		}
		
		var templatePath string
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				templatePath = path
				break
			}
		}
		
		if templatePath == "" {
			log.Printf("Template file not found, tried: %v", possiblePaths)
			c.String(404, "Download page not found")
			return
		}

		log.Printf("Loading template from: %s", templatePath)
		c.File(templatePath)
	})

	// 管理端页面
	r.GET("/admin", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		
		// 尝试多个可能的路径
		possiblePaths := []string{
			filepath.Join(webDir, "templates", "admin.html"),
			filepath.Join(execDir, "web", "templates", "admin.html"),
			filepath.Join(execDir, "templates", "admin.html"),
			"./web/templates/admin.html",
			"../server/web/templates/admin.html",
		}
		
		var templatePath string
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				templatePath = path
				break
			}
		}
		
		if templatePath == "" {
			log.Printf("Admin template file not found, tried: %v", possiblePaths)
			c.String(404, "Admin page not found")
			return
		}

		log.Printf("Loading admin template from: %s", templatePath)
		c.File(templatePath)
	})

	log.Printf("Server starting on %s", addr)
	return r.Run(addr)
}
