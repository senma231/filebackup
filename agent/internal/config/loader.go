package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Load 加载配置文件
func Load() (*Config, error) {
	// 优先尝试当前目录的 config.json
	currentDirConfig := "config.json"
	systemConfigPath := GetConfigPath()

	var configPath string
	if _, err := os.Stat(currentDirConfig); err == nil {
		// 当前目录有 config.json，优先使用
		configPath = currentDirConfig
	} else {
		// 使用系统默认路径
		configPath = systemConfigPath
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 配置文件不存在，创建默认配置
		cfg := DefaultConfig()
		if err := Save(cfg); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return cfg, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析JSON配置
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 如果AgentID为空，生成一个
	if cfg.AgentID == "" {
		cfg.AgentID = generateAgentID()
		if err := Save(&cfg); err != nil {
			return nil, fmt.Errorf("failed to save config with generated ID: %w", err)
		}
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Save 保存配置到文件
func Save(cfg *Config) error {
	// 确保配置目录存在
	configPath := GetConfigPath()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 序列化配置
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadWithoutValidation 加载配置但不验证（用于配置更新）
func LoadWithoutValidation() (*Config, error) {
	configPath := GetConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// generateAgentID 生成Agent唯一标识
func generateAgentID() string {
	// 简单的UUID生成器
	// 在实际生产环境中，建议使用更强大的UUID库
	return fmt.Sprintf("agent-%d", os.Getpid())
}
