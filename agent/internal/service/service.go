package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"doc-scanner-agent/internal/api"
	"doc-scanner-agent/internal/config"
	"doc-scanner-agent/internal/heartbeat"
	"doc-scanner-agent/internal/logger"
	"doc-scanner-agent/internal/model"
	"doc-scanner-agent/internal/scanner"
	"doc-scanner-agent/internal/uploader"

	"github.com/kardianos/service"
)

// AgentService Agent服务包装器
type AgentService struct {
	cfg             *config.Config
	log             *logger.Logger
	scanEngine      *scanner.Scanner
	transferManager *uploader.TransferManager
	heartbeatSvc    *heartbeat.Service
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewAgentService 创建Agent服务
func NewAgentService() (*AgentService, error) {
	// 初始化日志
	log := logger.New()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return &AgentService{
		cfg: cfg,
		log: log,
	}, nil
}

// Start 启动服务
func (s *AgentService) Start(svc service.Service) error {
	s.log.Info("正在启动文档扫描Agent服务...")

	// 检查是否已配置
	isConfigured, err := config.IsConfigured()
	if err != nil {
		return fmt.Errorf("检查配置失败: %w", err)
	}

	if !isConfigured {
		return fmt.Errorf("Agent未配置，请先运行安装命令或配置config.json")
	}

	// 创建存储配置客户端
	storageClient := api.NewStorageClient(s.cfg.ServerURL)

	// 从服务器获取存储配置
	s.log.Info("正在从服务器获取存储配置...")
	storageConfig, err := storageClient.GetStorageConfig(s.cfg.AgentID)
	if err != nil {
		s.log.Error("获取存储配置失败: %v", err)
		storageConfig = s.createFallbackStorageConfig()
	}

	if storageConfig == nil {
		storageConfig = s.createFallbackStorageConfig()
	}

	if storageConfig == nil {
		return fmt.Errorf("未找到存储配置，请在Server管理后台配置存储")
	}

	s.log.Info("存储配置: 类型=%s, 名称=%s", storageConfig.StorageType, storageConfig.Name)

	// 创建Uploader工厂
	uploaderFactory := uploader.NewFactory(s.log, s.cfg.ServerURL, s.cfg.AgentID)

	// 检查存储类型是否已实现
	if !uploaderFactory.IsSupportedType(storageConfig.StorageType) {
		return fmt.Errorf("存储类型 %s 暂未实现", storageConfig.StorageType)
	}

	// 创建Uploader
	uploadInstance, err := uploaderFactory.CreateUploader(storageConfig)
	if err != nil {
		return fmt.Errorf("创建上传器失败: %w", err)
	}
	s.log.Info("上传器创建成功: %s", uploadInstance.GetType())

	// 初始化文件扫描器
	s.scanEngine = scanner.NewEngine(s.cfg, s.log)

	// 创建文件传输管理器
	s.transferManager = uploader.NewTransferManager(s.cfg, uploadInstance, s.log)

	// 初始化心跳服务
	s.heartbeatSvc = heartbeat.NewService(s.cfg, s.log, s.transferManager)

	// 创建上下文
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// 启动各个服务
	go func() {
		if err := s.scanEngine.Start(s.ctx); err != nil {
			s.log.Error("文件扫描服务错误: %v", err)
		}
	}()

	go func() {
		if err := s.transferManager.Start(s.ctx); err != nil {
			s.log.Error("文件上传服务错误: %v", err)
		}
	}()

	go func() {
		if err := s.heartbeatSvc.Start(s.ctx); err != nil {
			s.log.Error("心跳服务错误: %v", err)
		}
	}()

	// 启动文件同步协程
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				files := s.scanEngine.GetFiles()
				if len(files) > 0 {
					s.transferManager.AddFiles(files)
					s.log.Debug("已将 %d 个文件添加到上传队列", len(files))
				}
			case <-s.ctx.Done():
				return
			}
		}
	}()

	s.log.Info("Agent服务启动成功")
	s.log.Info("存储类型: %s", s.transferManager.GetStorageType())

	return nil
}

// Stop 停止服务
func (s *AgentService) Stop(svc service.Service) error {
	s.log.Info("正在停止Agent服务...")

	if s.cancel != nil {
		s.cancel()
	}

	// 等待服务关闭
	time.Sleep(2 * time.Second)

	// 停止各个服务
	if s.transferManager != nil {
		if err := s.transferManager.Stop(); err != nil {
			s.log.Error("停止上传器时出错: %v", err)
		}
	}

	if s.heartbeatSvc != nil {
		if err := s.heartbeatSvc.Stop(); err != nil {
			s.log.Error("停止心跳服务时出错: %v", err)
		}
	}

	s.log.Info("Agent服务已停止")
	return nil
}

// createFallbackStorageConfig 创建后备存储配置
func (s *AgentService) createFallbackStorageConfig() *model.StorageConfig {
	if s.cfg.SFTPConfig.Host == "" {
		return nil
	}

	sftpConfigJSON := fmt.Sprintf(`{
		"host": "%s",
		"port": %d,
		"username": "%s",
		"password": "%s",
		"key_path": "%s",
		"remote_path": "%s",
		"timeout": 30
	}`, s.cfg.SFTPConfig.Host, s.cfg.SFTPConfig.Port,
		s.cfg.SFTPConfig.Username, s.cfg.SFTPConfig.Password,
		s.cfg.SFTPConfig.KeyPath, s.cfg.SFTPConfig.RemotePath)

	return &model.StorageConfig{
		Name:        "本地SFTP配置（后备）",
		StorageType: model.StorageTypeSFTP,
		ConfigData:  sftpConfigJSON,
		IsActive:    true,
		Description: "从本地config.json加载的SFTP配置",
		Priority:    0,
	}
}

// InstallService 安装Windows服务
func InstallService() error {
	// 获取可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	// 服务配置
	svcConfig := &service.Config{
		Name:        "DocScannerAgent",
		DisplayName: "文档扫描Agent",
		Description: "自动扫描并上传文档到服务器的后台服务",
		Arguments:   []string{"run"}, // 服务模式运行参数
		Executable:  exePath,
	}

	// 创建服务实例
	agentSvc, err := NewAgentService()
	if err != nil {
		return fmt.Errorf("创建服务实例失败: %w", err)
	}

	svc, err := service.New(agentSvc, svcConfig)
	if err != nil {
		return fmt.Errorf("创建服务失败: %w", err)
	}

	// 安装服务
	if err := svc.Install(); err != nil {
		return fmt.Errorf("安装服务失败: %w", err)
	}

	fmt.Println("✅ 服务安装成功！")
	fmt.Println("服务名称: DocScannerAgent")
	fmt.Println("")
	fmt.Println("后续操作:")
	fmt.Println("  启动服务: agent.exe start")
	fmt.Println("  停止服务: agent.exe stop")
	fmt.Println("  卸载服务: agent.exe uninstall")
	fmt.Println("")
	fmt.Println("或通过Windows服务管理器（services.msc）管理")

	return nil
}

// UninstallService 卸载Windows服务
func UninstallService() error {
	svcConfig := &service.Config{
		Name: "DocScannerAgent",
	}

	agentSvc, err := NewAgentService()
	if err != nil {
		return fmt.Errorf("创建服务实例失败: %w", err)
	}

	svc, err := service.New(agentSvc, svcConfig)
	if err != nil {
		return fmt.Errorf("创建服务失败: %w", err)
	}

	// 卸载服务
	if err := svc.Uninstall(); err != nil {
		return fmt.Errorf("卸载服务失败: %w", err)
	}

	fmt.Println("✅ 服务卸载成功！")
	return nil
}

// StartService 启动服务
func StartService() error {
	svcConfig := &service.Config{
		Name: "DocScannerAgent",
	}

	agentSvc, err := NewAgentService()
	if err != nil {
		return fmt.Errorf("创建服务实例失败: %w", err)
	}

	svc, err := service.New(agentSvc, svcConfig)
	if err != nil {
		return fmt.Errorf("创建服务失败: %w", err)
	}

	if err := svc.Start(); err != nil {
		return fmt.Errorf("启动服务失败: %w", err)
	}

	fmt.Println("✅ 服务启动成功！")
	return nil
}

// StopService 停止服务
func StopService() error {
	svcConfig := &service.Config{
		Name: "DocScannerAgent",
	}

	agentSvc, err := NewAgentService()
	if err != nil {
		return fmt.Errorf("创建服务实例失败: %w", err)
	}

	svc, err := service.New(agentSvc, svcConfig)
	if err != nil {
		return fmt.Errorf("创建服务失败: %w", err)
	}

	if err := svc.Stop(); err != nil {
		return fmt.Errorf("停止服务失败: %w", err)
	}

	fmt.Println("✅ 服务停止成功！")
	return nil
}

// RestartService 重启服务
func RestartService() error {
	if err := StopService(); err != nil {
		return err
	}

	time.Sleep(2 * time.Second)

	if err := StartService(); err != nil {
		return err
	}

	return nil
}

// RunService 运行服务
func RunService() error {
	agentSvc, err := NewAgentService()
	if err != nil {
		return fmt.Errorf("创建服务实例失败: %w", err)
	}

	svcConfig := &service.Config{
		Name:        "DocScannerAgent",
		DisplayName: "文档扫描Agent",
		Description: "自动扫描并上传文档到服务器的后台服务",
	}

	svc, err := service.New(agentSvc, svcConfig)
	if err != nil {
		return fmt.Errorf("创建服务失败: %w", err)
	}

	// 运行服务
	if err := svc.Run(); err != nil {
		return fmt.Errorf("运行服务失败: %w", err)
	}

	return nil
}
