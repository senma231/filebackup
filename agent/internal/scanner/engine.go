package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"doc-scanner-agent/internal/config"
)

// Scanner 扫描引擎
type Scanner struct {
	config  *config.Config
	logger  Logger
	files   []*FileInfo
	mu      sync.RWMutex
	scanCh  chan string
	resultCh chan *FileInfo
}

// Logger 日志接口
type Logger interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

// NewEngine 创建新的扫描引擎
func NewEngine(cfg *config.Config, logger Logger) *Scanner {
	return &Scanner{
		config:   cfg,
		logger:   logger,
		files:    make([]*FileInfo, 0),
		scanCh:   make(chan string, 100),
		resultCh: make(chan *FileInfo, 1000),
	}
}

// Start 启动扫描引擎
func (s *Scanner) Start(ctx context.Context) error {
	s.logger.Info("Starting file scanner...")

	// 启动扫描协程
	go s.scanWorker(ctx)

	// 启动结果处理协程
	go s.resultWorker(ctx)

	// 立即执行一次全量扫描
	s.logger.Info("Performing initial full scan...")
	if err := s.FullScan(); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	// 启动定时扫描
	go s.scheduleScan(ctx)

	s.logger.Info("File scanner started successfully")
	return nil
}

// scanWorker 扫描工作协程
func (s *Scanner) scanWorker(ctx context.Context) {
	defer close(s.scanCh)
	for {
		select {
		case path := <-s.scanCh:
			s.scanPath(ctx, path)
		case <-ctx.Done():
			s.logger.Debug("Scanner worker shutting down")
			return
		}
	}
}

// resultWorker 结果处理协程
func (s *Scanner) resultWorker(ctx context.Context) {
	defer close(s.resultCh)
	for {
		select {
		case file := <-s.resultCh:
			if file != nil {
				s.addFile(file)
			}
		case <-ctx.Done():
			s.logger.Debug("Result worker shutting down")
			return
		}
	}
}

// FullScan 执行全量扫描
func (s *Scanner) FullScan() error {
	var scanPaths []string

	// 根据配置决定扫描路径
	if s.config.FullDiskScan {
		s.logger.Info("Starting full disk scan (all available drives)")
		// 获取所有可用驱动器
		drives := s.getAllDrives()
		if len(drives) == 0 {
			s.logger.Warn("No available drives found")
			return fmt.Errorf("no available drives found")
		}
		scanPaths = drives
		s.logger.Info("Found %d available drive(s): %v", len(drives), drives)
	} else {
		s.logger.Info("Starting scan of %d configured paths", len(s.config.ScanPaths))
		scanPaths = s.config.ScanPaths
	}

	// 清空之前的文件列表
	s.mu.Lock()
	s.files = s.files[:0]
	s.mu.Unlock()

	// 扫描所有路径，过滤不存在的路径
	for _, path := range scanPaths {
		// 检查路径是否存在
		if _, err := os.Stat(path); os.IsNotExist(err) {
			s.logger.Debug("Path does not exist, skipping: %s", path)
			continue
		}

		select {
		case s.scanCh <- path:
		default:
			s.logger.Warn("Scan channel full, skipping path: %s", path)
		}
	}

	return nil
}

// getAllDrives 获取所有可用的驱动器（Windows）
func (s *Scanner) getAllDrives() []string {
	var drives []string

	// 在 Windows 上检查 A-Z 所有可能的驱动器号
	for drive := 'A'; drive <= 'Z'; drive++ {
		drivePath := string(drive) + ":\\"

		// 检查驱动器是否存在且可访问
		if _, err := os.Stat(drivePath); err == nil {
			drives = append(drives, drivePath)
			s.logger.Debug("Found available drive: %s", drivePath)
		}
	}

	return drives
}

// scanPath 扫描单个路径
func (s *Scanner) scanPath(ctx context.Context, rootPath string) {
	s.logger.Debug("Scanning path: %s", rootPath)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}

		if err != nil {
			s.logger.Warn("Error accessing path %s: %v", path, err)
			return nil
		}

		// 跳过目录，继续遍历
		if info.IsDir() {
			return nil
		}

		// 创建文件信息
		file := NewFileInfo(path, info)

		// 检查文件类型
		if !s.isTargetFile(file) {
			return nil
		}

		// 检查是否应该排除
		if file.ShouldExclude(s.config.ExcludePatterns) {
			return nil
		}

		// 检查文件大小
		if file.Size > s.config.MaxFileSize {
			s.logger.Debug("File too large, skipping: %s (%d bytes)", path, file.Size)
			return nil
		}

		// 发送文件信息
		select {
		case s.resultCh <- file:
		default:
			s.logger.Warn("Result channel full, skipping file: %s", path)
		}

		return nil
	})

	if err != nil {
		s.logger.Error("Error scanning path %s: %v", rootPath, err)
	}
}

// isTargetFile 检查是否为目标文件
func (s *Scanner) isTargetFile(file *FileInfo) bool {
	for _, ext := range s.config.FileTypes {
		if strings.EqualFold(file.Extension, ext) {
			return true
		}
	}
	return false
}

// addFile 添加文件到列表
func (s *Scanner) addFile(file *FileInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = append(s.files, file)
}

// GetFiles 获取所有扫描到的文件
func (s *Scanner) GetFiles() []*FileInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// 返回副本
	result := make([]*FileInfo, len(s.files))
	copy(result, s.files)
	return result
}

// GetFileCount 获取文件数量
func (s *Scanner) GetFileCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.files)
}

// GetTotalSize 获取总大小
func (s *Scanner) GetTotalSize() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total int64
	for _, file := range s.files {
		total += file.Size
	}
	return total
}

// GetFilesByType 按类型统计文件
func (s *Scanner) GetFilesByType() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := make(map[string]int)
	for _, file := range s.files {
		fileType := file.GetType()
		stats[fileType]++
	}
	return stats
}

// scheduleScan 定时扫描
func (s *Scanner) scheduleScan(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.logger.Info("Performing scheduled scan...")
			if err := s.FullScan(); err != nil {
				s.logger.Error("Scheduled scan failed: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
