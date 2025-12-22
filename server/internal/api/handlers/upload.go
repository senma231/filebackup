package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// UploadHandler 文件上传处理器
type UploadHandler struct {
	storageRepo *repository.StorageRepository
}

// NewUploadHandler 创建文件上传处理器
func NewUploadHandler(storageRepo *repository.StorageRepository) *UploadHandler {
	return &UploadHandler{
		storageRepo: storageRepo,
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
	storageConfig, err := h.storageRepo.GetActiveConfigForAgent(agentID)
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

	// 构建完整的目标路径
	destPath := filepath.Join(localConfig.BasePath, remotePath)

	// 确保目标目录存在
	destDir := filepath.Dir(destPath)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		if localConfig.CreateDir {
			if err := os.MkdirAll(destDir, 0755); err != nil {
				c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, fmt.Sprintf("Failed to create directory: %v", err)))
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "Target directory does not exist"))
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
	// 使用encoding/json包（需要在import中添加）
	// 这里提供简单的实现

	// 提取 base_path
	basePath := ""
	createDir := false

	if idx := strings.Index(configData, `"base_path":"`); idx >= 0 {
		start := idx + len(`"base_path":"`)
		if end := strings.Index(configData[start:], `"`); end >= 0 {
			basePath = configData[start : start+end]
		}
	}

	if strings.Contains(configData, `"create_dir":true`) {
		createDir = true
	}

	// 使用类型断言设置值
	if cfg, ok := target.(*struct {
		BasePath  string `json:"base_path"`
		CreateDir bool   `json:"create_dir"`
	}); ok {
		cfg.BasePath = basePath
		cfg.CreateDir = createDir

		if basePath == "" {
			return fmt.Errorf("base_path not found in config")
		}
	}

	return nil
}
