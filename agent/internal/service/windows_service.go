//go:build windows
// +build windows

package service

import (
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

// WindowsService Windows服务实现
type WindowsService struct {
	scanner    interface{}
	uploader   interface{}
	heartbeat  interface{}
	logger     Logger
}

// Logger 日志接口
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

// NewWindowsService 创建新的Windows服务
func NewWindowsService(scanner, uploader, heartbeat interface{}, logger Logger) *WindowsService {
	return &WindowsService{
		scanner:   scanner,
		uploader:  uploader,
		heartbeat: heartbeat,
		logger:    logger,
	}
}

// Execute 实现Windows服务接口
func (s *WindowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	// 报告服务状态
	changes <- svc.Status{State: svc.StartPending, Accepts: cmdsAccepted}

	// 启动服务
	s.logger.Info("Starting Windows service...")

	// 报告服务运行中
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// 处理服务命令
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s.logger.Info("Shutting down Windows service...")
				changes <- svc.Status{State: svc.StopPending, Accepts: cmdsAccepted}
				return false, 0
			default:
				s.logger.Warn("Unexpected service control request: %d", c.Cmd)
			}
		case <-time.After(5 * time.Second):
			// 定期报告服务状态
			changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
		}
	}
}

// Run 运行Windows服务
func Run(windowsService *WindowsService) error {
	// 注册事件日志
	elog, err := eventlog.Open("DocScannerAgent")
	if err != nil {
		return err
	}
	defer elog.Close()

	// 运行服务
	return svc.Run("DocScannerAgent", windowsService)
}
