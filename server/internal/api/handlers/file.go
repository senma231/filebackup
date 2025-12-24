package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// FileHandler 文件处理器
type FileHandler struct {
	repo *repository.FileRepository
}

// NewFileHandler 创建新的文件处理器
func NewFileHandler(db *sql.DB) *FileHandler {
	return &FileHandler{
		repo: repository.NewFileRepository(db),
	}
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
		file := &model.FileUpload{
			AgentID:         agentID,
			LocalPath:       req.LocalPath,
			RemotePath:      req.RemotePath,
			FileName:        req.FileName,
			FileSize:        req.FileSize,
			FileType:        req.FileType,
			UploadStatus:    model.FileStatusUploading,
			UploadStartTime: &time.Time{},
			RetryCount:      0,
			CreatedAt:       time.Now(),
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

// GetAll 获取所有文件记录
func (h *FileHandler) GetAll(c *gin.Context) {
	page := parseInt(c.DefaultQuery("page", "1"))
	perPage := parseInt(c.DefaultQuery("size", "20"))
	agentID := c.DefaultQuery("agent_id", "")
	status := c.DefaultQuery("status", "all")

	files, total, err := h.repo.GetAll(page, perPage, agentID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get files"))
		return
	}

	c.JSON(http.StatusOK, model.NewPaginatedResponse(total, page, perPage, files))
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

// parseInt 解析字符串为整数
func parseInt(s string) int {
	var result int
	// 简单的字符串转整数，出错时返回0
	if s == "" {
		return 0
	}
	// 使用 strconv
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return 0
}
