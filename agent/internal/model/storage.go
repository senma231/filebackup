package model

import "time"

// StorageType 存储类型
type StorageType string

const (
	StorageTypeSFTP       StorageType = "sftp"
	StorageTypeWebDAV     StorageType = "webdav"
	StorageTypeAliyunOSS  StorageType = "aliyun_oss"
	StorageTypeTencentCOS StorageType = "tencent_cos"
	StorageTypeAWSS3      StorageType = "aws_s3"
	StorageTypeSMB        StorageType = "smb"
	StorageTypeLocal      StorageType = "local"
)

// StorageConfig 存储配置（从Server获取）
type StorageConfig struct {
	ID            int         `json:"id"`
	Name          string      `json:"name"`
	StorageType   StorageType `json:"storage_type"`
	ConfigData    string      `json:"config_data"` // JSON格式的配置数据
	IsActive      bool        `json:"is_active"`
	Description   string      `json:"description"`
	TargetAgentID *string     `json:"target_agent_id"`
	Priority      int         `json:"priority"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// === 具体存储配置结构（用于解析ConfigData） ===

// SFTPConfig SFTP配置
type SFTPConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	KeyPath    string `json:"key_path"`
	KeyContent string `json:"key_content"`
	RemotePath string `json:"remote_path"`
	Timeout    int    `json:"timeout"`
}

// WebDAVConfig WebDAV配置
type WebDAVConfig struct {
	Endpoint   string `json:"endpoint"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	RemotePath string `json:"remote_path"`
	UseHTTPS   bool   `json:"use_https"`
	Timeout    int    `json:"timeout"`
}

// AliyunOSSConfig 阿里云OSS配置
type AliyunOSSConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	BucketName      string `json:"bucket_name"`
	BasePath        string `json:"base_path"`
	UseHTTPS        bool   `json:"use_https"`
	Timeout         int    `json:"timeout"`
}

// TencentCOSConfig 腾讯云COS配置
type TencentCOSConfig struct {
	Region     string `json:"region"`
	SecretID   string `json:"secret_id"`
	SecretKey  string `json:"secret_key"`
	BucketName string `json:"bucket_name"`
	BasePath   string `json:"base_path"`
	UseHTTPS   bool   `json:"use_https"`
	Timeout    int    `json:"timeout"`
}

// AWSS3Config AWS S3配置
type AWSS3Config struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	BucketName      string `json:"bucket_name"`
	BasePath        string `json:"base_path"`
	UseHTTPS        bool   `json:"use_https"`
	Endpoint        string `json:"endpoint"`
	Timeout         int    `json:"timeout"`
}

// SMBConfig SMB/CIFS配置
type SMBConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	ShareName  string `json:"share_name"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Domain     string `json:"domain"`
	RemotePath string `json:"remote_path"`
	Timeout    int    `json:"timeout"`
}

// LocalConfig 本地磁盘配置
type LocalConfig struct {
	BasePath  string `json:"base_path"`
	CreateDir bool   `json:"create_dir"`
}
