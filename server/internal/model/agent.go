package model

import (
	"strings"
	"time"
)

// Agent Agent模型
type Agent struct {
	ID             int       `json:"id"`
	AgentID        string    `json:"agent_id" gorm:"uniqueIndex;size:64"`
	Email          string    `json:"email" gorm:"size:255"`
	EmailPrefix    string    `json:"email_prefix" gorm:"size:255"`
	Hostname       string    `json:"hostname" gorm:"size:255"`
	IPAddress      string    `json:"ip_address" gorm:"size:45"`
	OSVersion      string    `json:"os_version" gorm:"size:255"`
	AgentVersion   string    `json:"agent_version" gorm:"size:50"`
	Status         string    `json:"status" gorm:"size:20;default:'online'"`
	LastHeartbeat  *time.Time `json:"last_heartbeat"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Agent) TableName() string {
	return "agents"
}

// Heartbeat 心跳记录
type Heartbeat struct {
	ID         int       `json:"id"`
	AgentID    string    `json:"agent_id" gorm:"index;size:64"`
	Status     string    `json:"status" gorm:"size:20"`
	CPUUsage   float64   `json:"cpu_usage"`
	MemoryUsage int64    `json:"memory_usage"`
	DiskUsage  int64     `json:"disk_usage"`
	ScanCount  int       `json:"scan_count"`
	UploadCount int      `json:"upload_count"`
	ErrorCount int       `json:"error_count"`
	Timestamp  time.Time `json:"timestamp" gorm:"index"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName 指定表名
func (Heartbeat) TableName() string {
	return "heartbeats"
}

// AgentStatus Agent状态枚举
const (
	AgentStatusOnline  = "online"
	AgentStatusOffline = "offline"
)

// GetStatusColor 获取状态颜色（用于前端显示）
func (a *Agent) GetStatusColor() string {
	switch a.Status {
	case AgentStatusOnline:
		return "green"
	case AgentStatusOffline:
		return "red"
	default:
		return "gray"
	}
}

// IsOnline 检查Agent是否在线
func (a *Agent) IsOnline() bool {
	if a.LastHeartbeat == nil {
		return false
	}

	// 如果5分钟内没有心跳，认为离线
	threshold := time.Now().Add(-5 * time.Minute)
	return a.LastHeartbeat.After(threshold)
}

// GetEmailPrefix 从邮箱中提取前缀
func (a *Agent) GetEmailPrefix() string {
	if a.EmailPrefix != "" {
		return a.EmailPrefix
	}

	if a.Email == "" {
		return ""
	}

	// 提取@前面的部分
	parts := strings.Split(a.Email, "@")
	if len(parts) > 0 {
		return parts[0]
	}

	return a.Email
}
