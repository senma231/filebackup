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
	"strings"
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
			".doc", ".docx",
			".xls", ".xlsx",
			".ppt", ".pptx",
			".pdf",
			// ".txt", // 已禁用：避免扫描大量日志文件
			".png", ".jpg", ".jpeg",
		},
		"exclude_patterns": []string{
			"*.log", "*.tmp", "~$*", "._*", ".DS_Store", "Thumbs.db", "desktop.ini",
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
	isInstaller := len(downloadPackage) > 12*1024*1024
	packageType := "EXE安装程序"
	if !isInstaller {
		packageType = "ZIP压缩包"
	}

	instructions := gin.H{
		"step1": "双击运行安装程序",
		"step2": "安装程序会自动安装并启动服务",
		"step3": "Agent将自动连接到服务器并开始扫描",
	}
	if !isInstaller {
		instructions = gin.H{
			"step1": "下载并解压ZIP文件",
			"step2": "双击运行agent.exe",
			"step3": "Agent将自动连接到服务器并开始扫描",
		}
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"agent_id":       agentID,
		"email":          req.Email,
		"email_prefix":   emailPrefix,
		"download_url":   fmt.Sprintf("%s/api/v1/download/enhanced/package/%s", getServerURL(c), agentID),
		"config_preview": config,
		"package_size":   len(downloadPackage),
		"package_type":   packageType,
		"instructions":   instructions,
	}))
}

// generateDownloadPackage 生成下载包
func (h *DownloadHandlerEnhanced) generateDownloadPackage(agentID string, config map[string]interface{}) ([]byte, error) {
	// 优先返回installer.exe（自解压安装程序）
	installerPath := "/app/bin/DocScannerAgent-Setup.exe"
	if _, err := os.Stat(installerPath); err == nil {
		// 返回installer.exe
		installerData, err := os.ReadFile(installerPath)
		if err != nil {
			log.Printf("Warning: failed to read installer.exe: %v", err)
		} else {
			log.Printf("Returning installer.exe, size: %d bytes", len(installerData))
			return installerData, nil
		}
	}

	// 回退到生成ZIP包
	log.Printf("Installer not found, generating ZIP package")
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
	// 将LF转换为CRLF（Windows文本文件兼容性）
	crlfReadme := strings.ReplaceAll(readmeContent, "\n", "\r\n")
	_, err = readmeFile.Write([]byte(crlfReadme))
	if err != nil {
		return nil, err
	}

	// 4. 添加启动脚本（Windows）
	batchContent := h.generateBatchContent(agentID)
	batchFile, err := zipWriter.Create("启动Agent.bat")
	if err != nil {
		return nil, err
	}
	// 将LF转换为CRLF（Windows批处理文件要求）
	crlfBatch := strings.ReplaceAll(batchContent, "\n", "\r\n")
	_, err = batchFile.Write([]byte(crlfBatch))
	if err != nil {
		return nil, err
	}

	// 5. 添加服务管理脚本（Windows）
	serviceScripts := map[string]string{
		"setup-agent.bat":      h.generateSetupAgentContent(),
		"install-service.bat":  h.generateInstallServiceContent(),
		"start-service.bat":    h.generateStartServiceContent(),
		"stop-service.bat":     h.generateStopServiceContent(),
		"restart-service.bat":  h.generateRestartServiceContent(),
		"uninstall-service.bat": h.generateUninstallServiceContent(),
		"run-console.bat":      h.generateRunConsoleContent(),
	}

	for scriptName, scriptContent := range serviceScripts {
		file, err := zipWriter.Create(scriptName)
		if err != nil {
			log.Printf("Warning: failed to add %s to zip: %v", scriptName, err)
			continue
		}
		// 将LF转换为CRLF（Windows批处理文件要求）
		crlfContent := strings.ReplaceAll(scriptContent, "\n", "\r\n")
		_, err = file.Write([]byte(crlfContent))
		if err != nil {
			log.Printf("Warning: failed to write %s to zip: %v", scriptName, err)
			continue
		}
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
echo Starting Document Scanner Agent...
echo.

REM Check if agent.exe exists
if not exist "agent.exe" (
    echo ERROR: agent.exe not found!
    echo Please check file integrity and try again.
    pause
    exit /b 1
)

REM Start the agent in console mode
echo Starting agent...
agent.exe console

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
			".doc", ".docx",
			".xls", ".xlsx",
			".ppt", ".pptx",
			".pdf",
			// ".txt", // 已禁用：避免扫描大量日志文件
			".png", ".jpg", ".jpeg",
		},
		"exclude_patterns": []string{
			"*.log", "*.tmp", "~$*", "._*", ".DS_Store", "Thumbs.db", "desktop.ini",
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

	// 根据文件类型设置响应头
	// 检查是否为installer.exe（通过文件头判断或文件大小）
	// installer.exe约15-20MB，ZIP包约10MB
	filename := fmt.Sprintf("DocScannerAgent_%s.zip", agent.EmailPrefix)
	contentType := "application/zip"

	// 如果文件大于12MB，很可能是installer.exe
	if len(downloadPackage) > 12*1024*1024 {
		filename = fmt.Sprintf("DocScannerAgent_%s.exe", agent.EmailPrefix)
		contentType = "application/vnd.microsoft.portable-executable"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", strconv.Itoa(len(downloadPackage)))

	// 返回文件
	c.Data(http.StatusOK, contentType, downloadPackage)
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
			".doc", ".docx",
			".xls", ".xlsx",
			".ppt", ".pptx",
			".pdf",
			// ".txt", // 已禁用：避免扫描大量日志文件
			".png", ".jpg", ".jpeg",
		},
		"exclude_patterns": []string{
			"*.log", "*.tmp", "~$*", "._*", ".DS_Store", "Thumbs.db", "desktop.ini",
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

// generateInstallServiceContent 生成安装服务脚本内容
func (h *DownloadHandlerEnhanced) generateInstallServiceContent() string {
	return `@echo off
chcp 65001 >nul
setlocal

:: 切换到脚本所在目录
cd /d "%~dp0"

echo ========================================
echo   文档扫描Agent - 服务安装脚本
echo ========================================
echo.

:: 检查agent.exe是否存在
if not exist "agent.exe" (
    echo [错误] agent.exe 不存在！
    echo 请确保此脚本与 agent.exe 在同一目录
    echo.
    pause
    exit /b 1
)

:: 检查管理员权限
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 此脚本需要管理员权限！
    echo.
    echo 请右键点击此文件，选择"以管理员身份运行"
    echo.
    pause
    exit /b 1
)

echo [提示] 正在安装Windows服务...
echo.

:: 安装服务
agent.exe install

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo   安装成功！
    echo ========================================
    echo.
    echo 后续操作：
    echo   1. 启动服务：双击 start-service.bat
    echo   2. 停止服务：双击 stop-service.bat
    echo   3. 卸载服务：双击 uninstall-service.bat
    echo.
    echo 或者使用Windows服务管理器（services.msc）管理
    echo.
) else (
    echo.
    echo [错误] 安装失败，请检查错误信息
    echo.
)

pause
`
}

// generateStartServiceContent 生成启动服务脚本内容
func (h *DownloadHandlerEnhanced) generateStartServiceContent() string {
	return `@echo off
chcp 65001 >nul

:: 切换到脚本所在目录
cd /d "%~dp0"

echo ========================================
echo 启动文档扫描Agent服务
echo ========================================
echo.

:: 检查管理员权限
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 此脚本需要管理员权限！
    echo 请右键点击此文件，选择"以管理员身份运行"
    echo.
    pause
    exit /b 1
)

:: 启动服务
sc start "DocScannerAgent" >nul 2>&1

if %errorlevel% equ 0 (
    echo [成功] 服务启动成功！
    echo.
    echo 查看服务状态：运行 services.msc
) else (
    echo [错误] 服务启动失败
    echo 请检查服务是否已安装
)

echo.
pause
`
}

// generateStopServiceContent 生成停止服务脚本内容
func (h *DownloadHandlerEnhanced) generateStopServiceContent() string {
	return `@echo off
chcp 65001 >nul

:: 切换到脚本所在目录
cd /d "%~dp0"

echo ========================================
echo 停止文档扫描Agent服务
echo ========================================
echo.

:: 检查管理员权限
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 此脚本需要管理员权限！
    echo 请右键点击此文件，选择"以管理员身份运行"
    echo.
    pause
    exit /b 1
)

:: 停止服务
sc stop "DocScannerAgent" >nul 2>&1

if %errorlevel% equ 0 (
    echo [成功] 服务停止成功！
) else (
    echo [错误] 服务停止失败
    echo 请检查服务是否正在运行
)

echo.
pause
`
}

// generateRestartServiceContent 生成重启服务脚本内容
func (h *DownloadHandlerEnhanced) generateRestartServiceContent() string {
	return `@echo off
chcp 65001 >nul

:: 切换到脚本所在目录
cd /d "%~dp0"

echo ========================================
echo 重启文档扫描Agent服务
echo ========================================
echo.

:: 检查管理员权限
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 此脚本需要管理员权限！
    echo 请右键点击此文件，选择"以管理员身份运行"
    echo.
    pause
    exit /b 1
)

:: 停止服务
echo 正在停止服务...
sc stop "DocScannerAgent" >nul 2>&1

:: 等待2秒
timeout /t 2 /nobreak >nul

:: 启动服务
echo 正在启动服务...
sc start "DocScannerAgent" >nul 2>&1

if %errorlevel% equ 0 (
    echo [成功] 服务重启成功！
) else (
    echo [错误] 服务重启失败
)

echo.
pause
`
}

// generateUninstallServiceContent 生成卸载服务脚本内容
func (h *DownloadHandlerEnhanced) generateUninstallServiceContent() string {
	return `@echo off
chcp 65001 >nul
setlocal

:: 切换到脚本所在目录
cd /d "%~dp0"

echo ========================================
echo   文档扫描Agent - 服务卸载脚本
echo ========================================
echo.

:: 检查agent.exe是否存在
if not exist "agent.exe" (
    echo [错误] agent.exe 不存在！
    echo 请确保此脚本与 agent.exe 在同一目录
    echo.
    pause
    exit /b 1
)

:: 检查管理员权限
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 此脚本需要管理员权限！
    echo.
    echo 请右键点击此文件，选择"以管理员身份运行"
    echo.
    pause
    exit /b 1
)

echo [警告] 即将卸载文档扫描Agent服务
echo.
set /p confirm="确认卸载？(Y/N): "
if /i not "%confirm%"=="Y" (
    echo 取消卸载
    pause
    exit /b 0
)

echo.
echo [提示] 正在卸载Windows服务...
echo.

:: 卸载服务
agent.exe uninstall

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo   卸载成功！
    echo ========================================
    echo.
) else (
    echo.
    echo [错误] 卸载失败，请检查错误信息
    echo.
)

pause
`
}

// generateRunConsoleContent 生成控制台运行脚本内容
func (h *DownloadHandlerEnhanced) generateRunConsoleContent() string {
	return `@echo off
chcp 65001 >nul

:: 切换到脚本所在目录
cd /d "%~dp0"

echo ========================================
echo 在控制台运行文档扫描Agent
echo ========================================
echo.
echo 此脚本将在前台控制台运行Agent
echo 关闭此窗口将停止Agent
echo.

:: 检查agent.exe是否存在
if not exist "agent.exe" (
    echo [错误] agent.exe 不存在！
    echo 请检查文件完整性
    echo.
    pause
    exit /b 1
)

:: 在控制台模式运行Agent
echo [启动] 正在启动Agent...
echo.
agent.exe console

echo.
echo [已停止] Agent已停止运行
pause
`
}

// generateSetupAgentContent 生成一键安装脚本内容
func (h *DownloadHandlerEnhanced) generateSetupAgentContent() string {
	return `@echo off
REM 文档扫描Agent 自动安装脚本
REM 此脚本由自解压EXE自动运行

chcp 65001 >nul
cd /d "%~dp0"

echo ========================================
echo   文档扫描Agent - 自动安装
echo ========================================
echo.

REM 检查是否已安装
sc query DocScannerAgent >nul 2>&1
if %errorlevel% equ 0 (
    echo [提示] 服务已存在，正在启动...
    net start DocScannerAgent >nul 2>&1
    if %errorlevel% equ 0 (
        echo ✓ 服务启动成功！
        goto :show_success
    ) else (
        echo ✓ 服务已在运行中！
        goto :show_success
    )
)

REM 安装服务
echo [1/2] 正在安装Windows服务...
agent.exe install >nul 2>&1
if %errorlevel% neq 0 (
    echo ✗ 服务安装失败！
    echo.
    echo 可能的原因：
    echo - 权限不足（请右键"以管理员身份运行"）
    echo - 服务已存在
    echo.
    pause
    exit /b 1
)

echo ✓ 服务安装成功！

REM 启动服务
echo [2/2] 正在启动服务...
agent.exe start >nul 2>&1
if %errorlevel% neq 0 (
    echo ⚠ 服务安装成功，但启动失败
    echo 请手动执行: agent.exe start
    goto :show_finish
)

echo ✓ 服务启动成功！

:show_success
echo.
echo ========================================
echo   安装完成！
echo ========================================
echo.
echo 服务名称: 文档扫描Agent (DocScannerAgent)
echo.
echo 您可以通过以下方式管理服务：
echo - Windows服务管理器 (services.msc)
echo - 命令: agent.exe start/stop/restart/uninstall
echo.
echo 查看状态: agent.exe status
echo 查看日志: %~dp0logs\ 目录
echo.
goto :show_finish

:show_finish
echo 按任意键退出...
pause >nul
exit /b 0
`
}
