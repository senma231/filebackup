package uploader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"doc-scanner-agent/internal/model"
)

// LocalUploader 本地存储上传器（通过HTTP API上传到Server）
type LocalUploader struct {
	config    *model.LocalConfig
	logger    Logger
	serverURL string
	agentID   string
	client    *http.Client
}

// NewLocalUploader 创建本地存储上传器
func NewLocalUploader(config *model.LocalConfig, serverURL, agentID string, logger Logger) *LocalUploader {
	return &LocalUploader{
		config:    config,
		logger:    logger,
		serverURL: serverURL,
		agentID:   agentID,
		client: &http.Client{
			Timeout: 0, // 不设置超时，因为大文件上传可能需要较长时间
		},
	}
}

// Connect 测试与Server的连接
func (u *LocalUploader) Connect() error {
	u.logger.Info("正在测试与Server的连接...")

	// 测试连接（访问健康检查endpoint）
	healthURL := fmt.Sprintf("%s/health", u.serverURL)
	resp, err := u.client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("无法连接到Server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Server响应异常，状态码: %d", resp.StatusCode)
	}

	u.logger.Info("成功连接到Server")
	return nil
}

// Disconnect 断开连接
func (u *LocalUploader) Disconnect() error {
	u.logger.Info("本地存储上传器已关闭")
	return nil
}

// UploadFile 上传单个文件（通过HTTP API）
func (u *LocalUploader) UploadFile(localPath, remotePath string) error {
	return u.UploadFileWithContext(context.Background(), localPath, remotePath)
}

// UploadFileWithContext 带上下文的文件上传（通过HTTP API）
func (u *LocalUploader) UploadFileWithContext(ctx context.Context, localPath, remotePath string) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 打开本地文件
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("无法获取文件信息: %w", err)
	}

	u.logger.Info("正在上传文件到Server: %s (大小: %d 字节)", filepath.Base(localPath), fileInfo.Size())

	// 创建multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加agent_id字段
	if err := writer.WriteField("agent_id", u.agentID); err != nil {
		return fmt.Errorf("无法写入agent_id字段: %w", err)
	}

	// 添加remote_path字段
	if err := writer.WriteField("remote_path", remotePath); err != nil {
		return fmt.Errorf("无法写入remote_path字段: %w", err)
	}

	// 添加文件字段
	part, err := writer.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return fmt.Errorf("无法创建文件字段: %w", err)
	}

	// 复制文件内容到multipart form
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("无法复制文件内容: %w", err)
	}

	// 关闭multipart writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("无法关闭multipart writer: %w", err)
	}

	// 创建HTTP请求
	uploadURL := fmt.Sprintf("%s/api/v1/files/upload", u.serverURL)
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		return fmt.Errorf("无法创建HTTP请求: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送HTTP请求
	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Server返回错误 (状态码: %d): %s", resp.StatusCode, string(respBody))
	}

	u.logger.Info("成功上传文件到Server: %s", filepath.Base(localPath))
	return nil
}

// UploadFiles 批量上传文件
func (u *LocalUploader) UploadFiles(files map[string]string) error {
	for localPath, remotePath := range files {
		if err := u.UploadFile(localPath, remotePath); err != nil {
			u.logger.Error("上传文件 %s 失败: %v", localPath, err)
			return err
		}
	}
	return nil
}

// IsConnected 检查是否已连接
func (u *LocalUploader) IsConnected() bool {
	// 简单测试：尝试访问健康检查endpoint
	healthURL := fmt.Sprintf("%s/health", u.serverURL)
	resp, err := u.client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetType 获取存储类型
func (u *LocalUploader) GetType() string {
	return string(model.StorageTypeLocal)
}

// TestConnection 测试连接
func (u *LocalUploader) TestConnection() error {
	return u.Connect()
}
