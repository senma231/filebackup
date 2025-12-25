package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// FileHandler 文件处理器
type FileHandler struct {
	repo     *repository.FileRepository
	agentRepo *repository.AgentRepository
}

// NewFileHandler 创建新的文件处理器
func NewFileHandler(db *sql.DB) *FileHandler {
	return &FileHandler{
		repo:     repository.NewFileRepository(db),
		agentRepo: repository.NewAgentRepository(db),
	}
}

// AgentFileStats Agent文件统计结构
type AgentFileStats struct {
	AgentID        string    `json:"agent_id"`
	Email          string    `json:"email"`
	EmailPrefix    string    `json:"email_prefix"`
	Hostname       string    `json:"hostname"`
	Status         string    `json:"status"`
	ScannedCount   int64     `json:"scanned_count"`
	UploadedCount  int64     `json:"uploaded_count"`
	PendingCount   int64     `json:"pending_count"`
	FailedCount    int64     `json:"failed_count"`
	LastUploadTime *time.Time `json:"last_upload_time"`
	CreatedAt      time.Time `json:"created_at"`
}

// UploadProgress 上传进度处理
func (h *FileHandler) UploadProgress(c *gin.Context) {
	agentID := c.Param("agent_id")

	var req struct {
		FileName      string `json:"file_name" binding:"required"`
		FileSize      int64  `json:"file_size"`
		FileType      string `json:"file_type"`
		UploadProgress int   `json:"upload_progress"`
		Status        string `json:"status" binding:"required"`
		LocalPath     string `json:"local_path"`
		RemotePath    string `json:"remote_path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, err.Error()))
		return
	}

	// 根据状态处理
	switch req.Status {
	case "started":
		// 文件上传开始
		now := time.Now()
		file := &model.FileUpload{
			AgentID:         agentID,
			LocalPath:       req.LocalPath,
			RemotePath:      req.RemotePath,
			FileName:        req.FileName,
			FileSize:        req.FileSize,
			FileType:        req.FileType,
			UploadStatus:    model.FileStatusUploading,
			UploadStartTime: &now,
			RetryCount:      0,
			CreatedAt:       now,
		}

		// 创建上传记录
		if err := h.repo.Create(file); err != nil {
			c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to create upload record"))
			return
		}

	case "progress":
		// 更新上传进度
		if err := h.repo.UpdateProgress(agentID, req.FileName, req.UploadProgress); err != nil {
			c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to update progress"))
			return
		}

	case "completed":
		// 上传完成
		if err := h.repo.UpdateStatusByFileName(agentID, req.FileName, model.FileStatusSuccess, ""); err != nil {
			c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to update status"))
			return
		}

	case "failed":
		// 上传失败
		errorMsg := "Upload failed"
		if err := h.repo.UpdateStatusByFileName(agentID, req.FileName, model.FileStatusFailed, errorMsg); err != nil {
			c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to update status"))
			return
		}
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"status": "ok",
	}))
}

// GetAll 获取所有Agent的文件统计（分页）
func (h *FileHandler) GetAll(c *gin.Context) {
	page := parseInt(c.DefaultQuery("page", "1"))
	perPage := parseInt(c.DefaultQuery("per_page", c.DefaultQuery("size", "20")))

	log.Printf("[GET_FILE_STATS] Query: page=%d, perPage=%d", page, perPage)

	// 获取所有Agent
	agents, _, err := h.agentRepo.GetAll(1, 1000, "all")
	if err != nil {
		log.Printf("[GET_FILE_STATS] Error getting agents: %v", err)
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agents"))
		return
	}

	log.Printf("[GET_FILE_STATS] Found %d agents", len(agents))

	// 为每个Agent获取文件统计
	var stats []*AgentFileStats
	for _, agent := range agents {
		stat := &AgentFileStats{
			AgentID:     agent.AgentID,
			Email:       agent.Email,
			EmailPrefix: agent.EmailPrefix,
			Hostname:    agent.Hostname,
			Status:      agent.Status,
			CreatedAt:   agent.CreatedAt,
		}

		// 获取该Agent的文件统计
		uploadedCount, err := h.repo.CountByAgentAndStatus(agent.AgentID, "success")
		if err != nil {
			log.Printf("[GET_FILE_STATS] Error getting uploaded count for %s: %v", agent.AgentID, err)
		} else {
			stat.UploadedCount = uploadedCount
		}

		pendingCount, err := h.repo.CountByAgentAndStatus(agent.AgentID, "pending")
		if err != nil {
			log.Printf("[GET_FILE_STATS] Error getting pending count for %s: %v", agent.AgentID, err)
		} else {
			stat.PendingCount = pendingCount
		}

		failedCount, err := h.repo.CountByAgentAndStatus(agent.AgentID, "failed")
		if err != nil {
			log.Printf("[GET_FILE_STATS] Error getting failed count for %s: %v", agent.AgentID, err)
		} else {
			stat.FailedCount = failedCount
		}

		// 扫描总数 = 上传成功 + 待上传 + 上传失败
		stat.ScannedCount = stat.UploadedCount + stat.PendingCount + stat.FailedCount

		// 获取最后上传时间
		lastUpload, err := h.repo.GetLastUploadTime(agent.AgentID)
		if err != nil {
			log.Printf("[GET_FILE_STATS] Error getting last upload time for %s: %v", agent.AgentID, err)
		} else {
			stat.LastUploadTime = lastUpload
		}

		stats = append(stats, stat)
	}

	// 分页处理
	total := int64(len(stats))
	start := (page - 1) * perPage
	end := start + perPage

	if start > int(total) {
		start = int(total)
	}
	if end > int(total) {
		end = int(total)
	}

	var pagedStats []*AgentFileStats
	if start < end {
		pagedStats = stats[start:end]
	} else {
		pagedStats = []*AgentFileStats{}
	}

	log.Printf("[GET_FILE_STATS] Returning: total=%d, page=%d, returned=%d", total, page, len(pagedStats))

	// 返回格式: { code: 200, message: "success", data: { total, page, per_page, records } }
	paginatedData := gin.H{
		"total":    total,
		"page":     page,
		"per_page": perPage,
		"records":  pagedStats,
	}
	c.JSON(http.StatusOK, model.Success(paginatedData))
}

// GetByID 根据ID获取文件记录
func (h *FileHandler) GetByID(c *gin.Context) {
	fileID := parseInt(c.Param("file_id"))

	file, err := h.repo.GetByID(fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get file"))
		return
	}

	if file == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "File not found"))
		return
	}

	c.JSON(http.StatusOK, model.Success(file))
}

// GetByAgentID 获取指定Agent的文件
func (h *FileHandler) GetByAgentID(c *gin.Context) {
	agentID := c.Param("agent_id")
	limit := parseInt(c.DefaultQuery("limit", "100"))

	files, err := h.repo.GetByAgentID(agentID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get files"))
		return
	}

	c.JSON(http.StatusOK, model.Success(files))
}
