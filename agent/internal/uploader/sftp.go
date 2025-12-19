package uploader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"doc-scanner-agent/internal/model"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPUploader SFTP上传器（实现Uploader接口）
type SFTPUploader struct {
	config     *model.SFTPConfig
	logger     Logger
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// NewSFTPUploader 创建SFTP上传器
func NewSFTPUploader(config *model.SFTPConfig, logger Logger) *SFTPUploader {
	return &SFTPUploader{
		config: config,
		logger: logger,
	}
}

// Connect 连接到SFTP服务器
func (u *SFTPUploader) Connect() error {
	u.logger.Info("正在连接到SFTP服务器 %s:%d...", u.config.Host, u.config.Port)

	// 构建SSH配置
	var auth ssh.AuthMethod

	// 优先使用密钥认证
	if u.config.KeyContent != "" {
		// 使用密钥内容
		signer, err := ssh.ParsePrivateKey([]byte(u.config.KeyContent))
		if err != nil {
			return fmt.Errorf("解析私钥内容失败: %w", err)
		}
		auth = ssh.PublicKeys(signer)
	} else if u.config.KeyPath != "" {
		// 使用密钥文件
		key, err := os.ReadFile(u.config.KeyPath)
		if err != nil {
			return fmt.Errorf("读取私钥文件失败: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("解析私钥失败: %w", err)
		}
		auth = ssh.PublicKeys(signer)
	} else {
		// 使用密码认证
		auth = ssh.Password(u.config.Password)
	}

	// 设置超时时间
	timeout := time.Duration(u.config.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 创建SSH客户端配置
	sshConfig := &ssh.ClientConfig{
		User: u.config.Username,
		Auth: []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境应该验证主机密钥
		Timeout:         timeout,
	}

	// 连接SSH服务器
	addr := fmt.Sprintf("%s:%d", u.config.Host, u.config.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("连接SSH服务器失败: %w", err)
	}
	u.sshClient = sshClient

	// 创建SFTP会话
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return fmt.Errorf("创建SFTP会话失败: %w", err)
	}
	u.sftpClient = sftpClient

	u.logger.Info("SFTP连接建立成功")
	return nil
}

// Disconnect 断开SFTP连接
func (u *SFTPUploader) Disconnect() error {
	if u.sftpClient != nil {
		if err := u.sftpClient.Close(); err != nil {
			u.logger.Warn("关闭SFTP会话时出错: %v", err)
		}
		u.sftpClient = nil
	}

	if u.sshClient != nil {
		if err := u.sshClient.Close(); err != nil {
			u.logger.Warn("关闭SSH客户端时出错: %v", err)
		}
		u.sshClient = nil
	}

	u.logger.Info("SFTP连接已关闭")
	return nil
}

// UploadFile 上传单个文件
func (u *SFTPUploader) UploadFile(localPath, remotePath string) error {
	return u.UploadFileWithContext(context.Background(), localPath, remotePath)
}

// UploadFileWithContext 带上下文的文件上传
func (u *SFTPUploader) UploadFileWithContext(ctx context.Context, localPath, remotePath string) error {
	if u.sftpClient == nil {
		return fmt.Errorf("SFTP未连接")
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
	if err := u.sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("创建远程目录 %s 失败: %w", remoteDir, err)
	}

	// 打开本地文件
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建远程文件
	dstFile, err := u.sftpClient.Create(fullRemotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer dstFile.Close()

	// 获取文件大小
	fileInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	u.logger.Debug("正在上传 %s 到 %s (大小: %d 字节)", localPath, fullRemotePath, fileInfo.Size())

	// 复制文件内容（支持上下文取消）
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	var written int64
	for {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		nr, er := srcFile.Read(buf)
		if nr > 0 {
			nw, ew := dstFile.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return fmt.Errorf("写入远程文件失败: %w", ew)
			}
			if nr != nw {
				return fmt.Errorf("写入字节数不匹配")
			}
		}
		if er != nil {
			if er != io.EOF {
				return fmt.Errorf("读取本地文件失败: %w", er)
			}
			break
		}
	}

	u.logger.Info("成功上传 %s (大小: %d 字节)", fullRemotePath, written)
	return nil
}

// UploadFiles 批量上传文件
func (u *SFTPUploader) UploadFiles(files map[string]string) error {
	for localPath, remotePath := range files {
		if err := u.UploadFile(localPath, remotePath); err != nil {
			u.logger.Error("上传文件 %s 失败: %v", localPath, err)
			return err
		}
	}
	return nil
}

// IsConnected 检查是否已连接
func (u *SFTPUploader) IsConnected() bool {
	return u.sshClient != nil && u.sftpClient != nil
}

// GetType 获取存储类型
func (u *SFTPUploader) GetType() string {
	return string(model.StorageTypeSFTP)
}

// TestConnection 测试连接
func (u *SFTPUploader) TestConnection() error {
	if err := u.Connect(); err != nil {
		return err
	}
	defer u.Disconnect()

	// 尝试列出根目录以验证连接和权限
	_, err := u.sftpClient.ReadDir(u.config.RemotePath)
	if err != nil {
		return fmt.Errorf("无法访问远程路径 %s: %w", u.config.RemotePath, err)
	}

	return nil
}
