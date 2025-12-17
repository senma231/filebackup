package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// DownloadHandler Agent下载处理器
type DownloadHandler struct {
	repo *repository.AgentRepository
}

// NewDownloadHandler 创建新的下载处理器
func NewDownloadHandler(db *sql.DB) *DownloadHandler {
	return &DownloadHandler{
		repo: repository.NewAgentRepository(db),
	}
}

// RequestDownload 请求下载Agent
func (h *DownloadHandler) RequestDownload(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required"`
		Hostname    string `json:"hostname" binding:"required"`
		IPAddress   string `json:"ip_address" binding:"required"`
		OSVersion   string `json:"os_version"`
		AgentVersion string `json:"agent_version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, err.Error()))
		return
	}

	// 验证邮箱格式
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "Invalid email format"))
		return
	}

	// 提取邮箱前缀
	emailPrefix := extractEmailPrefix(req.Email)

	// 生成唯一的Agent ID
	agentID := generateAgentID()

	// 创建Agent记录（状态为pending，等待Agent启动后激活）
	now := time.Now()
	agent := &model.Agent{
		AgentID:      agentID,
		Email:        req.Email,
		EmailPrefix:  emailPrefix,
		Hostname:     req.Hostname,
		IPAddress:    req.IPAddress,
		OSVersion:    req.OSVersion,
		AgentVersion: req.AgentVersion,
		Status:       "pending",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.repo.Create(agent); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to create agent record"))
		return
	}

	// 生成配置信息（包含Server地址和Agent ID）
	config := gin.H{
		"server_url":       getServerURL(c),
		"agent_id":         agentID,
		"email":            req.Email,
		"email_prefix":     emailPrefix,
		"heartbeat_interval": 30,
		"scan_paths": []string{
			getDefaultDocumentsPath(req.Hostname),
		},
		"file_types": []string{
			".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".pdf", ".txt",
		},
		"exclude_patterns": []string{
			"*.log", "*.tmp", "~$*",
		},
		"max_file_size": 104857600,
		"incremental_scan": true,
		"compress_enabled": true,
		"sftp_config": gin.H{
			"host":       getSFTPHost(),
			"port":       22,
			"username":   emailPrefix,
			"remote_path": "/uploads",
		},
	}

	// 返回配置信息和下载链接
	c.JSON(http.StatusOK, model.Success(gin.H{
		"agent_id":     agentID,
		"email":        req.Email,
		"email_prefix": emailPrefix,
		"config":       config,
		"download_url": fmt.Sprintf("%s/download/agent/%s", getServerURL(c), agentID),
		"instructions": gin.H{
			"step1": "下载并安装Agent",
			"step2": "Agent将自动连接到服务器",
			"step3": "开始扫描和上传文档",
		},
	}))
}

// GetAgentConfig 获取Agent配置（Agent启动时调用）
func (h *DownloadHandler) GetAgentConfig(c *gin.Context) {
	agentID := c.Param("agent_id")

	// 查找Agent
	agent, err := h.repo.GetByID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agent"))
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Agent not found"))
		return
	}

	// 生成配置信息
	config := gin.H{
		"server_url":        getServerURL(c),
		"agent_id":          agentID,
		"email":             agent.Email,
		"email_prefix":      agent.EmailPrefix,
		"heartbeat_interval": 30,
		"scan_paths": []string{
			getDefaultDocumentsPath(agent.Hostname),
		},
		"file_types": []string{
			".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".pdf", ".txt",
		},
		"exclude_patterns": []string{
			"*.log", "*.tmp", "~$*",
		},
		"max_file_size": 104857600,
		"incremental_scan": true,
		"compress_enabled": true,
		"sftp_config": gin.H{
			"host":       getSFTPHost(),
			"port":       22,
			"username":   agent.EmailPrefix,
			"remote_path": "/uploads",
		},
	}

	c.JSON(http.StatusOK, model.Success(config))
}

// DownloadAgent 下载Agent程序
func (h *DownloadHandler) DownloadAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	// 验证Agent是否存在
	agent, err := h.repo.GetByID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agent"))
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Agent not found"))
		return
	}

	// TODO: 返回实际的Agent程序文件
	// 目前返回配置信息作为示例
	c.JSON(http.StatusOK, model.Success(gin.H{
		"message":     "Agent download feature coming soon",
		"agent_id":    agentID,
		"email":       agent.Email,
		"config":      "Configuration will be provided separately",
	}))
}

// 辅助函数

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}

	// 简单的邮箱格式验证
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	if parts[0] == "" || parts[1] == "" {
		return false
	}

	return strings.Contains(parts[1], ".")
}

// generateAgentID 生成唯一的Agent ID
func generateAgentID() string {
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}

// getServerURL 获取服务器URL
func getServerURL(c *gin.Context) string {
	scheme := c.Request.URL.Scheme
	if scheme == "" {
		scheme = "http"
	}

	host := c.Request.Host
	return fmt.Sprintf("%s://%s", scheme, host)
}

// getDefaultDocumentsPath 获取默认文档路径
func getDefaultDocumentsPath(hostname string) string {
	// Windows系统的默认文档路径
	return "C:\\Users\\%USERNAME%\\Documents"
}

// getSFTPHost 获取SFTP主机地址
func getSFTPHost() string {
	// 从环境变量或配置中获取
	// 这里返回默认值
	return "sftp.example.com"
}
