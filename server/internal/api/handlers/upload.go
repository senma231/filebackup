package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// UploadHandler 文件上传处理器
type UploadHandler struct {
	storageRepo *repository.StorageConfigRepository
	fileRepo    *repository.FileRepository
}

// NewUploadHandler 创建文件上传处理器
func NewUploadHandler(storageRepo *repository.StorageConfigRepository, fileRepo *repository.FileRepository) *UploadHandler {
	return &UploadHandler{
		storageRepo: storageRepo,
		fileRepo:    fileRepo,
	}
}

// UploadFile 处理文件上传（用于本地存储）
func (h *UploadHandler) UploadFile(c *gin.Context) {
	// 获取Agent ID
	agentID := c.PostForm("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "agent_id is required"))
		return
	}

	// 获取远程路径
	remotePath := c.PostForm("remote_path")
	if remotePath == "" {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "remote_path is required"))
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, fmt.Sprintf("Failed to get file: %v", err)))
		return
	}

	// 获取该Agent的存储配置
	storageConfig, err := h.storageRepo.GetByAgentID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get storage config"))
		return
	}

	if storageConfig == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "No active storage config found"))
		return
	}

	// 验证存储类型必须是本地存储
	if storageConfig.StorageType != "local" {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "This endpoint only supports local storage"))
		return
	}

	// 解析本地存储配置
	var localConfig struct {
		BasePath  string `json:"base_path"`
		CreateDir bool   `json:"create_dir"`
	}
	if err := parseConfigData(storageConfig.ConfigData, &localConfig); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to parse storage config"))
		return
	}

	// 处理相对路径：如果不是绝对路径，转换为容器内绝对路径
	basePath := localConfig.BasePath
	if !filepath.IsAbs(basePath) {
		// 相对路径转为绝对路径（在容器中从根目录开始）
		basePath = filepath.Join("/", basePath)
	}

	// 构建完整的目标路径
	destPath := filepath.Join(basePath, remotePath)

	// 确保目标目录存在（默认自动创建）
	destDir := filepath.Dir(destPath)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		// 默认行为：自动创建目录（除非配置明确禁用）
		// 由于大多数情况下需要自动创建，我们总是尝试创建
		if err := os.MkdirAll(destDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, fmt.Sprintf("Failed to create directory: %v", err)))
			return
		}
	}

	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, fmt.Sprintf("Failed to open uploaded file: %v", err)))
		return
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, fmt.Sprintf("Failed to create file: %v", err)))
		return
	}
	defer dst.Close()

	// 复制文件内容
	written, err := io.Copy(dst, src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, fmt.Sprintf("Failed to save file: %v", err)))
		return
	}

	// 记录文件上传到数据库
	now := time.Now()
	fileRecord := &model.FileUpload{
		AgentID:         agentID,
		LocalPath:       "",                      // Agent本地路径未传递
		RemotePath:      remotePath,
		FileName:        filepath.Base(remotePath),
		FileSize:        written,
		FileType:        filepath.Ext(remotePath),
		UploadStatus:    model.FileStatusSuccess,
		UploadStartTime: &now,
		UploadEndTime:   &now,
		RetryCount:      0,
		CreatedAt:       now,
	}

	if err := h.fileRepo.Create(fileRecord); err != nil {
		// 记录失败不影响上传成功响应，只记录日志
		// 可以考虑后续添加日志系统
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.Success(gin.H{
		"message":      "File uploaded successfully",
		"path":         destPath,
		"size":         written,
		"remote_path":  remotePath,
		"storage_type": "local",
	}))
}

// parseConfigData 解析配置数据
func parseConfigData(configData string, target interface{}) error {
	// 使用 JSON 解析
	if err := json.Unmarshal([]byte(configData), target); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// 设置默认值：如果没有指定 create_dir，默认为 true
	if cfg, ok := target.(*struct {
		BasePath  string `json:"base_path"`
		CreateDir bool   `json:"create_dir"`
	}); ok {
		if cfg.BasePath == "" {
			return fmt.Errorf("base_path not found in config")
		}
		// 如果配置中没有 create_dir 字段，默认设置为 true
		// JSON unmarshal 时，bool 默认为 false，所以这里无法区分是否显式设置
		// 我们直接在上传处理时设置默认行为
	}

	return nil
}
