package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Load 加载配置文件
func Load() (*Config, error) {
	// 优先尝试以下路径（按优先级）：
	// 1. exe 所在目录的 config.json（适用于 Windows 服务）
	// 2. 当前目录的 config.json（适用于控制台模式）
	// 3. 系统默认路径

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		exeDirConfig := filepath.Join(exeDir, "config.json")
		if _, err := os.Stat(exeDirConfig); err == nil {
			// exe 所在目录有 config.json
			return loadConfigFile(exeDirConfig)
		}
	}

	// 尝试当前目录
	currentDirConfig := "config.json"
	if _, err := os.Stat(currentDirConfig); err == nil {
		return loadConfigFile(currentDirConfig)
	}

	// 使用系统默认路径
	systemConfigPath := GetConfigPath()
	return loadConfigFile(systemConfigPath)
}

// loadConfigFile 从指定路径加载配置文件
func loadConfigFile(configPath string) (*Config, error) {
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

		// 同时配置日志目录为exe所在目录/logs
		exePath, err := os.Executable()
		if err == nil {
			exeDir := filepath.Dir(exePath)
			logDir := filepath.Join(exeDir, "logs")
			cfg.LogPath = logDir
		}

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
	// 优先保存到 exe 所在目录，其次是当前目录，最后是系统路径
	var configPath string

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		configPath = filepath.Join(exeDir, "config.json")
		// 尝试在 exe 目录创建
		if err := os.MkdirAll(exeDir, 0755); err == nil {
			data, marshalErr := json.MarshalIndent(cfg, "", "  ")
			if marshalErr == nil {
				if writeErr := os.WriteFile(configPath, data, 0600); writeErr == nil {
					return nil
				}
			}
		}
	}

	// 回退到系统路径
	configPath = GetConfigPath()
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
