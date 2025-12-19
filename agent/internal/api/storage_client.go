package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"doc-scanner-agent/internal/model"
)

// StorageClient 存储配置API客户端
type StorageClient struct {
	serverURL  string
	httpClient *http.Client
}

// NewStorageClient 创建存储配置客户端
func NewStorageClient(serverURL string) *StorageClient {
	return &StorageClient{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetStorageConfig 获取Agent的存储配置
// agentID: Agent的唯一标识
// 返回：存储配置对象，如果没有配置返回nil
func (c *StorageClient) GetStorageConfig(agentID string) (*model.StorageConfig, error) {
	// 构造API URL
	url := fmt.Sprintf("%s/api/v1/agents/%s/storage-config", c.serverURL, agentID)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求服务器失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析API响应
	var apiResp struct {
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    *model.StorageConfig `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查响应状态
	if apiResp.Code == 404 {
		// 未找到配置，返回nil而不是错误
		return nil, nil
	}

	if apiResp.Code != 200 {
		return nil, fmt.Errorf("服务器返回错误: %s (code: %d)", apiResp.Message, apiResp.Code)
	}

	return apiResp.Data, nil
}

// ParseConfigData 解析存储配置的ConfigData字段
// config: 存储配置对象
// target: 目标配置结构的指针（如 &model.SFTPConfig{}）
func (c *StorageClient) ParseConfigData(config *model.StorageConfig, target interface{}) error {
	if config == nil {
		return fmt.Errorf("配置对象为空")
	}

	if err := json.Unmarshal([]byte(config.ConfigData), target); err != nil {
		return fmt.Errorf("解析配置数据失败: %w", err)
	}

	return nil
}
