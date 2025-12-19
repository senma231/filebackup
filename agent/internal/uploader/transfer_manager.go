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

// TransferManager 文件传输管理器（使用Uploader接口）
type TransferManager struct {
	agentConfig *config.Config // Agent配置
	logger      Logger
	uploader    Uploader       // 通用上传器接口
	files       map[string]string // 本地路径 -> 远程路径
	mu          sync.RWMutex
	inFlight    map[string]*UploadTask // 正在上传的任务
}

// NewTransferManager 创建新的文件传输管理器
func NewTransferManager(agentConfig *config.Config, uploader Uploader, logger Logger) *TransferManager {
	return &TransferManager{
		agentConfig: agentConfig,
		uploader:    uploader,
		logger:      logger,
		files:       make(map[string]string),
		inFlight:    make(map[string]*UploadTask),
	}
}

// Start 启动文件传输管理器
func (tm *TransferManager) Start(ctx context.Context) error {
	tm.logger.Info("正在启动文件上传管理器...")
	tm.logger.Info("存储类型: %s", tm.uploader.GetType())

	// 连接到存储服务，支持重试
	var connectErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := tm.uploader.Connect(); err != nil {
			connectErr = err
			tm.logger.Warn("连接存储服务失败（尝试 %d/3）: %v", attempt, err)
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*2) * time.Second)
			}
		} else {
			connectErr = nil
			break
		}
	}

	if connectErr != nil {
		return fmt.Errorf("连接存储服务失败（已重试3次）: %w", connectErr)
	}

	// 启动上传协程
	go tm.uploadWorker(ctx)

	tm.logger.Info("文件上传管理器已启动")
	return nil
}

// Stop 停止文件传输管理器
func (tm *TransferManager) Stop() error {
	tm.logger.Info("正在停止文件上传管理器...")

	if tm.uploader != nil {
		if err := tm.uploader.Disconnect(); err != nil {
			tm.logger.Warn("断开存储连接时出错: %v", err)
		}
	}

	tm.logger.Info("文件上传管理器已停止")
	return nil
}

// AddFiles 添加要上传的文件
func (tm *TransferManager) AddFiles(files []*scanner.FileInfo) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, file := range files {
		// 生成远程路径
		remotePath := tm.generateRemotePath(file)

		// 检查文件是否已存在
		if _, exists := tm.files[file.Path]; exists {
			continue
		}

		tm.files[file.Path] = remotePath
		tm.logger.Debug("已添加待上传文件: %s -> %s", file.Path, remotePath)
	}

	tm.logger.Info("已添加 %d 个文件到上传队列", len(files))
}

// GetUploadStatus 获取上传状态
func (tm *TransferManager) GetUploadStatus() map[string]*UploadTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	status := make(map[string]*UploadTask)
	for path, task := range tm.inFlight {
		status[path] = task
	}

	return status
}

// GetUploadStats 获取上传统计
func (tm *TransferManager) GetUploadStats() (pending, uploading, success int, failed int64) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var totalSize int64

	for _, task := range tm.inFlight {
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

	return pending, uploading, success, failed
}

// generateRemotePath 生成远程路径
func (tm *TransferManager) generateRemotePath(file *scanner.FileInfo) string {
	// 获取邮箱前缀
	emailPrefix := tm.agentConfig.GetEmailPrefix()
	if emailPrefix == "" {
		emailPrefix = "anonymous"
	}

	// 使用日期作为子目录
	date := file.ModifiedTime.Format("2006-01-02")
	filename := filepath.Base(file.Path)

	// 构建远程路径：邮箱前缀/日期/文件名
	remotePath := filepath.Join(emailPrefix, date, filename)

	return filepath.ToSlash(remotePath)
}

// uploadWorker 上传工作协程
func (tm *TransferManager) uploadWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.processUploads(ctx)
		case <-ctx.Done():
			tm.logger.Info("上传工作协程收到停止信号")
			return
		}
	}
}

// processUploads 处理上传任务
func (tm *TransferManager) processUploads(ctx context.Context) {
	tm.mu.Lock()

	// 获取待上传的文件
	var filesToUpload []string
	for localPath := range tm.files {
		// 只上传不在进行中的文件
		if _, exists := tm.inFlight[localPath]; !exists {
			filesToUpload = append(filesToUpload, localPath)
			// 限制每次批量上传的文件数量
			if len(filesToUpload) >= 10 {
				break
			}
		}
	}

	tm.mu.Unlock()

	if len(filesToUpload) > 0 {
		tm.logger.Info("准备上传 %d 个文件", len(filesToUpload))
	}

	// 上传文件
	for _, localPath := range filesToUpload {
		select {
		case <-ctx.Done():
			return
		default:
			tm.uploadFile(ctx, localPath)
		}
	}
}

// uploadFile 上传单个文件
func (tm *TransferManager) uploadFile(ctx context.Context, localPath string) {
	tm.mu.Lock()
	remotePath, exists := tm.files[localPath]
	if !exists {
		tm.mu.Unlock()
		return
	}

	task := &UploadTask{
		LocalPath:  localPath,
		RemotePath: remotePath,
		Status:     "uploading",
		StartTime:  time.Now(),
	}
	tm.inFlight[localPath] = task
	tm.mu.Unlock()

	tm.logger.Info("开始上传: %s -> %s", localPath, remotePath)

	// 执行上传（带上下文支持）
	if err := tm.uploader.UploadFileWithContext(ctx, localPath, remotePath); err != nil {
		task.Status = "failed"
		task.Error = err
		tm.logger.Error("上传失败: %s - %v", localPath, err)
	} else {
		task.Status = "success"
		tm.logger.Info("上传完成: %s", localPath)
	}

	task.EndTime = time.Now()

	// 从待上传列表中移除
	tm.mu.Lock()
	delete(tm.files, localPath)
	tm.mu.Unlock()
}

// RetryFailed 重试失败的上传
func (tm *TransferManager) RetryFailed() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	retryCount := 0
	for path, task := range tm.inFlight {
		if task.Status == "failed" {
			tm.files[path] = task.RemotePath
			delete(tm.inFlight, path)
			retryCount++
		}
	}

	if retryCount > 0 {
		tm.logger.Info("已将 %d 个失败文件加入重试队列", retryCount)
	}
}

// ClearCompleted 清除已完成的上传记录
func (tm *TransferManager) ClearCompleted() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	clearedCount := 0
	for path, task := range tm.inFlight {
		if task.Status == "success" || task.Status == "failed" {
			delete(tm.inFlight, path)
			clearedCount++
		}
	}

	if clearedCount > 0 {
		tm.logger.Info("已清除 %d 条上传记录", clearedCount)
	}
}

// GetStorageType 获取当前存储类型
func (tm *TransferManager) GetStorageType() string {
	if tm.uploader != nil {
		return tm.uploader.GetType()
	}
	return "unknown"
}

// TestConnection 测试存储连接
func (tm *TransferManager) TestConnection() error {
	if tm.uploader == nil {
		return fmt.Errorf("上传器未初始化")
	}
	return tm.uploader.TestConnection()
}
