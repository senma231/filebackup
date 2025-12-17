package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"doc-scanner-agent/internal/config"
)

// Service 心跳服务
type Service struct {
	config  *config.Config
	logger  Logger
	client  *http.Client
	stats   *SystemStats
	stopCh  chan struct{}
}

// SystemStats 系统统计信息
type SystemStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage int64   `json:"memory_usage"`
	DiskUsage   int64   `json:"disk_usage"`
	ScanCount   int     `json:"scan_count"`
	UploadCount int     `json:"upload_count"`
	ErrorCount  int     `json:"error_count"`
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	AgentID    string      `json:"agent_id"`
	Email      string      `json:"email"`
	Hostname   string      `json:"hostname"`
	IPAddress  string      `json:"ip_address"`
	Status     string      `json:"status"`
	Stats      SystemStats `json:"stats"`
	Timestamp  time.Time   `json:"timestamp"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	ConfigUpdate bool        `json:"config_update"`
	ConfigData   interface{} `json:"config_data"`
}

// Logger 日志接口
type Logger interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

// NewService 创建新的心跳服务
func NewService(cfg *config.Config, logger Logger, uploader interface{}) *Service {
	return &Service{
		config: cfg,
		logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		stats:  &SystemStats{
			CPUUsage:    0.0,
			MemoryUsage: 0,
			DiskUsage:   0,
			ScanCount:   0,
			UploadCount: 0,
			ErrorCount:  0,
		},
		stopCh: make(chan struct{}),
	}
}

// Start 启动心跳服务
func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("Starting heartbeat service...")

	// 启动定时心跳
	go s.heartbeatLoop(ctx)

	s.logger.Info("Heartbeat service started (interval: %ds)", s.config.HeartbeatInterval)
	return nil
}

// Stop 停止心跳服务
func (s *Service) Stop() error {
	s.logger.Info("Stopping heartbeat service...")
	close(s.stopCh)
	return nil
}

// UpdateStats 更新系统统计信息
func (s *Service) UpdateStats(stats SystemStats) {
	s.stats = &stats
}

// GetSystemInfo 获取系统信息
func (s *Service) GetSystemInfo() (string, string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", "", fmt.Errorf("failed to get hostname: %w", err)
	}

	// 获取IP地址（简单实现）
	ipAddress := "127.0.0.1"

	return hostname, ipAddress, nil
}

// heartbeatLoop 心跳循环
func (s *Service) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(s.config.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.sendHeartbeat()
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// sendHeartbeat 发送心跳
func (s *Service) sendHeartbeat() {
	// 获取系统信息
	hostname, ipAddress, err := s.GetSystemInfo()
	if err != nil {
		s.logger.Error("Failed to get system info: %v", err)
		return
	}

	// 构建心跳请求
	req := HeartbeatRequest{
		AgentID:   s.config.AgentID,
		Email:     s.config.Email,
		Hostname:  hostname,
		IPAddress: ipAddress,
		Status:    "online",
		Stats:     *s.stats,
		Timestamp: time.Now(),
	}

	// 发送心跳
	resp, err := s.sendRequest(req)
	if err != nil {
		s.logger.Error("Failed to send heartbeat: %v", err)
		return
	}

	// 处理响应
	if resp.ConfigUpdate {
		s.logger.Info("Configuration update received")
		// TODO: 处理配置更新
	}

	s.logger.Debug("Heartbeat sent successfully")
}

// sendRequest 发送HTTP请求
func (s *Service) sendRequest(req HeartbeatRequest) (*HeartbeatResponse, error) {
	// 序列化请求
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送POST请求
	url := fmt.Sprintf("%s/api/v1/agents/%s/heartbeat", s.config.ServerURL, s.config.AgentID)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.config.Token)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var heartbeatResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &heartbeatResp, nil
}

// GetStats 获取统计信息
func (s *Service) GetStats() *SystemStats {
	return s.stats
}
