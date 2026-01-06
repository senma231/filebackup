package uploader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"doc-scanner-agent/internal/config"
	"doc-scanner-agent/internal/database"
	"doc-scanner-agent/internal/scanner"
)

// TransferManager 文件传输管理器（使用Uploader接口）
type TransferManager struct {
	agentConfig *config.Config // Agent配置
	logger      Logger
	uploader    Uploader       // 通用上传器接口
	db          *database.DB   // 本地数据库（用于增量上传）
	files       map[string]string // 本地路径 -> 远程路径
	mu          sync.RWMutex
	inFlight    map[string]*UploadTask // 正在上传的任务
}

// NewTransferManager 创建新的文件传输管理器
func NewTransferManager(agentConfig *config.Config, uploader Uploader, logger Logger, db *database.DB) *TransferManager {
	return &TransferManager{
		agentConfig: agentConfig,
		uploader:    uploader,
		logger:      logger,
		db:          db,
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

	addedCount := 0
	for _, file := range files {
		// 生成远程路径
		remotePath := tm.generateRemotePath(file)

		// 检查文件是否已存在
		if _, exists := tm.files[file.Path]; exists {
			continue
		}

		tm.files[file.Path] = remotePath
		addedCount++
		tm.logger.Debug("已添加待上传文件: %s -> %s", file.Path, remotePath)
	}

	if addedCount > 0 {
		tm.logger.Info("已添加 %d 个新文件到上传队列（共扫描 %d 个文件）", addedCount, len(files))
	}
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
// 新路径结构：Agent邮箱前缀/AgentID/上传日期/文档类型/文件名
// 例如：22@j10vf/agent-1766548785379001281/2025-12-24/docx/报告.docx
func (tm *TransferManager) generateRemotePath(file *scanner.FileInfo) string {
	// 1. Agent邮箱前缀（用于区分不同用户）
	emailPrefix := tm.agentConfig.EmailPrefix
	if emailPrefix == "" {
		emailPrefix = "unknown"
	}

	// 2. AgentID（用于区分不同设备）
	agentID := tm.agentConfig.AgentID
	if agentID == "" {
		agentID = "unknown-agent"
	}

	// 3. 上传日期（当前日期）
	uploadDate := time.Now().Format("2006-01-02")

	// 4. 文档类型（文件扩展名，去掉点号，转小写）
	ext := strings.TrimPrefix(filepath.Ext(file.Path), ".")
	if ext == "" {
		ext = "unknown"
	}
	ext = strings.ToLower(ext)

	// 特殊处理：doc/docx统一为docx，xls/xlsx统一为xlsx，ppt/pptx统一为pptx
	switch ext {
	case "doc":
		ext = "docx"
	case "xls":
		ext = "xlsx"
	case "ppt":
		ext = "pptx"
	}

	// 5. 文件名
	filename := filepath.Base(file.Path)

	// 构建远程路径：邮箱前缀/AgentID/上传日期/文档类型/文件名
	// 这样既保留了用户和设备维度，又保留了上传时间维度，还按类型分类，便于查找和管理
	// 同一天内同名文件会被覆盖，保留最新版本
	remotePath := filepath.Join(emailPrefix, agentID, uploadDate, ext, filename)

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

	// 先通知Server开始上传（创建数据库记录）
	tm.reportUploadStatus(localPath, remotePath, "started")

	tm.logger.Info("开始上传: %s -> %s", localPath, remotePath)

	// 执行上传（带上下文支持）
	if err := tm.uploader.UploadFileWithContext(ctx, localPath, remotePath); err != nil {
		task.Status = "failed"
		task.Error = err
		tm.logger.Error("上传失败: %s - %v", localPath, err)

		// 通知Server上传失败
		tm.reportUploadStatus(localPath, remotePath, "failed")
	} else {
		task.Status = "success"
		tm.logger.Info("上传完成: %s", localPath)

		// 通知Server上传成功
		tm.reportUploadStatus(localPath, remotePath, "completed")

		// 记录到本地数据库（增量上传机制）
		if tm.db != nil {
			fileInfo, err := os.Stat(localPath)
			if err != nil {
				tm.logger.Warn("无法获取文件信息用于记录上传: %s - %v", localPath, err)
			} else {
				if err := tm.db.RecordUpload(localPath, fileInfo.Size(), fileInfo.ModTime(), remotePath); err != nil {
					tm.logger.Warn("记录文件上传到数据库失败: %s - %v", localPath, err)
				} else {
					tm.logger.Debug("已记录文件上传到本地数据库: %s", localPath)
				}
			}
		}
	}

	task.EndTime = time.Now()

	// 从待上传列表和进行中列表中移除
	tm.mu.Lock()
	delete(tm.files, localPath)
	delete(tm.inFlight, localPath)
	tm.mu.Unlock()
}

// reportUploadStatus 向Server报告上传状态
func (tm *TransferManager) reportUploadStatus(localPath, remotePath, status string) {
	// 获取文件信息
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		tm.logger.Warn("无法获取文件信息: %s - %v", localPath, err)
		return
	}

	// 构建请求体
	reqData := map[string]interface{}{
		"file_name":   filepath.Base(localPath),
		"file_size":   fileInfo.Size(),
		"file_type":   strings.TrimPrefix(filepath.Ext(localPath), "."),
		"status":      status,
		"local_path":  localPath,
		"remote_path": remotePath,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		tm.logger.Error("JSON编码失败: %v", err)
		return
	}

	// 构建API URL
	apiURL := fmt.Sprintf("%s/api/v1/agents/%s/upload/progress",
		strings.TrimSuffix(tm.agentConfig.ServerURL, "/"),
		tm.agentConfig.AgentID)

	// 发送HTTP请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		tm.logger.Warn("通知Server上传状态失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tm.logger.Warn("通知Server上传状态失败，状态码: %d", resp.StatusCode)
		return
	}

	tm.logger.Debug("已通知Server上传状态: %s -> %s", localPath, status)
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
