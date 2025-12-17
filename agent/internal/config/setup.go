package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SetupAgent 首次启动设置Agent配置
func SetupAgent(agentID, serverURL string) error {
	// 1. 从服务器获取配置
	configData, err := fetchConfigFromServer(agentID, serverURL)
	if err != nil {
		return fmt.Errorf("failed to fetch config from server: %w", err)
	}

	// 2. 解析配置
	var serverConfig map[string]interface{}
	if err := json.Unmarshal(configData, &serverConfig); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// 3. 更新本地配置
	cfg, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 更新配置字段
	if email, ok := serverConfig["email"].(string); ok {
		cfg.Email = email
		cfg.EmailPrefix = GetEmailPrefixFromEmail(email)
	}

	if serverURL != "" {
		cfg.ServerURL = serverURL
	}

	if agentID != "" {
		cfg.AgentID = agentID
	}

	// 更新扫描路径
	if scanPaths, ok := serverConfig["scan_paths"].([]interface{}); ok {
		cfg.ScanPaths = make([]string, len(scanPaths))
		for i, path := range scanPaths {
			cfg.ScanPaths[i] = path.(string)
		}
	}

	// 更新文件类型
	if fileTypes, ok := serverConfig["file_types"].([]interface{}); ok {
		cfg.FileTypes = make([]string, len(fileTypes))
		for i, ft := range fileTypes {
			cfg.FileTypes[i] = ft.(string)
		}
	}

	// 更新排除模式
	if excludePatterns, ok := serverConfig["exclude_patterns"].([]interface{}); ok {
		cfg.ExcludePatterns = make([]string, len(excludePatterns))
		for i, pattern := range excludePatterns {
			cfg.ExcludePatterns[i] = pattern.(string)
		}
	}

	// 更新SFTP配置
	if sftpConfig, ok := serverConfig["sftp_config"].(map[string]interface{}); ok {
		if host, ok := sftpConfig["host"].(string); ok {
			cfg.SFTPConfig.Host = host
		}
		if port, ok := sftpConfig["port"].(float64); ok {
			cfg.SFTPConfig.Port = int(port)
		}
		if username, ok := sftpConfig["username"].(string); ok {
			cfg.SFTPConfig.Username = username
		}
		if remotePath, ok := sftpConfig["remote_path"].(string); ok {
			cfg.SFTPConfig.RemotePath = remotePath
		}
	}

	// 4. 保存配置
	if err := Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✅ Agent配置更新成功！")
	return nil
}

// fetchConfigFromServer 从服务器获取配置
func fetchConfigFromServer(agentID, serverURL string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/download/agent/%s", strings.TrimSuffix(serverURL, "/"), agentID)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned error: %s", string(body))
	}

	return io.ReadAll(resp.Body)
}

// IsConfigured 检查Agent是否已配置
func IsConfigured() (bool, error) {
	cfg, err := Load()
	if err != nil {
		return false, err
	}

	// 检查必要字段
	if cfg.AgentID == "" {
		return false, nil
	}

	if cfg.ServerURL == "" {
		return false, nil
	}

	if cfg.Email == "" {
		return false, nil
	}

	return true, nil
}
