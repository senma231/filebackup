package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// ConfigHandler 配置处理器
type ConfigHandler struct {
	repo *repository.ConfigRepository
}

// NewConfigHandler 创建新的配置处理器
func NewConfigHandler(db *sql.DB) *ConfigHandler {
	return &ConfigHandler{
		repo: repository.NewConfigRepository(db),
	}
}

// Create 创建配置
func (h *ConfigHandler) Create(c *gin.Context) {
	var req struct {
		ConfigName      string `json:"config_name" binding:"required"`
		ConfigType      string `json:"config_type" binding:"required"`
		TargetAgentID   string `json:"target_agent_id"`
		ConfigData      string `json:"config_data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, err.Error()))
		return
	}

	// 验证配置类型
	if req.ConfigType != model.ConfigTypeGlobal && req.ConfigType != model.ConfigTypeAgent {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "Invalid config type"))
		return
	}

	// 验证配置数据格式
	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(req.ConfigData), &configData); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "Invalid config data format"))
		return
	}

	config := &model.Config{
		ConfigName:      req.ConfigName,
		ConfigType:      req.ConfigType,
		TargetAgentID:   stringPtr(req.TargetAgentID),
		ConfigData:      req.ConfigData,
		Version:         1,
		IsActive:        true,
	}

	if err := h.repo.Create(config); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to create config"))
		return
	}

	c.JSON(http.StatusOK, model.Success(config))
}

// GetAll 获取所有配置
func (h *ConfigHandler) GetAll(c *gin.Context) {
	configType := c.DefaultQuery("type", "")

	configs, err := h.repo.GetAll(configType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get configs"))
		return
	}

	c.JSON(http.StatusOK, model.Success(configs))
}

// GetByID 根据ID获取配置
func (h *ConfigHandler) GetByID(c *gin.Context) {
	configID := parseInt(c.Param("config_id"))

	config, err := h.repo.GetByID(configID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get config"))
		return
	}

	if config == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Config not found"))
		return
	}

	c.JSON(http.StatusOK, model.Success(config))
}

// Update 更新配置
func (h *ConfigHandler) Update(c *gin.Context) {
	configID := parseInt(c.Param("config_id"))

	var req struct {
		ConfigName  string `json:"config_name"`
		ConfigData  string `json:"config_data"`
		IsActive    *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, err.Error()))
		return
	}

	// 获取现有配置
	config, err := h.repo.GetByID(configID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get config"))
		return
	}

	if config == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Config not found"))
		return
	}

	// 更新字段
	if req.ConfigName != "" {
		config.ConfigName = req.ConfigName
	}

	if req.ConfigData != "" {
		// 验证配置数据格式
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(req.ConfigData), &configData); err != nil {
			c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "Invalid config data format"))
			return
		}
		config.ConfigData = req.ConfigData
		config.UpdateVersion()
	}

	if req.IsActive != nil {
		config.IsActive = *req.IsActive
	}

	if err := h.repo.Update(config); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to update config"))
		return
	}

	c.JSON(http.StatusOK, model.Success(config))
}

// Delete 删除配置
func (h *ConfigHandler) Delete(c *gin.Context) {
	configID := parseInt(c.Param("config_id"))

	if err := h.repo.Delete(configID); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to delete config"))
		return
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"message": "Config deleted successfully",
	}))
}

// GetByAgentID 获取Agent配置
func (h *ConfigHandler) GetByAgentID(c *gin.Context) {
	agentID := c.Param("agent_id")

	config, err := h.repo.GetByAgentID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agent config"))
		return
	}

	if config == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Config not found"))
		return
	}

	c.JSON(http.StatusOK, model.Success(config))
}

// 辅助函数

// stringPtr 将字符串转换为指针
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
