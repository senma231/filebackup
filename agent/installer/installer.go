//go:build windows
// +build windows

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 编译时设置，默认值可以被环境变量覆盖
var serverURL = getServerURL()

func getServerURL() string {
	// 1. 从环境变量读取
	if url := os.Getenv("SERVER_URL"); url != "" {
		return url
	}

	// 2. 从同目录的 .server_url 文件读取
	exePath, err := os.Executable()
	if err == nil {
		serverURLFile := filepath.Join(filepath.Dir(exePath), ".server_url")
		if data, err := os.ReadFile(serverURLFile); err == nil {
			url := strings.TrimSpace(string(data))
			if url != "" {
				return url
			}
		}
	}

	// 3. 默认值（本地测试用）
	return "http://localhost:8889"
}

func main() {
	fmt.Println("========================================")
	fmt.Println("  文档扫描Agent - 安装程序")
	fmt.Println("========================================")
	fmt.Println()

	// 1. 从文件名或命令行获取 agent_id
	agentID := getAgentID()
	if agentID == "" {
		fmt.Println("❌ 错误：无法获取Agent ID")
		fmt.Println()
		fmt.Println("请确保：")
		fmt.Println("  1. 文件名格式为: installer_{agent_id}.exe")
		fmt.Println("  2. 或使用命令行: installer.exe {agent_id}")
		fmt.Println()
		pause()
		return
	}

	fmt.Printf("Agent ID: %s\n", agentID)
	fmt.Printf("服务器: %s\n", serverURL)
	fmt.Println()

	// 2. 创建安装目录
	targetDir := filepath.Join("C:\\", "Users", "Public", "Documents", "DocScannerAgent")
	fmt.Printf("[1/4] 创建安装目录: %s\n", targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Printf("❌ 创建目录失败: %v\n", err)
		pause()
		return
	}
	fmt.Println("✓ 目录创建成功")
	fmt.Println()

	// 3. 从服务器下载 agent.exe
	fmt.Println("[2/4] 从服务器下载 Agent 程序...")
	agentEXEPath := filepath.Join(targetDir, "agent.exe")
	downloadURL := fmt.Sprintf("%s/api/v1/download/agent-binary", serverURL)

	if err := downloadFile(downloadURL, agentEXEPath); err != nil {
		fmt.Printf("❌ 下载失败: %v\n", err)
		fmt.Println()
		fmt.Println("可能的原因：")
		fmt.Println("  - 无法连接到服务器")
		fmt.Println("  - 网络连接问题")
		fmt.Printf("  - 服务器地址: %s\n", serverURL)
		fmt.Println()
		pause()
		return
	}
	fmt.Println("✓ 下载完成")
	fmt.Println()

	// 4. 下载配置文件
	fmt.Println("[3/4] 下载配置文件...")
	configPath := filepath.Join(targetDir, "config.json")
	configURL := fmt.Sprintf("%s/api/v1/download/agent/%s", serverURL, agentID)

	if err := downloadFile(configURL, configPath); err != nil {
		fmt.Printf("⚠ 配置下载失败: %v\n", err)
		fmt.Println("  Agent将使用默认配置启动")
	} else {
		fmt.Println("✓ 配置下载成功")
	}
	fmt.Println()

	// 5. 安装并启动服务
	fmt.Println("[4/4] 安装 Windows 服务...")
	os.Chdir(targetDir)

	// 安装服务
	cmd := exec.Command(agentEXEPath, "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ 安装服务失败: %v\n", err)
		pause()
		return
	}
	fmt.Println("✓ 服务安装成功")

	// 启动服务
	fmt.Println()
	fmt.Println("启动服务...")
	cmd = exec.Command(agentEXEPath, "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ 启动服务失败: %v\n", err)
		pause()
		return
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  ✅ 安装完成！")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Printf("安装目录: %s\n", targetDir)
	fmt.Printf("日志目录: %s\\logs\n", targetDir)
	fmt.Println()
	fmt.Println("服务已启动，Agent 将自动开始扫描和上传文档")
	fmt.Println()
	pause()
}

// getAgentID 从文件名或命令行参数获取 agent_id
func getAgentID() string {
	// 优先从命令行参数获取
	if len(os.Args) > 1 {
		return os.Args[1]
	}

	// 从文件名提取（格式：installer_{agent_id}.exe）
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}

	fileName := filepath.Base(exePath)
	// 去掉扩展名
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// 提取 agent_id（installer_{agent_id} -> agent_id）
	if strings.HasPrefix(name, "installer_") {
		return strings.TrimPrefix(name, "installer_")
	}

	// 如果文件名是 DocScannerAgent-Setup，尝试从同目录的 .agent_id 文件读取
	idFile := filepath.Join(filepath.Dir(exePath), ".agent_id")
	if data, err := os.ReadFile(idFile); err == nil {
		return strings.TrimSpace(string(data))
	}

	return ""
}

// downloadFile 从 URL 下载文件
func downloadFile(url, filepath string) error {
	// 创建 HTTP 客户端（设置超时）
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// 发送请求
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误: %s", resp.Status)
	}

	// 创建文件
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	// 写入文件并显示进度
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	fmt.Printf("  下载: %.2f MB\n", float64(written)/1024/1024)

	return nil
}

func pause() {
	fmt.Print("\n按回车键退出...")
	fmt.Scanln()
}
