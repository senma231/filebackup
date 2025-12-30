package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Logger 日志结构
type Logger struct {
	level       string
	logDir      string
	lastCleanup time.Time // 上次清理时间
}

// New 创建新的日志实例
func New() *Logger {
	// 默认日志配置
	return &Logger{
		level:       "info",
		logDir:      "",
		lastCleanup: time.Time{},
	}
}

// NewWithConfig 使用配置创建日志实例
func NewWithConfig(logPath, logLevel string) *Logger {
	// 确保日志目录存在
	if err := os.MkdirAll(logPath, 0755); err != nil {
		fmt.Printf("Warning: Failed to create log directory: %v\n", err)
	}

	logger := &Logger{
		level:       logLevel,
		logDir:      logPath,
		lastCleanup: time.Time{},
	}

	// 初始化时清理一次旧日志
	logger.cleanupOldLogs()

	return logger
}

// cleanupOldLogs 清理超过7天的旧日志文件
func (l *Logger) cleanupOldLogs() {
	if l.logDir == "" {
		return
	}

	// 检查是否需要清理（距离上次清理超过24小时才执行）
	if !l.lastCleanup.IsZero() && time.Since(l.lastCleanup) < 24*time.Hour {
		return
	}

	// 更新清理时间
	l.lastCleanup = time.Now()

	// 读取日志目录
	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return
	}

	// 7天前的时间
	cutoffTime := time.Now().AddDate(0, 0, -7)

	// 删除超过7天的日志文件
	deletedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 检查是否为日志文件
		matched, _ := filepath.Match("agent-*.log", entry.Name())
		if !matched {
			continue
		}

		// 获取文件信息
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 检查文件修改时间
		if info.ModTime().Before(cutoffTime) {
			logFile := filepath.Join(l.logDir, entry.Name())
			if err := os.Remove(logFile); err == nil {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		fmt.Printf("[INFO] 已清理 %d 个过期日志文件（超过7天）\n", deletedCount)
	}
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log("WARN", format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level == "debug" {
		l.log("DEBUG", format, args...)
	}
}

// Fatal 致命错误日志
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log("FATAL", format, args...)
	os.Exit(1)
}

// log 内部日志记录方法
func (l *Logger) log(level, format string, args ...interface{}) {
	// 格式化消息
	message := fmt.Sprintf(format, args...)

	// 创建日志行
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)

	// 输出到控制台
	fmt.Print(logLine)

	// 写入文件
	if l.logDir != "" {
		l.writeToFile(logLine)
	}
}

// writeToFile 写入日志文件
func (l *Logger) writeToFile(logLine string) error {
	// 每天触发一次清理检查
	l.cleanupOldLogs()

	// 获取当前日期作为日志文件名
	date := time.Now().Format("2006-01-02")
	logFile := filepath.Join(l.logDir, fmt.Sprintf("agent-%s.log", date))

	// 以追加模式打开文件
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入日志
	_, err = file.WriteString(logLine)
	return err
}
