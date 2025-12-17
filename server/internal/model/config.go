package model

import (
	"time"
)

// Config 配置模型
type Config struct {
	ID              int       `json:"id"`
	ConfigName      string    `json:"config_name" gorm:"size:255"`
	ConfigType      string    `json:"config_type" gorm:"size:50;index"`
	TargetAgentID   *string   `json:"target_agent_id" gorm:"index;size:64"`
	ConfigData      string    `json:"config_data" gorm:"type:text"`
	Version         int       `json:"version" gorm:"default:1"`
	IsActive        bool      `json:"is_active" gorm:"default:true"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Config) TableName() string {
	return "configs"
}

// ConfigType 配置类型枚举
const (
	ConfigTypeGlobal = "global"
	ConfigTypeAgent  = "agent"
)

// IsGlobal 检查是否为全局配置
func (c *Config) IsGlobal() bool {
	return c.ConfigType == ConfigTypeGlobal
}

// IsAgentSpecific 检查是否为Agent专属配置
func (c *Config) IsAgentSpecific() bool {
	return c.ConfigType == ConfigTypeAgent && c.TargetAgentID != nil
}

// GetEffectiveName 获取配置显示名称
func (c *Config) GetEffectiveName() string {
	if c.IsGlobal() {
		return c.ConfigName + " (全局)"
	}
	return c.ConfigName
}

// UpdateVersion 更新配置版本
func (c *Config) UpdateVersion() {
	c.Version++
}
