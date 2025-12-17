//go:build windows
// +build windows

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"doc-scanner-agent/internal/config"
	"doc-scanner-agent/internal/heartbeat"
	"doc-scanner-agent/internal/logger"
	"doc-scanner-agent/internal/scanner"
	"doc-scanner-agent/internal/uploader"
)

func main() {
	// 初始化日志
	log := logger.New()
	log.Info("Starting Document Scanner Agent...")

	// 检查是否已配置
	isConfigured, err := config.IsConfigured()
	if err != nil {
		log.Error("Failed to check configuration: %v", err)
		os.Exit(1)
	}

	// 如果未配置，尝试从命令行参数或配置文件获取
	if !isConfigured {
		// 1. 尝试从环境变量获取
		agentID := os.Getenv("AGENT_ID")
		serverURL := os.Getenv("SERVER_URL")

		// 2. 如果没有环境变量，尝试从同目录的config.json读取
		if agentID == "" || serverURL == "" {
			cfg, cfgErr := config.LoadWithoutValidation()
			if cfgErr == nil && cfg.AgentID != "" && cfg.ServerURL != "" {
				agentID = cfg.AgentID
				serverURL = cfg.ServerURL
				log.Info("Using configuration from config.json")
			}
		}

		// 3. 如果仍然没有配置，显示错误
		if agentID == "" || serverURL == "" {
			log.Error("Agent not configured!")
			log.Error("Please ensure one of the following:")
			log.Error("1. config.json file exists with agent_id and server_url")
			log.Error("2. Environment variables AGENT_ID and SERVER_URL are set")
			log.Error("3. You are running from a downloaded package")
			os.Exit(1)
		}

		log.Info("Agent not configured, fetching config from server...")
		if err := config.SetupAgent(agentID, serverURL); err != nil {
			log.Error("Failed to setup agent: %v", err)
			os.Exit(1)
		}
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to load config: %v", err)
		os.Exit(1)
	}
	log.Info("Config loaded successfully")

	// 初始化文件扫描器
	scanEngine := scanner.NewEngine(cfg, log)

	// 初始化文件上传器
	sftpConfig := &uploader.Config{
		Host:       cfg.SFTPConfig.Host,
		Port:       cfg.SFTPConfig.Port,
		Username:   cfg.SFTPConfig.Username,
		Password:   cfg.SFTPConfig.Password,
		KeyPath:    cfg.SFTPConfig.KeyPath,
		RemotePath: cfg.SFTPConfig.RemotePath,
		MaxRetries: 3,
		RetryDelay: 5 * time.Second,
	}
	transfer := uploader.NewTransfer(cfg, sftpConfig, log)

	// 初始化心跳服务
	heartbeatSvc := heartbeat.NewService(cfg, log, transfer)

	// 创建上下文，用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动各个服务
	log.Info("Starting services...")

	// 启动文件扫描器
	go func() {
		if err := scanEngine.Start(ctx); err != nil {
			log.Error("Scanner service error: %v", err)
		}
	}()

	// 启动文件上传器
	go func() {
		if err := transfer.Start(ctx); err != nil {
			log.Error("Uploader service error: %v", err)
		}
	}()

	// 启动心跳服务
	go func() {
		if err := heartbeatSvc.Start(ctx); err != nil {
			log.Error("Heartbeat service error: %v", err)
		}
	}()

	// 控制台模式运行
	log.Info("Agent running in console mode. Press Ctrl+C to exit.")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down agent...")

	// 等待服务优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 发送关闭信号到各个服务
	cancel()

	// 等待所有服务关闭
	<-time.After(2 * time.Second)

	// 停止各个服务
	if err := transfer.Stop(); err != nil {
		log.Error("Error stopping uploader: %v", err)
	}
	if err := heartbeatSvc.Stop(); err != nil {
		log.Error("Error stopping heartbeat service: %v", err)
	}

	// 等待关闭完成或超时
	select {
	case <-shutdownCtx.Done():
		log.Warn("Shutdown timeout, forcing exit")
	default:
		log.Info("Agent stopped gracefully")
	}
}
