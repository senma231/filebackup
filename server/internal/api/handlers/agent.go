package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// AgentHandler Agent处理器
type AgentHandler struct {
	repo *repository.AgentRepository
}

// NewAgentHandler 创建新的Agent处理器
func NewAgentHandler(db *sql.DB) *AgentHandler {
	return &AgentHandler{
		repo: repository.NewAgentRepository(db),
	}
}

// Register Agent注册
func (h *AgentHandler) Register(c *gin.Context) {
	var req struct {
		AgentID      string `json:"agent_id" binding:"required"`
		Email        string `json:"email"`
		Hostname     string `json:"hostname" binding:"required"`
		IPAddress    string `json:"ip_address" binding:"required"`
		OSVersion    string `json:"os_version"`
		AgentVersion string `json:"agent_version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, err.Error()))
		return
	}

	// 检查Agent是否已存在
	existing, err := h.repo.GetByID(req.AgentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to check agent"))
		return
	}

	now := time.Now()
	// 提取邮箱前缀
	emailPrefix := extractEmailPrefix(req.Email)

	if existing != nil {
		// 更新现有Agent
		existing.Email = req.Email
		existing.EmailPrefix = emailPrefix
		existing.Hostname = req.Hostname
		existing.IPAddress = req.IPAddress
		existing.OSVersion = req.OSVersion
		existing.AgentVersion = req.AgentVersion
		existing.Status = model.AgentStatusOnline
		existing.LastHeartbeat = &now
		existing.UpdatedAt = now

		if err := h.repo.Update(existing); err != nil {
			c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to update agent"))
			return
		}
	} else {
		// 创建新Agent
		agent := &model.Agent{
			AgentID:      req.AgentID,
			Email:        req.Email,
			EmailPrefix:  emailPrefix,
			Hostname:     req.Hostname,
			IPAddress:    req.IPAddress,
			OSVersion:    req.OSVersion,
			AgentVersion: req.AgentVersion,
			Status:       model.AgentStatusOnline,
			LastHeartbeat: &now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := h.repo.Create(agent); err != nil {
			c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to create agent"))
			return
		}
	}

	// 返回响应
	c.JSON(http.StatusOK, model.Success(gin.H{
		"agent_id": req.AgentID,
		"status":   "registered",
	}))
}

// Heartbeat 心跳处理
func (h *AgentHandler) Heartbeat(c *gin.Context) {
	agentID := c.Param("agent_id")

	var req struct {
		Email     string `json:"email"`
		Hostname  string `json:"hostname"`
		IPAddress string `json:"ip_address"`
		Status    string `json:"status"`
		Stats     struct {
			CPUUsage    float64 `json:"cpu_usage"`
			MemoryUsage int64   `json:"memory_usage"`
			DiskUsage   int64   `json:"disk_usage"`
			ScanCount   int     `json:"scan_count"`
			UploadCount int     `json:"upload_count"`
			ErrorCount  int     `json:"error_count"`
		} `json:"stats"`
		Timestamp time.Time `json:"timestamp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, err.Error()))
		return
	}

	// 更新心跳和Agent信息
	stats := model.SystemStats{
		CPUUsage:    req.Stats.CPUUsage,
		MemoryUsage: req.Stats.MemoryUsage,
		DiskUsage:   req.Stats.DiskUsage,
		ScanCount:   req.Stats.ScanCount,
		UploadCount: req.Stats.UploadCount,
		ErrorCount:  req.Stats.ErrorCount,
	}

	// 获取或创建Agent记录
	agent, err := h.repo.GetByID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agent"))
		return
	}

	// 如果Agent不存在，自动创建
	if agent == nil {
		now := time.Now()
		agent = &model.Agent{
			AgentID:      agentID,
			Email:        req.Email,
			EmailPrefix:  extractEmailPrefix(req.Email),
			Hostname:     req.Hostname,
			IPAddress:    req.IPAddress,
			Status:       "online",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := h.repo.Create(agent); err != nil {
			c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to create agent"))
			return
		}
	} else {
		// 更新Agent的基本信息
		updated := false

		if req.Email != "" && agent.Email != req.Email {
			agent.Email = req.Email
			agent.EmailPrefix = extractEmailPrefix(req.Email)
			updated = true
		}

		if req.Hostname != "" && agent.Hostname != req.Hostname {
			agent.Hostname = req.Hostname
			updated = true
		}

		if req.IPAddress != "" && agent.IPAddress != req.IPAddress {
			agent.IPAddress = req.IPAddress
			updated = true
		}

		// 确保状态更新为online
		if agent.Status != "online" {
			agent.Status = "online"
			updated = true
		}

		if updated {
			agent.UpdatedAt = time.Now()
			h.repo.Update(agent)
		}
	}

	// 更新心跳时间和统计数据
	if err := h.repo.UpdateHeartbeat(agentID, stats); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to update heartbeat"))
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, model.Success(gin.H{
		"config_update": false,
		"config_data":   nil,
	}))
}

// GetConfig 获取Agent配置
func (h *AgentHandler) GetConfig(c *gin.Context) {
	agentID := c.Param("agent_id")

	// 检查Agent是否存在
	agent, err := h.repo.GetByID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agent"))
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Agent not found"))
		return
	}

	// 返回默认配置
	config := gin.H{
		"scan_paths": []string{
			"C:\\Users\\%USERNAME%\\Documents",
		},
		"file_types": []string{
			".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".pdf", ".txt",
		},
		"exclude_patterns": []string{
			"*.log", "*.tmp", "~$*",
		},
		"heartbeat_interval": 30,
		"max_file_size": 104857600,
		"compress_enabled": true,
	}

	c.JSON(http.StatusOK, model.Success(config))
}

// GetAll 获取所有Agent
func (h *AgentHandler) GetAll(c *gin.Context) {
	page := parseInt(c.DefaultQuery("page", "1"))
	perPage := parseInt(c.DefaultQuery("per_page", c.DefaultQuery("size", "20")))
	status := c.DefaultQuery("status", "all")

	agents, total, err := h.repo.GetAll(page, perPage, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agents"))
		return
	}

	c.JSON(http.StatusOK, model.NewPaginatedResponse(total, page, perPage, agents))
}

// GetByID 根据ID获取Agent
func (h *AgentHandler) GetByID(c *gin.Context) {
	agentID := c.Param("agent_id")

	agent, err := h.repo.GetByID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agent"))
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Agent not found"))
		return
	}

	c.JSON(http.StatusOK, model.Success(agent))
}

// GetHeartbeats 获取心跳记录
func (h *AgentHandler) GetHeartbeats(c *gin.Context) {
	agentID := c.Param("agent_id")
	limit := parseInt(c.DefaultQuery("limit", "100"))

	heartbeats, err := h.repo.GetHeartbeats(agentID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get heartbeats"))
		return
	}

	c.JSON(http.StatusOK, model.Success(heartbeats))
}

// parseInt 解析字符串为整数
func parseInt(s string) int {
	var result int
	json.Unmarshal([]byte(s), &result)
	return result
}

// extractEmailPrefix 从邮箱中提取前缀
func extractEmailPrefix(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}

	return email
}
