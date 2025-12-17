package uploader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"doc-scanner-agent/internal/config"
	"doc-scanner-agent/internal/scanner"
)

// Transfer 文件传输器
type Transfer struct {
	agentConfig *config.Config // Agent配置
	sftpConfig  *Config        // SFTP配置
	logger      Logger
	sftp        *SFTPClient
	files       map[string]string // 本地路径 -> 远程路径
	mu          sync.RWMutex
	inFlight    map[string]*UploadTask // 正在上传的任务
}

// UploadTask 上传任务
type UploadTask struct {
	LocalPath  string
	RemotePath string
	Status     string // pending/uploading/success/failed
	StartTime  time.Time
	EndTime    time.Time
	Error      error
	Progress   int // 0-100
}

// NewTransfer 创建新的文件传输器
func NewTransfer(agentConfig *config.Config, sftpConfig *Config, logger Logger) *Transfer {
	return &Transfer{
		agentConfig: agentConfig,
		sftpConfig:  sftpConfig,
		logger:      logger,
		files:       make(map[string]string),
		inFlight:    make(map[string]*UploadTask),
	}
}

// Start 启动文件传输器
func (t *Transfer) Start(ctx context.Context) error {
	t.logger.Info("Starting file uploader...")

	// 连接到SFTP服务器，支持重试
	var connectErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := t.connect(); err != nil {
			connectErr = err
			t.logger.Warn("SFTP connection attempt %d failed: %v, retrying...", attempt, err)
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*2) * time.Second)
			}
		} else {
			connectErr = nil
			break
		}
	}

	if connectErr != nil {
		return fmt.Errorf("failed to connect to SFTP server after retries: %w", connectErr)
	}

	// 启动上传协程
	go t.uploadWorker(ctx)

	t.logger.Info("File uploader started")
	return nil
}

// Stop 停止文件传输器
func (t *Transfer) Stop() error {
	t.logger.Info("Stopping file uploader...")

	if t.sftp != nil {
		t.sftp.Disconnect()
	}

	t.logger.Info("File uploader stopped")
	return nil
}

// AddFiles 添加要上传的文件
func (t *Transfer) AddFiles(files []*scanner.FileInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, file := range files {
		// 生成远程路径
		remotePath := t.generateRemotePath(file)

		// 检查文件是否已存在
		if _, exists := t.files[file.Path]; exists {
			continue
		}

		t.files[file.Path] = remotePath
		t.logger.Debug("Added file for upload: %s -> %s", file.Path, remotePath)
	}
}

// GetUploadStatus 获取上传状态
func (t *Transfer) GetUploadStatus() map[string]*UploadTask {
	t.mu.RLock()
	defer t.mu.RUnlock()

	status := make(map[string]*UploadTask)
	for path, task := range t.inFlight {
		status[path] = task
	}

	return status
}

// GetUploadStats 获取上传统计
func (t *Transfer) GetUploadStats() (int, int, int, int64) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var pending, uploading, success, failed int
	var totalSize int64

	for _, task := range t.inFlight {
		switch task.Status {
		case "pending":
			pending++
		case "uploading":
			uploading++
		case "success":
			success++
		case "failed":
			failed++
		}

		if stat, err := os.Stat(task.LocalPath); err == nil {
			totalSize += stat.Size()
		}
	}

	return pending, uploading, success, int64(failed)
}

// generateRemotePath 生成远程路径
func (t *Transfer) generateRemotePath(file *scanner.FileInfo) string {
	// 获取邮箱前缀
	emailPrefix := t.agentConfig.GetEmailPrefix()
	if emailPrefix == "" {
		emailPrefix = "anonymous"
	}

	// 使用日期作为子目录
	date := file.ModifiedTime.Format("2006-01-02")
	filename := filepath.Base(file.Path)

	// 构建远程路径
	parts := []string{
		t.sftpConfig.RemotePath,
		emailPrefix,
		date,
		filename,
	}

	return filepath.ToSlash(filepath.Join(parts...))
}

// connect 连接到SFTP服务器
func (t *Transfer) connect() error {
	t.logger.Info("Connecting to SFTP server...")
	t.sftp = NewSFTPClient(t.sftpConfig, t.logger)

	if err := t.sftp.Connect(); err != nil {
		return err
	}

	t.logger.Info("Connected to SFTP server successfully")
	return nil
}

// uploadWorker 上传工作协程
func (t *Transfer) uploadWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.processUploads(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// processUploads 处理上传任务
func (t *Transfer) processUploads(ctx context.Context) {
	t.mu.Lock()

	// 获取待上传的文件
	var filesToUpload []string
	for localPath := range t.files {
		if _, exists := t.inFlight[localPath]; !exists {
			filesToUpload = append(filesToUpload, localPath)
		}
	}

	t.mu.Unlock()

	// 上传文件
	for _, localPath := range filesToUpload {
		select {
		case <-ctx.Done():
			return
		default:
			t.uploadFile(localPath)
		}
	}
}

// uploadFile 上传单个文件
func (t *Transfer) uploadFile(localPath string) {
	t.mu.Lock()
	task := &UploadTask{
		LocalPath: localPath,
		RemotePath: t.files[localPath],
		Status:    "uploading",
		StartTime: time.Now(),
	}
	t.inFlight[localPath] = task
	t.mu.Unlock()

	t.logger.Info("Starting upload: %s", localPath)

	// 执行上传
	if err := t.sftp.UploadFile(localPath, task.RemotePath); err != nil {
		task.Status = "failed"
		task.Error = err
		t.logger.Error("Upload failed: %s - %v", localPath, err)
	} else {
		task.Status = "success"
		t.logger.Info("Upload completed: %s", localPath)
	}

	task.EndTime = time.Now()

	// 从待上传列表中移除
	t.mu.Lock()
	delete(t.files, localPath)
	t.mu.Unlock()
}

// RetryFailed 重试失败的上传
func (t *Transfer) RetryFailed() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for path, task := range t.inFlight {
		if task.Status == "failed" {
			t.files[path] = task.RemotePath
			delete(t.inFlight, path)
			t.logger.Info("Queued for retry: %s", path)
		}
	}
}

// ClearCompleted 清除已完成的上传记录
func (t *Transfer) ClearCompleted() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for path, task := range t.inFlight {
		if task.Status == "success" || task.Status == "failed" {
			delete(t.inFlight, path)
		}
	}
}
