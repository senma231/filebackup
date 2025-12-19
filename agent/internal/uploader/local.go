package uploader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"doc-scanner-agent/internal/model"
)

// LocalUploader 本地存储上传器（实现Uploader接口）
type LocalUploader struct {
	config *model.LocalConfig
	logger Logger
}

// NewLocalUploader 创建本地存储上传器
func NewLocalUploader(config *model.LocalConfig, logger Logger) *LocalUploader {
	return &LocalUploader{
		config: config,
		logger: logger,
	}
}

// Connect 验证本地存储路径
func (u *LocalUploader) Connect() error {
	u.logger.Info("正在验证本地存储路径 %s...", u.config.BasePath)

	// 检查路径是否存在
	info, err := os.Stat(u.config.BasePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 路径不存在
			if u.config.CreateDir {
				// 创建目录
				if err := os.MkdirAll(u.config.BasePath, 0755); err != nil {
					return fmt.Errorf("创建存储目录失败: %w", err)
				}
				u.logger.Info("已创建存储目录: %s", u.config.BasePath)
			} else {
				return fmt.Errorf("存储路径不存在: %s", u.config.BasePath)
			}
		} else {
			return fmt.Errorf("访问存储路径失败: %w", err)
		}
	} else {
		// 路径存在，检查是否为目录
		if !info.IsDir() {
			return fmt.Errorf("存储路径不是目录: %s", u.config.BasePath)
		}
	}

	// 检查读写权限（尝试创建临时文件）
	testFile := filepath.Join(u.config.BasePath, ".test_write_permission")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("存储路径无写入权限: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	u.logger.Info("本地存储路径验证成功")
	return nil
}

// Disconnect 断开连接（本地存储无需操作）
func (u *LocalUploader) Disconnect() error {
	u.logger.Info("本地存储上传器已关闭")
	return nil
}

// UploadFile 上传单个文件（复制到本地目录）
func (u *LocalUploader) UploadFile(localPath, remotePath string) error {
	return u.UploadFileWithContext(context.Background(), localPath, remotePath)
}

// UploadFileWithContext 带上下文的文件上传
func (u *LocalUploader) UploadFileWithContext(ctx context.Context, localPath, remotePath string) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 构建目标路径
	destPath := filepath.Join(u.config.BasePath, remotePath)

	// 确保目标目录存在
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 打开源文件
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 获取文件信息
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	u.logger.Debug("正在复制 %s 到 %s (大小: %d 字节)", localPath, destPath, srcInfo.Size())

	// 创建目标文件
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	// 复制文件内容（支持上下文取消）
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	var written int64
	for {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			// 删除未完成的目标文件
			destFile.Close()
			os.Remove(destPath)
			return ctx.Err()
		default:
		}

		nr, er := srcFile.Read(buf)
		if nr > 0 {
			nw, ew := destFile.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return fmt.Errorf("写入目标文件失败: %w", ew)
			}
			if nr != nw {
				return fmt.Errorf("写入字节数不匹配")
			}
		}
		if er != nil {
			if er != io.EOF {
				return fmt.Errorf("读取源文件失败: %w", er)
			}
			break
		}
	}

	// 保留源文件的权限
	if err := destFile.Chmod(srcInfo.Mode()); err != nil {
		u.logger.Warn("设置文件权限失败: %v", err)
	}

	u.logger.Info("成功复制文件到 %s (大小: %d 字节)", destPath, written)
	return nil
}

// UploadFiles 批量上传文件
func (u *LocalUploader) UploadFiles(files map[string]string) error {
	for localPath, remotePath := range files {
		if err := u.UploadFile(localPath, remotePath); err != nil {
			u.logger.Error("复制文件 %s 失败: %v", localPath, err)
			return err
		}
	}
	return nil
}

// IsConnected 检查是否已连接（本地存储始终连接）
func (u *LocalUploader) IsConnected() bool {
	return true
}

// GetType 获取存储类型
func (u *LocalUploader) GetType() string {
	return string(model.StorageTypeLocal)
}

// TestConnection 测试连接
func (u *LocalUploader) TestConnection() error {
	return u.Connect()
}
