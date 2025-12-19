package uploader

import "context"

// Uploader 通用上传器接口
// 所有存储上传器（SFTP、WebDAV、OSS、COS、S3、SMB、Local）都需要实现此接口
type Uploader interface {
	// Connect 连接到存储服务
	// 对于本地存储，此方法主要用于验证路径
	Connect() error

	// Disconnect 断开与存储服务的连接
	// 释放资源，关闭连接
	Disconnect() error

	// UploadFile 上传单个文件
	// localPath: 本地文件路径
	// remotePath: 远程文件路径（相对于存储根目录）
	UploadFile(localPath, remotePath string) error

	// UploadFileWithContext 带上下文的文件上传（支持取消）
	// ctx: 上下文对象，可用于取消操作
	// localPath: 本地文件路径
	// remotePath: 远程文件路径
	UploadFileWithContext(ctx context.Context, localPath, remotePath string) error

	// UploadFiles 批量上传文件
	// files: 本地路径到远程路径的映射
	UploadFiles(files map[string]string) error

	// IsConnected 检查是否已连接到存储服务
	IsConnected() bool

	// GetType 获取存储类型
	// 返回存储类型标识（sftp、webdav、oss等）
	GetType() string

	// TestConnection 测试连接是否正常
	// 用于健康检查和配置验证
	TestConnection() error
}

// UploadProgress 上传进度回调
type UploadProgress struct {
	FileName      string // 文件名
	BytesUploaded int64  // 已上传字节数
	TotalBytes    int64  // 总字节数
	Percentage    int    // 完成百分比
	Error         error  // 错误信息（如果有）
}

// ProgressCallback 上传进度回调函数类型
type ProgressCallback func(progress UploadProgress)

// UploaderWithProgress 支持进度回调的上传器接口（可选实现）
type UploaderWithProgress interface {
	Uploader

	// UploadFileWithProgress 上传文件并报告进度
	UploadFileWithProgress(localPath, remotePath string, callback ProgressCallback) error
}
