package handlers

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// DownloadHandlerEnhanced 增强版下载处理器
type DownloadHandlerEnhanced struct {
	repo *repository.AgentRepository
	agentBinaryPath string // 预编译的Agent程序路径
}

// NewDownloadHandlerEnhanced 创建新的增强版下载处理器
func NewDownloadHandlerEnhanced(db *sql.DB, agentBinaryPath string) *DownloadHandlerEnhanced {
	return &DownloadHandlerEnhanced{
		repo:             repository.NewAgentRepository(db),
		agentBinaryPath:  agentBinaryPath,
	}
}

// RequestDownloadEnhanced 请求下载Agent（增强版）
func (h *DownloadHandlerEnhanced) RequestDownloadEnhanced(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required"`
		Hostname    string `json:"hostname"`    // 不再必需，Agent启动时会自动获取
		IPAddress   string `json:"ip_address"` // 不再必需，Agent启动时会自动获取
		OSVersion   string `json:"os_version"` // 不再必需，Agent启动时会自动获取
		AgentVersion string `json:"agent_version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, err.Error()))
		return
	}

	// 验证邮箱格式
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, model.Error(http.StatusBadRequest, "Invalid email format"))
		return
	}

	// 提取邮箱前缀
	emailPrefix := extractEmailPrefix(req.Email)

	// 生成唯一的Agent ID
	agentID := generateAgentID()

	// 创建Agent记录（状态为pending，等待Agent启动后激活）
	now := time.Now()

	// 如果是自动检测的值，设为空字符串，Agent启动后会更新
	hostname := req.Hostname
	if hostname == "" || hostname == "auto-detect" {
		hostname = ""
	}
	ipAddress := req.IPAddress
	if ipAddress == "" || ipAddress == "auto-detect" {
		ipAddress = ""
	}
	osVersion := req.OSVersion
	if osVersion == "" || osVersion == "auto-detect" {
		osVersion = ""
	}

	agent := &model.Agent{
		AgentID:      agentID,
		Email:        req.Email,
		EmailPrefix:  emailPrefix,
		Hostname:     hostname,
		IPAddress:    ipAddress,
		OSVersion:    osVersion,
		AgentVersion: req.AgentVersion,
		Status:       "pending",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.repo.Create(agent); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to create agent record"))
		return
	}

	// 生成配置信息 - 启用全盘扫描
	// 注意：存储配置由 Server 端统一管理，Agent 启动时会自动从 Server 获取
	config := gin.H{
		"server_url":       getServerURL(c),
		"agent_id":         agentID,
		"email":            req.Email,
		"email_prefix":     emailPrefix,
		"heartbeat_interval": 30,
		"full_disk_scan":   true, // 启用全盘扫描，自动检测并扫描所有可用驱动器
		"scan_paths": []string{
			// 仅当 full_disk_scan=false 时使用
			"C:\\Users\\%USERNAME%\\Documents",
		},
		"file_types": []string{
			".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".pdf", ".txt",
		},
		"exclude_patterns": []string{
			"*.log", "*.tmp", "~$*",
		},
		"max_file_size": 104857600,
		"incremental_scan": true,
		"compress_enabled": true,
		// 存储配置说明：Agent 启动后会自动从 Server 获取存储配置
		// 请在 Server 管理后台配置存储类型（SFTP、WebDAV、本地存储等）
		"_storage_note": "Storage configuration will be automatically fetched from server on agent startup",
	}

	// 生成下载包
	downloadPackage, err := h.generateDownloadPackage(agentID, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to generate download package"))
		return
	}

	// 返回下载信息
	c.JSON(http.StatusOK, model.Success(gin.H{
		"agent_id":          agentID,
		"email":             req.Email,
		"email_prefix":      emailPrefix,
		"download_url":      fmt.Sprintf("%s/api/v1/download/enhanced/package/%s", getServerURL(c), agentID),
		"config_preview":    config,
		"package_size":      len(downloadPackage),
		"instructions": gin.H{
			"step1": "下载并解压ZIP文件",
			"step2": "双击运行agent.exe",
			"step3": "Agent将自动连接到服务器并开始扫描",
		},
	}))
}

// generateDownloadPackage 生成下载包
func (h *DownloadHandlerEnhanced) generateDownloadPackage(agentID string, config map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// 1. 添加配置文件
	configData, _ := json.MarshalIndent(config, "", "  ")
	configFile, err := zipWriter.Create("config.json")
	if err != nil {
		return nil, err
	}
	_, err = configFile.Write(configData)
	if err != nil {
		return nil, err
	}

	// 2. 添加Agent程序（如果存在）
	if h.agentBinaryPath != "" {
		// 仅在二进制文件实际存在时才添加
		if _, err := os.Stat(h.agentBinaryPath); err == nil {
			agentFile, err := zipWriter.Create("agent.exe")
			if err != nil {
				return nil, err
			}

			// 读取预编译的Agent文件
			agentData, err := os.ReadFile(h.agentBinaryPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read agent binary: %w", err)
			}

			_, err = agentFile.Write(agentData)
			if err != nil {
				return nil, err
			}
		} else {
			// 记录日志并继续，不要因缺少二进制而失败
			log.Printf("Agent binary not found at %s, skipping adding agent.exe: %v", h.agentBinaryPath, err)
		}
	}

	// 3. 添加说明文件
	readmeContent := h.generateReadmeContent(agentID, config)
	readmeFile, err := zipWriter.Create("README.txt")
	if err != nil {
		return nil, err
	}
	_, err = readmeFile.Write([]byte(readmeContent))
	if err != nil {
		return nil, err
	}

	// 4. 添加启动脚本（Windows）
	batchContent := h.generateBatchContent(agentID)
	batchFile, err := zipWriter.Create("启动Agent.bat")
	if err != nil {
		return nil, err
	}
	_, err = batchFile.Write([]byte(batchContent))
	if err != nil {
		return nil, err
	}

	// 关闭ZIP写入器
	err = zipWriter.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// generateReadmeContent 生成说明文件内容
func (h *DownloadHandlerEnhanced) generateReadmeContent(agentID string, config map[string]interface{}) string {
	// 安全地获取配置值，避免类型断言panic
	email := ""
	if v, ok := config["email"].(string); ok {
		email = v
	}

	emailPrefix := ""
	if v, ok := config["email_prefix"].(string); ok {
		emailPrefix = v
	}

	serverURL := ""
	if v, ok := config["server_url"].(string); ok {
		serverURL = v
	}

	// 检查是否启用了全盘扫描
	fullDiskScan := true
	if v, ok := config["full_disk_scan"].(bool); ok {
		fullDiskScan = v
	}

	scanMode := "全盘扫描模式（自动检测并扫描所有可用驱动器 C:, D:, E: 等）"
	if !fullDiskScan {
		scanMode = "指定路径扫描模式（仅扫描配置文件中指定的目录）"
	}

	return fmt.Sprintf(`文件备份系统 - 备份客户端
=====================================

客户端 ID: %s
邮箱地址: %s
备份标识: %s
服务器: %s

【安装步骤】
1. 解压压缩包
2. 双击 agent.exe 或运行 启动.bat
3. 客户端启动后将自动连接服务器
4. 存储配置将自动从服务器获取

【扫描模式】
%s

【存储配置说明】
- 存储配置由服务器端统一管理
- 管理员可在后台配置存储类型（本地、SFTP、WebDAV等）
- Agent 启动时会自动获取并应用配置
- 无需手动配置存储信息

【备份文件类型】
- Word 文档 (.doc, .docx)
- Excel 表格 (.xls, .xlsx)
- PowerPoint 演示文稿 (.ppt, .pptx)
- PDF 文档 (.pdf)
- 文本文件 (.txt)

【注意事项】
- 确保防火墙允许网络访问
- 首次运行可能需要管理员权限
- 不要中断备份过程
- 存储配置请联系管理员在后台设置
- 原文件不会被删除或移动，仅上传副本

生成时间: %s
=====================================
`, agentID, email, emailPrefix, serverURL, scanMode, time.Now().Format("2006-01-02 15:04:05"))
}

// generateBatchContent 生成批处理文件内容
func (h *DownloadHandlerEnhanced) generateBatchContent(agentID string) string {
	return `@echo off
chcp 65001 >nul
echo 正在启动文件备份系统...
echo.

REM 检查agent.exe是否存在
if not exist "agent.exe" (
    echo 错误：未找到 agent.exe 文件！
    echo 请确保文件完整性后重试。
    pause
    exit /b 1
)

REM 启动备份客户端
echo 启动备份客户端...
start "" "agent.exe"

echo.
echo 备份客户端已启动！
echo 程序将在后台运行。
echo.
pause
`
}

// DownloadPackage 下载完整包
func (h *DownloadHandlerEnhanced) DownloadPackage(c *gin.Context) {
	agentID := c.Param("agent_id")

	// 验证Agent是否存在
	agent, err := h.repo.GetByID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agent"))
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Agent not found"))
		return
	}

	// 生成配置
	// 注意：存储配置由 Server 端统一管理，Agent 启动时会自动从 Server 获取
	config := gin.H{
		"server_url":        getServerURL(c),
		"agent_id":          agentID,
		"email":             agent.Email,
		"email_prefix":      agent.EmailPrefix,
		"heartbeat_interval": 30,
		"full_disk_scan":    true, // 启用全盘扫描
		"scan_paths": []string{
			// 仅当 full_disk_scan=false 时使用
			getDefaultDocumentsPath(agent.Hostname),
		},
		"file_types": []string{
			".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".pdf", ".txt",
		},
		"exclude_patterns": []string{
			"*.log", "*.tmp", "~$*",
		},
		"max_file_size": 104857600,
		"incremental_scan": true,
		"compress_enabled": true,
		// 存储配置说明：Agent 启动后会自动从 Server 获取存储配置
		// 请在 Server 管理后台配置存储类型（SFTP、WebDAV、本地存储等）
		"_storage_note": "Storage configuration will be automatically fetched from server on agent startup",
	}

	// 生成下载包
	downloadPackage, err := h.generateDownloadPackage(agentID, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to generate package"))
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("DocScannerAgent_%s.zip", agent.EmailPrefix)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", strconv.Itoa(len(downloadPackage)))

	// 返回ZIP文件
	c.Data(http.StatusOK, "application/zip", downloadPackage)
}

// GetAgentConfig 获取Agent配置（用于Agent启动时获取）
func (h *DownloadHandlerEnhanced) GetAgentConfig(c *gin.Context) {
	agentID := c.Param("agent_id")

	// 查找Agent
	agent, err := h.repo.GetByID(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(http.StatusInternalServerError, "Failed to get agent"))
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Agent not found"))
		return
	}

	// 生成配置信息
	// 注意：存储配置由 Server 端统一管理，Agent 启动时会自动从 Server 获取
	config := gin.H{
		"server_url":        getServerURL(c),
		"agent_id":          agentID,
		"email":             agent.Email,
		"email_prefix":      agent.EmailPrefix,
		"heartbeat_interval": 30,
		"full_disk_scan":    true, // 启用全盘扫描
		"scan_paths": []string{
			// 仅当 full_disk_scan=false 时使用
			getDefaultDocumentsPath(agent.Hostname),
		},
		"file_types": []string{
			".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".pdf", ".txt",
		},
		"exclude_patterns": []string{
			"*.log", "*.tmp", "~$*",
		},
		"max_file_size": 104857600,
		"incremental_scan": true,
		"compress_enabled": true,
		// 存储配置说明：Agent 启动后会自动从 Server 获取存储配置
		// 请在 Server 管理后台配置存储类型（SFTP、WebDAV、本地存储等）
		"_storage_note": "Storage configuration will be automatically fetched from server on agent startup",
	}

	c.JSON(http.StatusOK, model.Success(config))
}
