package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// StorageConfigHandler 存储配置处理器
type StorageConfigHandler struct {
	repo *repository.StorageConfigRepository
}

// NewStorageConfigHandler 创建存储配置处理器
func NewStorageConfigHandler(db *sql.DB) *StorageConfigHandler {
	repo := repository.NewStorageConfigRepository(db)
	// 初始化表
	if err := repo.InitTable(); err != nil {
		panic(err)
	}
	return &StorageConfigHandler{repo: repo}
}

// GetAll 获取所有存储配置
func (h *StorageConfigHandler) GetAll(c *gin.Context) {
	storageType := c.Query("storage_type")
	agentID := c.Query("agent_id")

	configs, err := h.repo.GetAll(storageType, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "获取存储配置失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(configs))
}

// GetByID 根据ID获取存储配置
func (h *StorageConfigHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "无效的配置ID"))
		return
	}

	config, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "获取存储配置失败"))
		return
	}

	if config == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "存储配置不存在"))
		return
	}

	c.JSON(http.StatusOK, model.Success(config))
}

// GetByAgentID 获取Agent的存储配置
func (h *StorageConfigHandler) GetByAgentID(c *gin.Context) {
	agentID := c.Param("agent_id")

	config, err := h.repo.GetByAgentID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "获取Agent存储配置失败"))
		return
	}

	if config == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "未找到适用的存储配置"))
		return
	}

	c.JSON(http.StatusOK, model.Success(config))
}

// Create 创建存储配置
func (h *StorageConfigHandler) Create(c *gin.Context) {
	var req struct {
		Name          string             `json:"name" binding:"required"`
		StorageType   model.StorageType  `json:"storage_type" binding:"required"`
		ConfigData    string             `json:"config_data" binding:"required"`
		Description   string             `json:"description"`
		TargetAgentID *string            `json:"target_agent_id"`
		Priority      int                `json:"priority"`
		IsActive      bool               `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "请求参数错误"))
		return
	}

	// 验证存储类型
	validTypes := []model.StorageType{
		model.StorageTypeSFTP,
		model.StorageTypeWebDAV,
		model.StorageTypeAliyunOSS,
		model.StorageTypeTencentCOS,
		model.StorageTypeAWSS3,
		model.StorageTypeSMB,
		model.StorageTypeLocal,
	}

	isValidType := false
	for _, t := range validTypes {
		if req.StorageType == t {
			isValidType = true
			break
		}
	}

	if !isValidType {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "不支持的存储类型"))
		return
	}

	// 验证配置数据格式
	if err := repository.ValidateConfig(req.StorageType, req.ConfigData); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, err.Error()))
		return
	}

	config := &model.StorageConfig{
		Name:          req.Name,
		StorageType:   req.StorageType,
		ConfigData:    req.ConfigData,
		Description:   req.Description,
		TargetAgentID: req.TargetAgentID,
		Priority:      req.Priority,
		IsActive:      req.IsActive,
	}

	if err := h.repo.Create(config); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "创建存储配置失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(config))
}

// Update 更新存储配置
func (h *StorageConfigHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "无效的配置ID"))
		return
	}

	// 获取现有配置
	existingConfig, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "获取存储配置失败"))
		return
	}

	if existingConfig == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "存储配置不存在"))
		return
	}

	var req struct {
		Name          *string            `json:"name"`
		StorageType   *model.StorageType `json:"storage_type"`
		ConfigData    *string            `json:"config_data"`
		Description   *string            `json:"description"`
		TargetAgentID *string            `json:"target_agent_id"`
		Priority      *int               `json:"priority"`
		IsActive      *bool              `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "请求参数错误"))
		return
	}

	// 更新字段
	if req.Name != nil {
		existingConfig.Name = *req.Name
	}
	if req.StorageType != nil {
		existingConfig.StorageType = *req.StorageType
	}
	if req.ConfigData != nil {
		// 验证配置数据格式
		if err := repository.ValidateConfig(existingConfig.StorageType, *req.ConfigData); err != nil {
			c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, err.Error()))
			return
		}
		existingConfig.ConfigData = *req.ConfigData
	}
	if req.Description != nil {
		existingConfig.Description = *req.Description
	}
	if req.TargetAgentID != nil {
		existingConfig.TargetAgentID = req.TargetAgentID
	}
	if req.Priority != nil {
		existingConfig.Priority = *req.Priority
	}
	if req.IsActive != nil {
		existingConfig.IsActive = *req.IsActive
	}

	if err := h.repo.Update(existingConfig); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "更新存储配置失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(existingConfig))
}

// Delete 删除存储配置
func (h *StorageConfigHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "无效的配置ID"))
		return
	}

	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "删除存储配置失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"message": "存储配置删除成功",
	}))
}

// GetStorageTypes 获取支持的存储类型列表
func (h *StorageConfigHandler) GetStorageTypes(c *gin.Context) {
	types := []gin.H{
		{"value": model.StorageTypeSFTP, "label": "SFTP", "description": "SSH File Transfer Protocol"},
		{"value": model.StorageTypeWebDAV, "label": "WebDAV", "description": "Web分布式创作和版本控制"},
		{"value": model.StorageTypeAliyunOSS, "label": "阿里云OSS", "description": "阿里云对象存储"},
		{"value": model.StorageTypeTencentCOS, "label": "腾讯云COS", "description": "腾讯云对象存储"},
		{"value": model.StorageTypeAWSS3, "label": "AWS S3", "description": "Amazon Simple Storage Service"},
		{"value": model.StorageTypeSMB, "label": "SMB/CIFS", "description": "Windows网络共享"},
		{"value": model.StorageTypeLocal, "label": "本地存储", "description": "本地磁盘路径"},
	}

	c.JSON(http.StatusOK, model.Success(types))
}
