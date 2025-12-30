//go:build windows
// +build windows

package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed agent.exe
var embeddedAgentEXE embed.FS

func init() {
	// 调试：检查嵌入的文件是否存在及其大小
	if file, err := embeddedAgentEXE.Open("agent.exe"); err == nil {
		if info, err := file.Stat(); err == nil {
			fmt.Printf("[DEBUG] Embedded agent.exe size: %d bytes\n", info.Size())
		}
		file.Close()
	} else {
		fmt.Printf("[DEBUG] Failed to open embedded agent.exe: %v\n", err)
	}
}

func main() {
	fmt.Println("========================================")
	fmt.Println("  文档扫描Agent - 安装程序")
	fmt.Println("========================================")
	fmt.Println()

	// 解压目标目录：C:\Users\Public\Documents\DocScannerAgent
	targetDir := filepath.Join("C:\\", "Users", "Public", "Documents", "DocScannerAgent")

	// 检查是否已安装
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Println("检测到已安装，正在重新安装...")
		fmt.Println()
	}

	// 创建目标目录
	fmt.Printf("[1/3] 创建安装目录: %s\n", targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		pause()
		return
	}

	// 解压agent.exe
	fmt.Println("[2/3] 解压程序文件...")
	agentEXEPath := filepath.Join(targetDir, "agent.exe")
	if err := copyEmbeddedFile("agent.exe", agentEXEPath); err != nil {
		fmt.Printf("解压失败: %v\n", err)
		pause()
		return
	}

	// 安装服务
	fmt.Println("[3/3] 安装Windows服务...")
	// 切换到目标目录
	os.Chdir(targetDir)

	// 执行agent.exe install
	cmd := exec.Command(agentEXEPath, "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("\n安装失败: %v\n", err)
		pause()
		return
	}

	// 启动服务
	fmt.Println("\n启动服务...")
	cmd = exec.Command(agentEXEPath, "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("\n启动失败: %v\n", err)
		pause()
		return
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  安装完成！")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Printf("安装目录: %s\n", targetDir)
	fmt.Printf("日志目录: %s\\logs\n", targetDir)
	fmt.Println()
	fmt.Println("管理命令:")
	fmt.Println("  查看状态: agent.exe status")
	fmt.Println("  启动服务: agent.exe start")
	fmt.Println("  停止服务: agent.exe stop")
	fmt.Println()
	pause()
}

func copyEmbeddedFile(embeddedPath, destPath string) error {
	srcFile, err := embeddedAgentEXE.Open(embeddedPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func pause() {
	fmt.Print("\n按回车键退出...")
	fmt.Scanln()
}
