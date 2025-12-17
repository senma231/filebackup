package uploader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
)

// SFTPClient SFTP客户端
type SFTPClient struct {
	config *Config
	logger Logger
	client *ssh.Client
	sftp   SFTPConn
}

// Config 上传配置
type Config struct {
	Host       string
	Port       int
	Username   string
	Password   string
	KeyPath    string
	RemotePath string
	MaxRetries int
	RetryDelay time.Duration
}

// Logger 日志接口
type Logger interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

// SFTPConn SFTP连接接口
type SFTPConn interface {
	Put(src, dst string) error
	MkdirAll(path string) error
	Remove(path string) error
	Close() error
}

// NewSFTPClient 创建新的SFTP客户端
func NewSFTPClient(cfg *Config, logger Logger) *SFTPClient {
	return &SFTPClient{
		config: cfg,
		logger: logger,
	}
}

// Connect 连接到SFTP服务器
func (c *SFTPClient) Connect() error {
	c.logger.Info("Connecting to SFTP server %s:%d...", c.config.Host, c.config.Port)

	// 构建SSH配置
	var auth ssh.AuthMethod
	if c.config.KeyPath != "" {
		// 使用私钥认证
		key, err := os.ReadFile(c.config.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		auth = ssh.PublicKeys(signer)
	} else {
		// 使用密码认证
		auth = ssh.Password(c.config.Password)
	}

	// 创建SSH客户端
	config := &ssh.ClientConfig{
		User: c.config.Username,
		Auth: []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证主机密钥
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	c.client = client

	// 创建SFTP会话
	sftp, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}

	c.sftp = &RealSFTPConn{sftp}
	c.logger.Info("SFTP connection established")
	return nil
}

// Disconnect 断开SFTP连接
func (c *SFTPClient) Disconnect() error {
	if c.sftp != nil {
		if err := c.sftp.Close(); err != nil {
			c.logger.Warn("Error closing SFTP connection: %v", err)
		}
	}

	if c.client != nil {
		if err := c.client.Close(); err != nil {
			c.logger.Warn("Error closing SSH client: %v", err)
		}
	}

	c.logger.Info("SFTP connection closed")
	return nil
}

// UploadFile 上传文件
func (c *SFTPClient) UploadFile(localPath, remotePath string) error {
	// 确保远程目录存在
	remoteDir := filepath.Dir(remotePath)
	if err := c.sftp.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("failed to create remote directory %s: %w", remoteDir, err)
	}

	// 上传文件
	c.logger.Debug("Uploading %s to %s", localPath, remotePath)
	if err := c.sftp.Put(localPath, remotePath); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	c.logger.Info("Successfully uploaded %s", remotePath)
	return nil
}

// UploadFiles 批量上传文件
func (c *SFTPClient) UploadFiles(files map[string]string) error {
	for localPath, remotePath := range files {
		if err := c.UploadFile(localPath, remotePath); err != nil {
			c.logger.Error("Failed to upload %s: %v", localPath, err)
			return err
		}
	}
	return nil
}

// IsConnected 检查连接状态
func (c *SFTPClient) IsConnected() bool {
	return c.client != nil && c.client.Conn != nil
}

// RealSFTPConn 真实SFTP连接实现
type RealSFTPConn struct {
	*sftp.Client
}

func (c *RealSFTPConn) Put(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := c.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (c *RealSFTPConn) MkdirAll(path string) error {
	return c.MkdirAll(path)
}

func (c *RealSFTPConn) Remove(path string) error {
	return c.Remove(path)
}
