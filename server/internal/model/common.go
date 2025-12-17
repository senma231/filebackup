package model

import (
	"time"
)

// SystemLog 系统日志模型
type SystemLog struct {
	ID        int       `json:"id"`
	LogLevel  string    `json:"log_level" gorm:"size:20;index"`
	Module    string    `json:"module" gorm:"size:50;index"`
	Message   string    `json:"message" gorm:"type:text"`
	AgentID   *string   `json:"agent_id" gorm:"index;size:64"`
	StackTrace string   `json:"stack_trace" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

// TableName 指定表名
func (SystemLog) TableName() string {
	return "system_logs"
}

// LogLevel 日志级别枚举
const (
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelDebug = "debug"
)

// GetLogLevelColor 获取日志级别颜色
func (l *SystemLog) GetLogLevelColor() string {
	switch l.LogLevel {
	case LogLevelInfo:
		return "blue"
	case LogLevelWarn:
		return "orange"
	case LogLevelError:
		return "red"
	case LogLevelDebug:
		return "gray"
	default:
		return "gray"
	}
}

// SystemStats 系统统计模型
type SystemStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage int64   `json:"memory_usage"`
	DiskUsage   int64   `json:"disk_usage"`
	ScanCount   int     `json:"scan_count"`
	UploadCount int     `json:"upload_count"`
	ErrorCount  int     `json:"error_count"`
}

// FileStats 文件统计模型
type FileStats struct {
	ID          int       `json:"id"`
	AgentID     string    `json:"agent_id" gorm:"index;size:64"`
	StatDate    string    `json:"stat_date" gorm:"size:10;index"`
	TotalFiles  int       `json:"total_files" gorm:"default:0"`
	TotalSize   int64     `json:"total_size" gorm:"default:0"`
	DocCount    int       `json:"doc_count" gorm:"default:0"`
	ExcelCount  int       `json:"excel_count" gorm:"default:0"`
	PPTCount    int       `json:"ppt_count" gorm:"default:0"`
	PDFCount    int       `json:"pdf_count" gorm:"default:0"`
	TXTCount    int       `json:"txt_count" gorm:"default:0"`
	UploadedCount int     `json:"uploaded_count" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (FileStats) TableName() string {
	return "file_stats"
}

// GetUploadRate 获取上传率
func (s *FileStats) GetUploadRate() float64 {
	if s.TotalFiles == 0 {
		return 0
	}
	return float64(s.UploadedCount) / float64(s.TotalFiles) * 100
}

// GetTotalFormattedSize 获取格式化总大小
func (s *FileStats) GetTotalFormattedSize() string {
	return formatBytes(s.TotalSize)
}

// APIResponse API响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 创建成功响应
func Success(data interface{}) *APIResponse {
	return &APIResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}
}

// Error 创建错误响应
func Error(code int, message string) *APIResponse {
	return &APIResponse{
		Code:    code,
		Message: message,
	}
}

// PaginatedResponse 分页响应结构
type PaginatedResponse struct {
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Data    interface{} `json:"data"`
}

// NewPaginatedResponse 创建分页响应
func NewPaginatedResponse(total int64, page, perPage int, data interface{}) *PaginatedResponse {
	return &PaginatedResponse{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Data:    data,
	}
}
