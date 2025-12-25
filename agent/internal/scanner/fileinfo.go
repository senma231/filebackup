package scanner

import (
	"os"
	"time"
)

// FileInfo 文件信息结构
type FileInfo struct {
	Path         string    `json:"path"`          // 文件完整路径
	Name         string    `json:"name"`          // 文件名
	Extension    string    `json:"extension"`     // 文件扩展名
	Size         int64     `json:"size"`          // 文件大小(字节)
	ModifiedTime time.Time `json:"modified_time"` // 修改时间
	IsDirectory  bool      `json:"is_directory"`  // 是否为目录
}

// NewFileInfo 创建新的文件信息
func NewFileInfo(path string, stat os.FileInfo) *FileInfo {
	return &FileInfo{
		Path:         path,
		Name:         stat.Name(),
		Extension:    getExtension(path),
		Size:         stat.Size(),
		ModifiedTime: stat.ModTime(),
		IsDirectory:  stat.IsDir(),
	}
}

// getExtension 获取文件扩展名
func getExtension(path string) string {
	// 简单实现，实际应该使用 filepath.Ext
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			if i < len(path)-1 {
				return path[i:]
			}
			break
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}

// GetType 获取文件类型
func (f *FileInfo) GetType() string {
	switch f.Extension {
	case ".doc", ".docx":
		return "word"
	case ".xls", ".xlsx":
		return "excel"
	case ".ppt", ".pptx":
		return "powerpoint"
	case ".pdf":
		return "pdf"
	case ".txt":
		return "text"
	default:
		return "unknown"
	}
}

// ShouldExclude 检查文件是否应该被排除
func (f *FileInfo) ShouldExclude(patterns []string) bool {
	for _, pattern := range patterns {
		if matchesPattern(f.Name, pattern) {
			return true
		}
	}
	return false
}

// matchesPattern 简单的模式匹配
func matchesPattern(filename, pattern string) bool {
	// 处理前缀模式（如 "._*" - 匹配以._开头的文件）
	if len(pattern) >= 2 && pattern[0] == '.' && pattern[1] == '_' {
		return len(filename) >= 2 && filename[0] == '.' && filename[1] == '_'
	}

	// 处理后缀模式（如 "*.log" - 匹配以.log结尾的文件）
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(filename) >= len(suffix) && filename[len(filename)-len(suffix):] == suffix
	}

	// 精确匹配
	return filename == pattern
}
