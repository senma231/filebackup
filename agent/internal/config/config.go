package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config Agent配置结构
type Config struct {
	// 扫描配置
	ScanPaths     []string `json:"scan_paths"`      // 扫描路径列表
	FileTypes     []string `json:"file_types"`      // 文件类型
	ExcludePatterns []string `json:"exclude_patterns"` // 排除模式
	MaxFileSize   int64    `json:"max_file_size"`   // 最大文件大小(字节)
	IncrementalScan bool   `json:"incremental_scan"` // 是否增量扫描

	// 用户配置
	Email       string `json:"email"`           // 用户邮箱地址
	EmailPrefix string `json:"email_prefix"`    // 邮箱前缀（用于文件夹命名）

	// Server配置
	ServerURL    string `json:"server_url"`     // Server地址
	AgentID      string `json:"agent_id"`       // Agent唯一标识
	Token        string `json:"token"`          // 认证令牌
	HeartbeatInterval int `json:"heartbeat_interval"` // 心跳间隔(秒)

	// SFTP配置（已废弃，保留仅为向后兼容，实际存储配置由Server端管理）
	SFTPConfig SFTPConfig `json:"sftp_config,omitempty"`

	// 系统配置
	ServiceMode bool   `json:"service_mode"` // 是否Windows服务模式
	LogLevel    string `json:"log_level"`    // 日志级别
	LogPath     string `json:"log_path"`     // 日志路径
	CompressEnabled bool `json:"compress_enabled"` // 是否启用压缩

	// 加密配置
	EncryptConfig bool `json:"encrypt_config"` // 是否加密配置
}

// SFTPConfig SFTP配置
type SFTPConfig struct {
	Host     string `json:"host"`     // 服务器地址
	Port     int    `json:"port"`     // 端口
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码(或私钥路径)
	KeyPath  string `json:"key_path"` // SSH私钥路径
	RemotePath string `json:"remote_path"` // 远程路径
}

// DefaultConfig 获取默认配置
func DefaultConfig() *Config {
	return &Config{
		ScanPaths: []string{
			getDefaultDocumentsPath(),
		},
		FileTypes: []string{
			".doc", ".docx",
			".xls", ".xlsx",
			".ppt", ".pptx",
			".pdf",
			".txt",
		},
		ExcludePatterns: []string{
			"*.log",
			"*.tmp",
			"~$*",
		},
		MaxFileSize:   100 * 1024 * 1024, // 100MB
		IncrementalScan: true,
		Email:         "",
		EmailPrefix:   "",
		ServerURL:     "http://localhost:8889",
		HeartbeatInterval: 30,
		LogLevel:      "info",
		LogPath:       getDefaultLogPath(),
		CompressEnabled: true,
		EncryptConfig: false,
	}
}

// getDefaultDocumentsPath 获取默认文档路径
func getDefaultDocumentsPath() string {
	// Windows下的常见文档路径
	userDir, err := os.UserHomeDir()
	if err != nil {
		return "C:\\Users\\Public\\Documents"
	}
	return filepath.Join(userDir, "Documents")
}

// getDefaultLogPath 获取默认日志路径
func getDefaultLogPath() string {
	userDir, err := os.UserHomeDir()
	if err != nil {
		return "C:\\ProgramData\\DocScannerAgent\\logs"
	}
	return filepath.Join(userDir, "AppData", "Local", "DocScannerAgent", "logs")
}

// Validate 验证配置
func (c *Config) Validate() error {
	if len(c.ScanPaths) == 0 {
		return fmt.Errorf("scan_paths cannot be empty")
	}

	if c.ServerURL == "" {
		return fmt.Errorf("server_url cannot be empty")
	}

	// 注意：存储配置（SFTP/WebDAV/本地等）由 Server 端统一管理
	// Agent 启动后会自动从 Server 获取存储配置，因此不再验证 SFTPConfig

	if c.HeartbeatInterval <= 0 {
		return fmt.Errorf("heartbeat_interval must be greater than 0")
	}

	if c.MaxFileSize <= 0 {
		return fmt.Errorf("max_file_size must be greater than 0")
	}

	return nil
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	userDir, err := os.UserHomeDir()
	if err != nil {
		return "C:\\ProgramData\\DocScannerAgent\\config.json"
	}
	return filepath.Join(userDir, "AppData", "Local", "DocScannerAgent", "config.json")
}

// GetLogPath 获取日志目录
func (c *Config) GetLogPath() string {
	if c.LogPath == "" {
		return getDefaultLogPath()
	}
	return c.LogPath
}

// GetSFTPConfig 获取SFTP配置
func (c *Config) GetSFTPConfig() SFTPConfig {
	return c.SFTPConfig
}

// GetEmailPrefix 从邮箱中提取前缀
func (c *Config) GetEmailPrefix() string {
	if c.EmailPrefix != "" {
		return c.EmailPrefix
	}

	if c.Email == "" {
		return ""
	}

	// 提取@前面的部分
	parts := strings.Split(c.Email, "@")
	if len(parts) > 0 {
		return parts[0]
	}

	return c.Email
}

// SetEmail 设置邮箱并自动提取前缀
func (c *Config) SetEmail(email string) {
	c.Email = email
	c.EmailPrefix = GetEmailPrefixFromEmail(email)
}

// GetEmailPrefixFromEmail 从邮箱字符串中提取前缀
func GetEmailPrefixFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}
