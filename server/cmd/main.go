package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"doc-scanner-server/internal/api"
	"doc-scanner-server/internal/repository"
)

func main() {
	fmt.Println("Starting Document Scanner Server...")

	// 初始化数据库
	db, err := repository.NewDB("data/database.db")
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 运行数据库迁移
	if err := repository.Migrate(db); err != nil {
		fmt.Printf("Failed to migrate database: %v\n", err)
		os.Exit(1)
	}

	// 创建上下文，用于优雅关闭
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动HTTP服务器
	server := api.NewServer(db)

	go func() {
		if err := server.Start(":8889"); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Println("Server started on port 8889")
	fmt.Println("API Documentation: http://localhost:8889/api/docs")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down server...")

	// 等待服务优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 发送关闭信号
	cancel()

	// 等待服务关闭或超时
	select {
	case <-shutdownCtx.Done():
		fmt.Println("Shutdown timeout, forcing exit")
	default:
		fmt.Println("Server stopped gracefully")
	}

	// 关闭数据库连接
	if err := db.Close(); err != nil {
		fmt.Printf("Error closing database: %v\n", err)
	}
}
