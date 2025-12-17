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

// StorageConfig 存储配置（数据库模型）
type StorageConfig struct {
	ID              int         `json:"id" db:"id"`
	Name            string      `json:"name" db:"name"`                         // 配置名称
	StorageType     StorageType `json:"storage_type" db:"storage_type"`         // 存储类型
	ConfigData      string      `json:"config_data" db:"config_data"`           // JSON配置数据
	IsActive        bool        `json:"is_active" db:"is_active"`               // 是否启用
	Description     string      `json:"description" db:"description"`           // 描述
	TargetAgentID   *string     `json:"target_agent_id" db:"target_agent_id"`   // 目标Agent ID（null为全局配置）
	Priority        int         `json:"priority" db:"priority"`                 // 优先级（数字越大优先级越高）
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at" db:"updated_at"`
}

// TableName 指定表名
func (StorageConfig) TableName() string {
	return "storage_configs"
}

// IsGlobal 是否为全局配置
func (s *StorageConfig) IsGlobal() bool {
	return s.TargetAgentID == nil || *s.TargetAgentID == ""
}

// === 具体存储配置结构（用于ConfigData JSON序列化） ===

// SFTPConfig SFTP配置
type SFTPConfig struct {
	Host       string `json:"host" binding:"required"`         // 服务器地址
	Port       int    `json:"port" binding:"required"`         // 端口
	Username   string `json:"username" binding:"required"`     // 用户名
	Password   string `json:"password"`                        // 密码
	KeyPath    string `json:"key_path"`                        // SSH私钥路径（与密码二选一）
	KeyContent string `json:"key_content"`                     // SSH私钥内容（与密码二选一）
	RemotePath string `json:"remote_path" binding:"required"`  // 远程路径
	Timeout    int    `json:"timeout"`                         // 超时时间（秒），默认30
}

// WebDAVConfig WebDAV配置
type WebDAVConfig struct {
	Endpoint   string `json:"endpoint" binding:"required"`     // WebDAV服务器地址
	Username   string `json:"username" binding:"required"`     // 用户名
	Password   string `json:"password" binding:"required"`     // 密码
	RemotePath string `json:"remote_path" binding:"required"`  // 远程路径
	UseHTTPS   bool   `json:"use_https"`                       // 是否使用HTTPS
	Timeout    int    `json:"timeout"`                         // 超时时间（秒），默认30
}

// AliyunOSSConfig 阿里云OSS配置
type AliyunOSSConfig struct {
	Endpoint        string `json:"endpoint" binding:"required"`         // 区域节点
	AccessKeyID     string `json:"access_key_id" binding:"required"`    // AccessKey ID
	AccessKeySecret string `json:"access_key_secret" binding:"required"` // AccessKey Secret
	BucketName      string `json:"bucket_name" binding:"required"`      // 存储桶名称
	BasePath        string `json:"base_path"`                           // 基础路径
	UseHTTPS        bool   `json:"use_https"`                           // 是否使用HTTPS
	Timeout         int    `json:"timeout"`                             // 超时时间（秒），默认30
}

// TencentCOSConfig 腾讯云COS配置
type TencentCOSConfig struct {
	Region          string `json:"region" binding:"required"`           // 区域
	SecretID        string `json:"secret_id" binding:"required"`        // SecretId
	SecretKey       string `json:"secret_key" binding:"required"`       // SecretKey
	BucketName      string `json:"bucket_name" binding:"required"`      // 存储桶名称（格式：BucketName-APPID）
	BasePath        string `json:"base_path"`                           // 基础路径
	UseHTTPS        bool   `json:"use_https"`                           // 是否使用HTTPS
	Timeout         int    `json:"timeout"`                             // 超时时间（秒），默认30
}

// AWSS3Config AWS S3配置
type AWSS3Config struct {
	Region          string `json:"region" binding:"required"`           // 区域
	AccessKeyID     string `json:"access_key_id" binding:"required"`    // Access Key ID
	SecretAccessKey string `json:"secret_access_key" binding:"required"` // Secret Access Key
	BucketName      string `json:"bucket_name" binding:"required"`      // 存储桶名称
	BasePath        string `json:"base_path"`                           // 基础路径
	UseHTTPS        bool   `json:"use_https"`                           // 是否使用HTTPS
	Endpoint        string `json:"endpoint"`                            // 自定义Endpoint（用于兼容S3的其他服务）
	Timeout         int    `json:"timeout"`                             // 超时时间（秒），默认30
}

// SMBConfig SMB/CIFS配置
type SMBConfig struct {
	Host       string `json:"host" binding:"required"`         // 服务器地址
	Port       int    `json:"port"`                            // 端口，默认445
	ShareName  string `json:"share_name" binding:"required"`   // 共享名称
	Username   string `json:"username"`                        // 用户名（可选，匿名访问时不需要）
	Password   string `json:"password"`                        // 密码
	Domain     string `json:"domain"`                          // 域（可选）
	RemotePath string `json:"remote_path" binding:"required"`  // 远程路径
	Timeout    int    `json:"timeout"`                         // 超时时间（秒），默认30
}

// LocalConfig 本地磁盘配置
type LocalConfig struct {
	BasePath string `json:"base_path" binding:"required"`    // 本地存储路径
	CreateDir bool  `json:"create_dir"`                      // 路径不存在时是否自动创建
}

// GetDefaultTimeout 获取默认超时时间
func GetDefaultTimeout(timeout int) int {
	if timeout <= 0 {
		return 30
	}
	return timeout
}
