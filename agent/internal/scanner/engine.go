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
	"doc-scanner-agent/internal/database"
)

// Scanner 扫描引擎
type Scanner struct {
	config  *config.Config
	logger  Logger
	db      *database.DB
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
func NewEngine(cfg *config.Config, logger Logger, db *database.DB) *Scanner {
	return &Scanner{
		config:   cfg,
		logger:   logger,
		db:       db,
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
	// 用于记录已扫描的用户目录（避免重复扫描）
	scannedUserDirs := make(map[string]bool)

	// 根据配置决定扫描路径
	if s.config.FullDiskScan {
		s.logger.Info("Starting smart scan (Users whitelist, other drives full scan)")
		// 获取所有可用驱动器
		drives := s.getAllDrives()
		if len(drives) == 0 {
			s.logger.Warn("No available drives found")
			return fmt.Errorf("no available drives found")
		}
		s.logger.Info("Found %d available drive(s): %v", len(drives), drives)

		// 首先检查C:\Users是否重定向到其他盘
		usersDrive := s.getUsersDriveLetter()
		s.logger.Info("Users directory is on drive: %s", usersDrive)

		// 用户目录所在盘：使用白名单（只扫描用户目录）
		userDirs := s.getAllowedUserDirectories(usersDrive)
		if len(userDirs) > 0 {
			scanPaths = append(scanPaths, userDirs...)
			for _, dir := range userDirs {
				scannedUserDirs[dir] = true
			}
			s.logger.Info("%s will scan %d allowed director(ies)", usersDrive, len(userDirs))
		}

		// 其他盘：全盘扫描，但排除已扫描的用户目录
		for _, drive := range drives {
			if strings.EqualFold(drive, usersDrive) {
				// 已处理用户目录所在盘
				continue
			}
			// 其他盘：全盘扫描
			scanPaths = append(scanPaths, drive)
			s.logger.Info("%s will be fully scanned", drive)
		}
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

// getUsersDriveLetter 获取Users目录实际所在的盘符
// 支持处理Users目录重定向到其他盘的情况
func (s *Scanner) getUsersDriveLetter() string {
	// 尝试常见的Users路径
	candidatePaths := []struct {
		path string
		desc string
	}{
		{"C:\\Users", "C:\\Users (default)"},
		{"D:\\Users", "D:\\Users"},
		{"E:\\Users", "E:\\Users"},
		{"F:\\Users", "F:\\Users"},
	}

	// 1. 首先检查C:\Users是否存在
	cUsersPath := "C:\\Users"
	info, err := os.Lstat(cUsersPath)
	if err == nil {
		// 如果存在，检查是否为符号链接或重定向
		if info.Mode()&os.ModeSymlink != 0 {
			// 是符号链接，获取实际路径
			actualPath, err := os.Readlink(cUsersPath)
			if err == nil {
				s.logger.Debug("C:\\Users is a symlink to: %s", actualPath)
				if len(actualPath) >= 2 {
					return string(actualPath[0]) + ":\\"
				}
			}
		}

		// 检查是否为目录重定向（通过检查是否真的有子目录）
		entries, err := os.ReadDir(cUsersPath)
		if err == nil && len(entries) > 0 {
			// 有内容，说明是实际的Users目录
			s.logger.Debug("C:\\Users exists with %d entries", len(entries))
			return "C:\\"
		}

		// C:\Users存在但为空，可能是重定向
		s.logger.Debug("C:\\Users exists but is empty, checking other drives...")
	}

	// 2. 检查其他盘的Users目录
	for _, candidate := range candidatePaths {
		if candidate.path == "C:\\Users" {
			continue // 已检查
		}

		info, err := os.Stat(candidate.path)
		if err == nil && info.IsDir() {
			entries, readErr := os.ReadDir(candidate.path)
			if readErr == nil && len(entries) > 0 {
				// 找到实际的Users目录
				s.logger.Info("Found Users directory at: %s (%s)", candidate.path, candidate.desc)
				if len(candidate.path) >= 3 {
					return candidate.path[:3] // 返回 "D:\", "E:\" 等
				}
			}
		}
	}

	// 3. 都没找到，回退到C盘
	s.logger.Warn("Could not locate Users directory, defaulting to C:\\")
	return "C:\\"
}

// getAllowedUserDirectories 获取指定驱动器下所有用户目录的允许目录（白名单）
// 只返回：Documents、Desktop、Downloads、Pictures
func (s *Scanner) getAllowedUserDirectories(drivePath string) []string {
	var allowedDirs []string

	// 允许的目录名称
	allowedDirNames := []string{"Documents", "Desktop", "Downloads", "Pictures"}

	// 构建 Users 路径（如 C:\Users）
	usersPath := filepath.Join(drivePath, "Users")

	// 检查 Users 目录是否存在
	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		s.logger.Debug("Users directory does not exist: %s", usersPath)
		return allowedDirs
	}

	// 读取 Users 目录下的所有用户目录
	entries, err := os.ReadDir(usersPath)
	if err != nil {
		s.logger.Warn("Failed to read Users directory: %v", err)
		return allowedDirs
	}

	// 遍历每个用户目录
	for _, entry := range entries {
		// 跳过非目录和公共目录
		if !entry.IsDir() {
			continue
		}

		userName := entry.Name()
		// 排除公共目录
		if strings.EqualFold(userName, "Public") || strings.EqualFold(userName, "Default") ||
			strings.EqualFold(userName, "All Users") {
			continue
		}

		userPath := filepath.Join(usersPath, userName)

		// 为每个用户目录，收集允许的子目录
		for _, dirName := range allowedDirNames {
			targetDir := filepath.Join(userPath, dirName)
			// 检查目录是否存在
			if _, err := os.Stat(targetDir); err == nil {
				allowedDirs = append(allowedDirs, targetDir)
				s.logger.Debug("Found allowed directory: %s", targetDir)
			}
		}
	}

	return allowedDirs
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

		// 对于目录，检查是否为系统文件夹需要排除
		if info.IsDir() {
			if s.isSystemDirectory(path) {
				s.logger.Debug("Skipping system directory: %s", path)
				return filepath.SkipDir
			}
			return nil
		}

		// 创建文件信息
		file := NewFileInfo(path, info)

		// 检查文件类型
		if !s.isTargetFile(file) {
			return nil
		}

		// 图片文件只扫描Users白名单目录（Documents、Desktop、Downloads、Pictures）
		if s.shouldSkipImageFile(file) {
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

		// 检查文件是否已上传且未修改（增量上传机制）
		if s.db != nil {
			uploaded, err := s.db.IsFileUploaded(file.Path, file.ModifiedTime)
			if err != nil {
				s.logger.Warn("Error checking file upload status: %v", err)
				// 出错时继续扫描该文件
			} else if uploaded {
				// 文件已上传且未修改，跳过
				s.logger.Debug("File already uploaded and not modified, skipping: %s", path)
				// 更新最后检查时间
				s.db.UpdateCheckTime(file.Path)
				return nil
			}
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

// shouldSkipImageFile 检查图片文件是否应该跳过
// 图片文件（.jpg, .png, .jpeg）只扫描Users白名单目录
// 其他目录下的图片文件跳过，避免扫描软件图标等非必要文件
func (s *Scanner) shouldSkipImageFile(file *FileInfo) bool {
	// 检查是否为图片文件
	ext := strings.ToLower(file.Extension)
	imageExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	if !imageExtensions[ext] {
		// 不是图片文件，不需要跳过
		return false
	}

	// 是图片文件，检查是否在Users白名单目录下
	path := filepath.Clean(file.Path)
	lowerPath := strings.ToLower(path)
	parts := strings.Split(lowerPath, string(filepath.Separator))

	// 检查是否在Users目录的白名单子目录中
	// 路径格式: C:\Users\Username\Documents\...\image.jpg
	// 我们需要检查 parts[3] 是否为允许的目录名
	if len(parts) >= 4 && (parts[2] == "users" || s.isUserDriveRoot(parts)) {
		allowedDirs := map[string]bool{
			"documents": true,
			"desktop":   true,
			"downloads": true,
			"pictures":  true,
		}

		// 检查父目录是否为允许的目录
		if len(parts) >= 4 {
			parentDir := parts[3]
			if allowedDirs[parentDir] {
				// 在允许的目录中，不跳过
				return false
			}
		}
	}

	// 图片文件不在Users白名单目录中，跳过
	s.logger.Debug("跳过非Users白名单目录的图片文件: %s", file.Path)
	return true
}

// isUserDriveRoot 检查路径是否为Users驱动器根目录下的路径
// 用于处理Users目录被重定向到其他盘的情况（如 D:\Users\Username）
func (s *Scanner) isUserDriveRoot(parts []string) bool {
	if len(parts) < 3 {
		return false
	}

	// 检查第二部分是否为 Users（处理 D:\Users\Username 的情况）
	if parts[1] == "users" {
		return true
	}

	// 检查是否为Users重定向驱动器
	// 例如：如果C:\Users是符号链接指向D:\Users
	usersDriveLetter := s.getUsersDriveLetter()
	if usersDriveLetter != "" && len(parts) >= 2 {
		// 提取驱动器号（例如 "D:\" -> "d:"）
		drivePart := strings.ToLower(parts[0]) + ":"
		if drivePart == strings.ToLower(usersDriveLetter) {
			// 这个驱动器上的Users目录结构
			if len(parts) >= 3 && parts[1] == "users" {
				return true
			}
		}
	}

	return false
}

// isSystemDirectory 检查是否为系统目录（需要排除）
// 注意：C盘使用白名单方式，此函数主要用于其他驱动器
func (s *Scanner) isSystemDirectory(path string) bool {
	// 将路径转换为小写并标准化（统一使用反斜杠）
	lowerPath := strings.ToLower(filepath.Clean(path))

	// 获取路径段进行精确匹配
	parts := strings.Split(lowerPath, string(filepath.Separator))
	if len(parts) < 2 {
		return false
	}

	// 检查根级系统目录（直接在驱动器根目录下）
	// 例如: D:\Windows, D:\Program Files
	if len(parts) == 2 {
		systemRootDirs := []string{
			"windows",
			"program files",
			"program files (x86)",
			"programdata",
			"$recycle.bin",
			"system volume information",
			"perflogs",
			"recovery",
			"boot",
			"system32",
			"syswow64",
		}

		for _, sysDir := range systemRootDirs {
			if parts[1] == sysDir {
				return true
			}
		}
	}

	// 检查是否为用户目录（D:\Users\XXX）
	// 如果是用户目录，只允许扫描指定的子目录
	if s.isUserDirectoryPath(parts) {
		return s.shouldSkipUserSubDir(parts)
	}

	// 检查 AppData 目录（在用户目录下）
	// 例如: C:\Users\David\AppData
	for i, part := range parts {
		if part == "appdata" || part == "application data" || part == "local settings" {
			return true
		}
		// 同时检查 LocalLow（IE的低权限缓存）
		if i > 0 && parts[i-1] == "appdata" {
			return true
		}
	}

	// 检查隐藏文件夹（以.开头）
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}

	return false
}

// isUserDirectoryPath 检查是否为用户目录路径（C:\Users\XXX）
func (s *Scanner) isUserDirectoryPath(parts []string) bool {
	// Windows 用户目录结构：C:\Users\用户名
	// parts = ["c:", "users", "用户名", ...]
	if len(parts) < 3 {
		return false
	}

	// 检查是否为 C:\Users 或其他盘符的 Users 目录
	if strings.ToLower(parts[1]) != "users" {
		return false
	}

	// 排除公共目录（C:\Users\Public、C:\Users\Default 等）
	publicDirs := []string{"public", "default", "all users"}
	for _, publicDir := range publicDirs {
		if len(parts) >= 3 && strings.ToLower(parts[2]) == publicDir {
			return false
		}
	}

	return true
}

// shouldSkipUserSubDir 检查用户目录下的子目录是否应该跳过
// 只允许扫描：Documents、Desktop、Downloads、Pictures
func (s *Scanner) shouldSkipUserSubDir(parts []string) bool {
	// 如果是用户目录根本身（C:\Users\XXX），不跳过，继续扫描子目录
	if len(parts) == 3 {
		return false
	}

	// 如果是用户目录下的子目录（C:\Users\XXX\YYY），检查是否为允许的目录
	if len(parts) >= 4 {
		subDirName := strings.ToLower(parts[3])

		// 允许的目录（不区分大小写）
		allowedDirs := map[string]bool{
			"documents": true,
			"desktop":   true,
			"downloads": true,
			"pictures":  true,
		}

		// 如果是允许的目录，不跳过，继续扫描其子目录
		if allowedDirs[subDirName] {
			return false
		}

		// 如果是允许目录下的子目录（C:\Users\XXX\Documents\SubFolder），不跳过
		if len(parts) > 4 {
			// 检查父目录是否为允许的目录
			parentDirName := strings.ToLower(parts[3])
			if allowedDirs[parentDirName] {
				return false
			}
		}

		// 其他所有子目录都跳过
		return true
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
