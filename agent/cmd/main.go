//go:build windows
// +build windows

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"doc-scanner-agent/internal/api"
	"doc-scanner-agent/internal/config"
	"doc-scanner-agent/internal/heartbeat"
	"doc-scanner-agent/internal/logger"
	"doc-scanner-agent/internal/model"
	"doc-scanner-agent/internal/scanner"
	"doc-scanner-agent/internal/service"
	"doc-scanner-agent/internal/uploader"
)

func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "install":
		// 安装Windows服务
		if err := service.InstallService(); err != nil {
			fmt.Printf("❌ 安装服务失败: %v\n", err)
			os.Exit(1)
		}

	case "uninstall":
		// 卸载Windows服务
		if err := service.UninstallService(); err != nil {
			fmt.Printf("❌ 卸载服务失败: %v\n", err)
			os.Exit(1)
		}

	case "start":
		// 启动服务
		if err := service.StartService(); err != nil {
			fmt.Printf("❌ 启动服务失败: %v\n", err)
			os.Exit(1)
		}

	case "stop":
		// 停止服务
		if err := service.StopService(); err != nil {
			fmt.Printf("❌ 停止服务失败: %v\n", err)
			os.Exit(1)
		}

	case "restart":
		// 重启服务
		if err := service.RestartService(); err != nil {
			fmt.Printf("❌ 重启服务失败: %v\n", err)
			os.Exit(1)
		}

	case "status":
		// 显示服务状态
		if err := showStatus(); err != nil {
			fmt.Printf("❌ 获取状态失败: %v\n", err)
			os.Exit(1)
		}

	case "run":
		// 运行服务（由Windows服务管理器调用）
		if err := service.RunService(); err != nil {
			fmt.Printf("❌ 运行服务失败: %v\n", err)
			os.Exit(1)
		}

	case "console":
		// 控制台模式（开发/调试用）
		fmt.Println("⚠️  警告: 控制台模式仅用于开发和调试")
		fmt.Println("⚠️  生产环境请使用服务模式: agent.exe install")
		fmt.Println("")
		if err := runConsoleMode(); err != nil {
			fmt.Printf("❌ 运行失败: %v\n", err)
			os.Exit(1)
		}

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Printf("❌ 未知命令: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("文档扫描Agent - Windows服务管理")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  agent.exe <command>")
	fmt.Println("")
	fmt.Println("服务管理命令:")
	fmt.Println("  install    安装Windows服务（需要管理员权限）")
	fmt.Println("  uninstall  卸载Windows服务（需要管理员权限）")
	fmt.Println("  start      启动服务")
	fmt.Println("  stop       停止服务")
	fmt.Println("  restart    重启服务")
	fmt.Println("  status     查看服务状态和运行信息")
	fmt.Println("")
	fmt.Println("开发/调试命令:")
	fmt.Println("  console    控制台模式运行（不推荐生产环境使用）")
	fmt.Println("")
	fmt.Println("其他命令:")
	fmt.Println("  help       显示此帮助信息")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  1. 安装服务:")
	fmt.Println("     agent.exe install")
	fmt.Println("")
	fmt.Println("  2. 启动服务:")
	fmt.Println("     agent.exe start")
	fmt.Println("")
	fmt.Println("  3. 查看状态:")
	fmt.Println("     agent.exe status")
	fmt.Println("")
	fmt.Println("注意:")
	fmt.Println("  - install/uninstall 需要管理员权限")
	fmt.Println("  - 安装后服务会设置为自动启动")
	fmt.Println("  - 服务以SYSTEM账户运行")
}

// runConsoleMode 控制台模式运行（开发/调试）
func runConsoleMode() error {
	// 初始化日志
	log := logger.New()
	log.Info("正在启动文档扫描Agent（控制台模式）...")

	// 检查是否已配置
	isConfigured, err := config.IsConfigured()
	if err != nil {
		log.Error("检查配置失败: %v", err)
		return err
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
				log.Info("使用config.json中的配置")
			}
		}

		// 3. 如果仍然没有配置，显示错误
		if agentID == "" || serverURL == "" {
			log.Error("Agent未配置!")
			log.Error("请确保以下条件之一:")
			log.Error("1. config.json文件存在且包含agent_id和server_url")
			log.Error("2. 设置了环境变量AGENT_ID和SERVER_URL")
			log.Error("3. 从下载包中运行")
			return fmt.Errorf("Agent未配置")
		}

		log.Info("Agent未配置，正在从服务器获取配置...")
		if err := config.SetupAgent(agentID, serverURL); err != nil {
			log.Error("设置Agent失败: %v", err)
			return err
		}
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Error("加载配置失败: %v", err)
		return err
	}
	log.Info("配置加载成功")

	// 重新创建logger（使用配置中的日志目录）
	log = logger.NewWithConfig(cfg.GetLogPath(), cfg.LogLevel)
	log.Info("日志目录: %s", cfg.GetLogPath())

	// 创建存储配置客户端
	storageClient := api.NewStorageClient(cfg.ServerURL)

	// 从服务器获取存储配置
	log.Info("正在从服务器获取存储配置...")
	storageConfig, err := storageClient.GetStorageConfig(cfg.AgentID)
	if err != nil {
		log.Error("获取存储配置失败: %v", err)
		log.Error("将使用本地配置中的SFTP配置作为后备方案")
		storageConfig = createFallbackStorageConfig(cfg)
	}

	// 如果服务器没有配置，使用本地配置作为后备
	if storageConfig == nil {
		log.Warn("服务器未返回存储配置，使用本地SFTP配置")
		storageConfig = createFallbackStorageConfig(cfg)
	}

	// 如果仍然没有存储配置，给出友好提示
	if storageConfig == nil {
		log.Error("==============================================")
		log.Error("错误：未找到存储配置！")
		log.Error("")
		log.Error("请执行以下操作之一：")
		log.Error("1. 在 Server 管理后台配置存储（推荐）")
		log.Error("   - 访问 %s/admin", cfg.ServerURL)
		log.Error("   - 登录后进入「存储配置」菜单")
		log.Error("   - 添加本地存储、SFTP 或 WebDAV 配置")
		log.Error("")
		log.Error("2. 或在 config.json 中手动添加 sftp_config（不推荐）")
		log.Error("==============================================")
		return fmt.Errorf("未找到存储配置")
	}

	if storageConfig != nil {
		log.Info("存储配置: 类型=%s, 名称=%s", storageConfig.StorageType, storageConfig.Name)
	}

	// 创建Uploader工厂
	uploaderFactory := uploader.NewFactory(log, cfg.ServerURL, cfg.AgentID)

	// 检查存储类型是否已实现
	if !uploaderFactory.IsSupportedType(storageConfig.StorageType) {
		log.Error("存储类型 %s 暂未实现", storageConfig.StorageType)
		log.Error("已支持的存储类型: %v", uploaderFactory.GetSupportedTypes())
		return fmt.Errorf("存储类型不支持")
	}

	// 创建Uploader
	uploadInstance, err := uploaderFactory.CreateUploader(storageConfig)
	if err != nil {
		log.Error("创建上传器失败: %v", err)
		return err
	}
	log.Info("上传器创建成功: %s", uploadInstance.GetType())

	// 初始化文件扫描器
	scanEngine := scanner.NewEngine(cfg, log)

	// 创建文件传输管理器（使用新的TransferManager）
	transferManager := uploader.NewTransferManager(cfg, uploadInstance, log)

	// 初始化心跳服务
	heartbeatSvc := heartbeat.NewService(cfg, log, transferManager)

	// 创建上下文，用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动各个服务
	log.Info("正在启动服务...")

	// 启动文件扫描器
	go func() {
		if err := scanEngine.Start(ctx); err != nil {
			log.Error("文件扫描服务错误: %v", err)
		}
	}()

	// 启动文件上传器
	go func() {
		if err := transferManager.Start(ctx); err != nil {
			log.Error("文件上传服务错误: %v", err)
		}
	}()

	// 启动心跳服务
	go func() {
		if err := heartbeatSvc.Start(ctx); err != nil {
			log.Error("心跳服务错误: %v", err)
		}
	}()

	// 启动文件同步协程（将扫描到的文件添加到上传队列）
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				files := scanEngine.GetFiles()
				if len(files) > 0 {
					transferManager.AddFiles(files)
					log.Debug("已将 %d 个文件添加到上传队列", len(files))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 控制台模式运行
	log.Info("Agent正在以控制台模式运行。按Ctrl+C退出。")
	log.Info("存储类型: %s", transferManager.GetStorageType())

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("正在关闭Agent...")

	// 等待服务优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 发送关闭信号到各个服务
	cancel()

	// 等待所有服务关闭
	<-time.After(2 * time.Second)

	// 停止各个服务
	if err := transferManager.Stop(); err != nil {
		log.Error("停止上传器时出错: %v", err)
	}
	if err := heartbeatSvc.Stop(); err != nil {
		log.Error("停止心跳服务时出错: %v", err)
	}

	// 等待关闭完成或超时
	select {
	case <-shutdownCtx.Done():
		log.Warn("关闭超时，强制退出")
	default:
		log.Info("Agent已优雅停止")
	}

	return nil
}

// createFallbackStorageConfig 创建后备存储配置（使用本地配置中的SFTP）
func createFallbackStorageConfig(cfg *config.Config) *model.StorageConfig {
	// 如果本地配置中没有SFTP配置，返回nil
	if cfg.SFTPConfig.Host == "" {
		return nil
	}

	// 构造SFTP配置JSON
	sftpConfigJSON := `{
		"host": "` + cfg.SFTPConfig.Host + `",
		"port": ` + toString(cfg.SFTPConfig.Port) + `,
		"username": "` + cfg.SFTPConfig.Username + `",
		"password": "` + cfg.SFTPConfig.Password + `",
		"key_path": "` + cfg.SFTPConfig.KeyPath + `",
		"remote_path": "` + cfg.SFTPConfig.RemotePath + `",
		"timeout": 30
	}`

	return &model.StorageConfig{
		Name:        "本地SFTP配置（后备）",
		StorageType: model.StorageTypeSFTP,
		ConfigData:  sftpConfigJSON,
		IsActive:    true,
		Description: "从本地config.json加载的SFTP配置",
		Priority:    0,
	}
}

// toString 将int转为字符串
func toString(n int) string {
	if n == 0 {
		return "22"
	}
	// 使用fmt.Sprintf更可靠
	return fmt.Sprintf("%d", n)
}

// showStatus 显示服务状态和运行信息
func showStatus() error {
	fmt.Println("========================================")
	fmt.Println("   文档扫描Agent - 运行状态")
	fmt.Println("========================================")
	fmt.Println()

	// 1. 检查Windows服务状态
	fmt.Println("【服务状态】")
	cmd := exec.Command("sc", "query", "DocScannerAgent")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("  ⚠ 无法查询服务状态: %v\n", err)
		fmt.Println("  提示: 服务可能未安装")
	} else {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "STATE") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					state := parts[2]
					switch state {
					case "RUNNING":
						fmt.Printf("  ✓ 运行中\n")
					case "STOPPED":
						fmt.Printf("  ✗ 已停止\n")
					default:
						fmt.Printf("  ⚠ 状态: %s\n", state)
					}
				}
				break
			}
		}
	}

	fmt.Println()

	// 2. 读取配置信息
	fmt.Println("【配置信息】")
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("  ⚠ 无法加载配置: %v\n", err)
	} else {
		fmt.Printf("  Agent ID: %s\n", cfg.AgentID)
		fmt.Printf("  Server URL: %s\n", cfg.ServerURL)
		fmt.Printf("  日志目录: %s\n", cfg.GetLogPath())
		fmt.Printf("  心跳间隔: %d 秒\n", cfg.HeartbeatInterval)
		fmt.Printf("  扫描模式: %s\n", map[bool]string{true: "全盘扫描", false: "指定路径"}[cfg.FullDiskScan])
	}

	fmt.Println()

	// 3. 显示日志信息
	fmt.Println("【日志信息】")
	logDir := ""
	if cfg != nil {
		logDir = cfg.GetLogPath()
	}

	if logDir == "" {
		fmt.Println("  ⚠ 日志目录未配置")
	} else {
		// 检查日志目录是否存在
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			fmt.Printf("  ⚠ 日志目录不存在: %s\n", logDir)
		} else {
			// 列出最近的日志文件
			files, err := os.ReadDir(logDir)
			if err != nil {
				fmt.Printf("  ⚠ 无法读取日志目录: %v\n", err)
			} else {
				// 筛选日志文件并按修改时间排序
				type logFileInfo struct {
					name    string
					modTime time.Time
					size    int64
				}
				var logFiles []logFileInfo

				for _, file := range files {
					if !file.IsDir() && strings.HasPrefix(file.Name(), "agent-") && strings.HasSuffix(file.Name(), ".log") {
						info, _ := file.Info()
						logFiles = append(logFiles, logFileInfo{
							name:    file.Name(),
							modTime: info.ModTime(),
							size:    info.Size(),
						})
					}
				}

				// 按修改时间降序排序
				sort.Slice(logFiles, func(i, j int) bool {
					return logFiles[i].modTime.After(logFiles[j].modTime)
				})

				if len(logFiles) == 0 {
					fmt.Println("  ℹ 暂无日志文件")
				} else {
					fmt.Printf("  共有 %d 个日志文件\n", len(logFiles))
					fmt.Println()

					// 显示最近3个日志文件
					count := 3
					if len(logFiles) < count {
						count = len(logFiles)
					}

					for i := 0; i < count; i++ {
						file := logFiles[i]
						fmt.Printf("  • %s\n", file.name)
						fmt.Printf("    大小: %s, 修改时间: %s\n",
							formatFileSize(file.size),
							file.modTime.Format("2006-01-02 15:04:05"))

						// 读取最后几行日志
						logPath := filepath.Join(logDir, file.name)
						if lastLines, err := readLastNLines(logPath, 3); err == nil {
							fmt.Println("    最新日志:")
							for _, line := range lastLines {
								fmt.Printf("      %s\n", strings.TrimSpace(line))
							}
						}
						fmt.Println()
					}
				}
			}
		}
	}

	fmt.Println("========================================")
	fmt.Println("提示:")
	fmt.Println("  - 使用 'agent.exe start' 启动服务")
	fmt.Println("  - 使用 'agent.exe stop' 停止服务")
	fmt.Println("  - 使用 'agent.exe restart' 重启服务")
	fmt.Println("  - 日志文件位置: " + logDir)
	fmt.Println("========================================")

	return nil
}

// readLastNLines 读取文件的最后N行
func readLastNLines(filePath string, n int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取所有行
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 返回最后n行
	if len(lines) <= n {
		return lines, nil
	}

	return lines[len(lines)-n:], nil
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
