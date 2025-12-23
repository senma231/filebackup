package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"doc-scanner-server/internal/model"
)

// StorageConfigRepository 存储配置仓储
type StorageConfigRepository struct {
	db *sql.DB
}

// NewStorageConfigRepository 创建存储配置仓储
func NewStorageConfigRepository(db *sql.DB) *StorageConfigRepository {
	return &StorageConfigRepository{db: db}
}

// InitTable 初始化存储配置表
func (r *StorageConfigRepository) InitTable() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS storage_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		storage_type TEXT NOT NULL,
		config_data TEXT NOT NULL,
		is_active INTEGER DEFAULT 1,
		description TEXT,
		target_agent_id TEXT,
		priority INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_storage_type ON storage_configs(storage_type);
	CREATE INDEX IF NOT EXISTS idx_target_agent ON storage_configs(target_agent_id);
	CREATE INDEX IF NOT EXISTS idx_is_active ON storage_configs(is_active);
	`

	_, err := r.db.Exec(createTableSQL)
	return err
}

// Create 创建存储配置
func (r *StorageConfigRepository) Create(config *model.StorageConfig) error {
	query := `
		INSERT INTO storage_configs (name, storage_type, config_data, is_active, description, target_agent_id, priority, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`

	result, err := r.db.Exec(query,
		config.Name,
		config.StorageType,
		config.ConfigData,
		config.IsActive,
		config.Description,
		config.TargetAgentID,
		config.Priority,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	config.ID = int(id)
	return nil
}

// GetAll 获取所有存储配置
func (r *StorageConfigRepository) GetAll(storageType string, agentID string) ([]*model.StorageConfig, error) {
	query := `SELECT id, name, storage_type, config_data, is_active, description, target_agent_id, priority, created_at, updated_at FROM storage_configs WHERE 1=1`
	args := []interface{}{}

	if storageType != "" {
		query += " AND storage_type = ?"
		args = append(args, storageType)
	}

	if agentID != "" {
		query += " AND (target_agent_id = ? OR target_agent_id IS NULL)"
		args = append(args, agentID)
	}

	query += " ORDER BY priority DESC, created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*model.StorageConfig
	for rows.Next() {
		config := &model.StorageConfig{}
		var targetAgentID sql.NullString

		err := rows.Scan(
			&config.ID,
			&config.Name,
			&config.StorageType,
			&config.ConfigData,
			&config.IsActive,
			&config.Description,
			&targetAgentID,
			&config.Priority,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if targetAgentID.Valid {
			config.TargetAgentID = &targetAgentID.String
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// GetByID 根据ID获取存储配置
func (r *StorageConfigRepository) GetByID(id int) (*model.StorageConfig, error) {
	query := `SELECT id, name, storage_type, config_data, is_active, description, target_agent_id, priority, created_at, updated_at FROM storage_configs WHERE id = ?`

	config := &model.StorageConfig{}
	var targetAgentID sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&config.ID,
		&config.Name,
		&config.StorageType,
		&config.ConfigData,
		&config.IsActive,
		&config.Description,
		&targetAgentID,
		&config.Priority,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if targetAgentID.Valid {
		config.TargetAgentID = &targetAgentID.String
	}

	return config, nil
}

// GetByAgentID 获取Agent的存储配置（优先返回Agent专属配置，否则返回全局配置）
func (r *StorageConfigRepository) GetByAgentID(agentID string) (*model.StorageConfig, error) {
	// 先查找Agent专属配置
	query := `SELECT id, name, storage_type, config_data, is_active, description, target_agent_id, priority, created_at, updated_at
			  FROM storage_configs
			  WHERE target_agent_id = ? AND is_active = 1
			  ORDER BY priority DESC, created_at DESC
			  LIMIT 1`

	config := &model.StorageConfig{}
	var targetAgentID sql.NullString

	err := r.db.QueryRow(query, agentID).Scan(
		&config.ID,
		&config.Name,
		&config.StorageType,
		&config.ConfigData,
		&config.IsActive,
		&config.Description,
		&targetAgentID,
		&config.Priority,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == nil {
		if targetAgentID.Valid {
			config.TargetAgentID = &targetAgentID.String
		}
		return config, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// 未找到专属配置，查找全局配置（兼容 NULL 和空字符串两种情况）
	query = `SELECT id, name, storage_type, config_data, is_active, description, target_agent_id, priority, created_at, updated_at
			 FROM storage_configs
			 WHERE (target_agent_id IS NULL OR target_agent_id = '') AND is_active = 1
			 ORDER BY priority DESC, created_at DESC
			 LIMIT 1`

	err = r.db.QueryRow(query).Scan(
		&config.ID,
		&config.Name,
		&config.StorageType,
		&config.ConfigData,
		&config.IsActive,
		&config.Description,
		&targetAgentID,
		&config.Priority,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if targetAgentID.Valid {
		config.TargetAgentID = &targetAgentID.String
	}

	return config, nil
}

// Update 更新存储配置
func (r *StorageConfigRepository) Update(config *model.StorageConfig) error {
	query := `
		UPDATE storage_configs
		SET name = ?, storage_type = ?, config_data = ?, is_active = ?, description = ?, target_agent_id = ?, priority = ?, updated_at = datetime('now')
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		config.Name,
		config.StorageType,
		config.ConfigData,
		config.IsActive,
		config.Description,
		config.TargetAgentID,
		config.Priority,
		config.ID,
	)

	return err
}

// Delete 删除存储配置
func (r *StorageConfigRepository) Delete(id int) error {
	query := `DELETE FROM storage_configs WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// ValidateConfig 验证配置数据格式
func ValidateConfig(storageType model.StorageType, configData string) error {
	switch storageType {
	case model.StorageTypeSFTP:
		var config model.SFTPConfig
		if err := json.Unmarshal([]byte(configData), &config); err != nil {
			return fmt.Errorf("SFTP配置格式错误: %v", err)
		}
		if config.Host == "" || config.Port == 0 || config.Username == "" || config.RemotePath == "" {
			return fmt.Errorf("SFTP配置缺少必要字段")
		}
		if config.Password == "" && config.KeyPath == "" && config.KeyContent == "" {
			return fmt.Errorf("SFTP配置必须提供密码或SSH密钥")
		}

	case model.StorageTypeWebDAV:
		var config model.WebDAVConfig
		if err := json.Unmarshal([]byte(configData), &config); err != nil {
			return fmt.Errorf("WebDAV配置格式错误: %v", err)
		}
		if config.Endpoint == "" || config.Username == "" || config.Password == "" || config.RemotePath == "" {
			return fmt.Errorf("WebDAV配置缺少必要字段")
		}

	case model.StorageTypeAliyunOSS:
		var config model.AliyunOSSConfig
		if err := json.Unmarshal([]byte(configData), &config); err != nil {
			return fmt.Errorf("阿里云OSS配置格式错误: %v", err)
		}
		if config.Endpoint == "" || config.AccessKeyID == "" || config.AccessKeySecret == "" || config.BucketName == "" {
			return fmt.Errorf("阿里云OSS配置缺少必要字段")
		}

	case model.StorageTypeTencentCOS:
		var config model.TencentCOSConfig
		if err := json.Unmarshal([]byte(configData), &config); err != nil {
			return fmt.Errorf("腾讯云COS配置格式错误: %v", err)
		}
		if config.Region == "" || config.SecretID == "" || config.SecretKey == "" || config.BucketName == "" {
			return fmt.Errorf("腾讯云COS配置缺少必要字段")
		}

	case model.StorageTypeAWSS3:
		var config model.AWSS3Config
		if err := json.Unmarshal([]byte(configData), &config); err != nil {
			return fmt.Errorf("AWS S3配置格式错误: %v", err)
		}
		if config.Region == "" || config.AccessKeyID == "" || config.SecretAccessKey == "" || config.BucketName == "" {
			return fmt.Errorf("AWS S3配置缺少必要字段")
		}

	case model.StorageTypeSMB:
		var config model.SMBConfig
		if err := json.Unmarshal([]byte(configData), &config); err != nil {
			return fmt.Errorf("SMB配置格式错误: %v", err)
		}
		if config.Host == "" || config.ShareName == "" || config.RemotePath == "" {
			return fmt.Errorf("SMB配置缺少必要字段")
		}

	case model.StorageTypeLocal:
		var config model.LocalConfig
		if err := json.Unmarshal([]byte(configData), &config); err != nil {
			return fmt.Errorf("本地存储配置格式错误: %v", err)
		}
		if config.BasePath == "" {
			return fmt.Errorf("本地存储配置缺少路径")
		}

	default:
		return fmt.Errorf("不支持的存储类型: %s", storageType)
	}

	return nil
}
