package model

import (
	"fmt"
	"time"
)

// FileUpload 文件上传记录模型
type FileUpload struct {
	ID              int       `json:"id"`
	AgentID         string    `json:"agent_id" gorm:"index;size:64"`
	LocalPath       string    `json:"local_path" gorm:"size:1024"`
	RemotePath      string    `json:"remote_path" gorm:"size:1024"`
	FileName        string    `json:"file_name" gorm:"size:255"`
	FileSize        int64     `json:"file_size"`
	FileType        string    `json:"file_type" gorm:"size:50"`
	UploadStatus    string    `json:"upload_status" gorm:"size:20;index;default:'pending'"`
	ErrorMessage    string    `json:"error_message" gorm:"type:text"`
	UploadStartTime *time.Time `json:"upload_start_time"`
	UploadEndTime   *time.Time `json:"upload_end_time"`
	RetryCount      int       `json:"retry_count" gorm:"default:0"`
	CreatedAt       time.Time `json:"created_at"`
}

// TableName 指定表名
func (FileUpload) TableName() string {
	return "file_uploads"
}

// FileStatus 文件上传状态枚举
const (
	FileStatusPending = "pending"
	FileStatusUploading = "uploading"
	FileStatusSuccess = "success"
	FileStatusFailed = "failed"
)

// GetFileTypeIcon 获取文件类型图标
func (f *FileUpload) GetFileTypeIcon() string {
	switch f.FileType {
	case ".doc", ".docx":
		return "📄"
	case ".xls", ".xlsx":
		return "📊"
	case ".ppt", ".pptx":
		return "📈"
	case ".pdf":
		return "📕"
	case ".txt":
		return "📝"
	default:
		return "📦"
	}
}

// GetStatusColor 获取状态颜色
func (f *FileUpload) GetStatusColor() string {
	switch f.UploadStatus {
	case FileStatusSuccess:
		return "green"
	case FileStatusFailed:
		return "red"
	case FileStatusUploading:
		return "blue"
	default:
		return "gray"
	}
}

// GetDuration 获取上传耗时
func (f *FileUpload) GetDuration() time.Duration {
	if f.UploadStartTime == nil || f.UploadEndTime == nil {
		return 0
	}
	return f.UploadEndTime.Sub(*f.UploadStartTime)
}

// GetFormattedSize 获取格式化的大小
func (f *FileUpload) GetFormattedSize() string {
	return formatBytes(f.FileSize)
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	if bytes >= GB {
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	} else if bytes >= MB {
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	} else if bytes >= KB {
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	} else {
		return fmt.Sprintf("%d B", bytes)
	}
}
