package uploader

import (
	"encoding/json"
	"fmt"

	"doc-scanner-agent/internal/model"
)

// Factory Uploader工厂
type Factory struct {
	logger Logger
}

// NewFactory 创建Uploader工厂
func NewFactory(logger Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateUploader 根据存储配置创建Uploader
func (f *Factory) CreateUploader(storageConfig *model.StorageConfig) (Uploader, error) {
	if storageConfig == nil {
		return nil, fmt.Errorf("存储配置为空")
	}

	f.logger.Info("创建 %s 类型的上传器: %s", storageConfig.StorageType, storageConfig.Name)

	switch storageConfig.StorageType {
	case model.StorageTypeSFTP:
		return f.createSFTPUploader(storageConfig)

	case model.StorageTypeWebDAV:
		return f.createWebDAVUploader(storageConfig)

	case model.StorageTypeLocal:
		return f.createLocalUploader(storageConfig)

	case model.StorageTypeAliyunOSS:
		return nil, fmt.Errorf("阿里云OSS上传器暂未实现，敬请期待")

	case model.StorageTypeTencentCOS:
		return nil, fmt.Errorf("腾讯云COS上传器暂未实现，敬请期待")

	case model.StorageTypeAWSS3:
		return nil, fmt.Errorf("AWS S3上传器暂未实现，敬请期待")

	case model.StorageTypeSMB:
		return nil, fmt.Errorf("SMB/CIFS上传器暂未实现，敬请期待")

	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", storageConfig.StorageType)
	}
}

// createSFTPUploader 创建SFTP上传器
func (f *Factory) createSFTPUploader(storageConfig *model.StorageConfig) (Uploader, error) {
	var config model.SFTPConfig
	if err := json.Unmarshal([]byte(storageConfig.ConfigData), &config); err != nil {
		return nil, fmt.Errorf("解析SFTP配置失败: %w", err)
	}

	// 设置默认值
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}

	return NewSFTPUploader(&config, f.logger), nil
}

// createWebDAVUploader 创建WebDAV上传器
func (f *Factory) createWebDAVUploader(storageConfig *model.StorageConfig) (Uploader, error) {
	var config model.WebDAVConfig
	if err := json.Unmarshal([]byte(storageConfig.ConfigData), &config); err != nil {
		return nil, fmt.Errorf("解析WebDAV配置失败: %w", err)
	}

	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = 30
	}

	return NewWebDAVUploader(&config, f.logger), nil
}

// createLocalUploader 创建本地存储上传器
func (f *Factory) createLocalUploader(storageConfig *model.StorageConfig) (Uploader, error) {
	var config model.LocalConfig
	if err := json.Unmarshal([]byte(storageConfig.ConfigData), &config); err != nil {
		return nil, fmt.Errorf("解析本地存储配置失败: %w", err)
	}

	return NewLocalUploader(&config, f.logger), nil
}

// GetSupportedTypes 获取支持的存储类型列表
func (f *Factory) GetSupportedTypes() []model.StorageType {
	return []model.StorageType{
		model.StorageTypeSFTP,
		model.StorageTypeWebDAV,
		model.StorageTypeLocal,
		// 以下类型暂未实现
		// model.StorageTypeAliyunOSS,
		// model.StorageTypeTencentCOS,
		// model.StorageTypeAWSS3,
		// model.StorageTypeSMB,
	}
}

// IsSupportedType 检查存储类型是否已实现
func (f *Factory) IsSupportedType(storageType model.StorageType) bool {
	supportedTypes := f.GetSupportedTypes()
	for _, t := range supportedTypes {
		if t == storageType {
			return true
		}
	}
	return false
}
