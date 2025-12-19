package uploader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"doc-scanner-agent/internal/model"

	"github.com/studio-b12/gowebdav"
)

// WebDAVUploader WebDAV上传器（实现Uploader接口）
type WebDAVUploader struct {
	config *model.WebDAVConfig
	logger Logger
	client *gowebdav.Client
}

// NewWebDAVUploader 创建WebDAV上传器
func NewWebDAVUploader(config *model.WebDAVConfig, logger Logger) *WebDAVUploader {
	return &WebDAVUploader{
		config: config,
		logger: logger,
	}
}

// Connect 连接到WebDAV服务器
func (u *WebDAVUploader) Connect() error {
	u.logger.Info("正在连接到WebDAV服务器 %s...", u.config.Endpoint)

	// 创建WebDAV客户端
	client := gowebdav.NewClient(u.config.Endpoint, u.config.Username, u.config.Password)

	// 测试连接
	if err := client.Connect(); err != nil {
		return fmt.Errorf("连接WebDAV服务器失败: %w", err)
	}

	u.client = client
	u.logger.Info("WebDAV连接建立成功")
	return nil
}

// Disconnect 断开WebDAV连接
func (u *WebDAVUploader) Disconnect() error {
	// WebDAV客户端不需要显式断开连接
	u.client = nil
	u.logger.Info("WebDAV连接已关闭")
	return nil
}

// UploadFile 上传单个文件
func (u *WebDAVUploader) UploadFile(localPath, remotePath string) error {
	return u.UploadFileWithContext(context.Background(), localPath, remotePath)
}

// UploadFileWithContext 带上下文的文件上传
func (u *WebDAVUploader) UploadFileWithContext(ctx context.Context, localPath, remotePath string) error {
	if u.client == nil {
		return fmt.Errorf("WebDAV未连接")
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 构建完整的远程路径
	fullRemotePath := filepath.Join(u.config.RemotePath, remotePath)

	// 确保远程目录存在
	remoteDir := filepath.Dir(fullRemotePath)
	if err := u.client.MkdirAll(remoteDir, 0755); err != nil {
		return fmt.Errorf("创建远程目录 %s 失败: %w", remoteDir, err)
	}

	// 读取本地文件
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("读取本地文件失败: %w", err)
	}

	u.logger.Debug("正在上传 %s 到 %s (大小: %d 字节)", localPath, fullRemotePath, len(data))

	// 上传文件
	if err := u.client.Write(fullRemotePath, data, 0644); err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}

	u.logger.Info("成功上传 %s (大小: %d 字节)", fullRemotePath, len(data))
	return nil
}

// UploadFiles 批量上传文件
func (u *WebDAVUploader) UploadFiles(files map[string]string) error {
	for localPath, remotePath := range files {
		if err := u.UploadFile(localPath, remotePath); err != nil {
			u.logger.Error("上传文件 %s 失败: %v", localPath, err)
			return err
		}
	}
	return nil
}

// IsConnected 检查是否已连接
func (u *WebDAVUploader) IsConnected() bool {
	return u.client != nil
}

// GetType 获取存储类型
func (u *WebDAVUploader) GetType() string {
	return string(model.StorageTypeWebDAV)
}

// TestConnection 测试连接
func (u *WebDAVUploader) TestConnection() error {
	if err := u.Connect(); err != nil {
		return err
	}
	defer u.Disconnect()

	// 尝试读取根目录以验证连接和权限
	_, err := u.client.ReadDir(u.config.RemotePath)
	if err != nil {
		return fmt.Errorf("无法访问远程路径 %s: %w", u.config.RemotePath, err)
	}

	return nil
}
