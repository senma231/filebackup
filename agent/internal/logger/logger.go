package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Logger 日志结构
type Logger struct {
	level  string
	logDir string
}

// New 创建新的日志实例
func New() *Logger {
	// 默认日志配置
	return &Logger{
		level:  "info",
		logDir: "",
	}
}

// NewWithConfig 使用配置创建日志实例
func NewWithConfig(logPath, logLevel string) *Logger {
	// 确保日志目录存在
	if err := os.MkdirAll(logPath, 0755); err != nil {
		fmt.Printf("Warning: Failed to create log directory: %v\n", err)
	}

	return &Logger{
		level:  logLevel,
		logDir: logPath,
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
