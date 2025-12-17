package repository

import (
	"database/sql"
	"fmt"
	"time"

	"doc-scanner-server/internal/model"
)

// ConfigRepository 配置数据访问层
type ConfigRepository struct {
	db *sql.DB
}

// NewConfigRepository 创建新的配置仓库
func NewConfigRepository(db *sql.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

// Create 创建配置
func (r *ConfigRepository) Create(config *model.Config) error {
	query := `
		INSERT INTO configs (config_name, config_type, target_agent_id, config_data, version, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		config.ConfigName,
		config.ConfigType,
		config.TargetAgentID,
		config.ConfigData,
		config.Version,
		config.IsActive,
		config.CreatedAt,
		config.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	return nil
}

// GetByID 根据ID获取配置
func (r *ConfigRepository) GetByID(id int) (*model.Config, error) {
	query := `
		SELECT id, config_name, config_type, target_agent_id, config_data, version, is_active, created_at, updated_at
		FROM configs WHERE id = ?
	`

	var config model.Config

	err := r.db.QueryRow(query, id).Scan(
		&config.ID,
		&config.ConfigName,
		&config.ConfigType,
		&config.TargetAgentID,
		&config.ConfigData,
		&config.Version,
		&config.IsActive,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return &config, nil
}

// GetAll 获取所有配置
func (r *ConfigRepository) GetAll(configType string) ([]*model.Config, error) {
	query := `
		SELECT id, config_name, config_type, target_agent_id, config_data, version, is_active, created_at, updated_at
		FROM configs
		WHERE (? = '' OR config_type = ?)
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, configType, configType)
	if err != nil {
		return nil, fmt.Errorf("failed to query configs: %w", err)
	}
	defer rows.Close()

	var configs []*model.Config
	for rows.Next() {
		var config model.Config

		err := rows.Scan(
			&config.ID,
			&config.ConfigName,
			&config.ConfigType,
			&config.TargetAgentID,
			&config.ConfigData,
			&config.Version,
			&config.IsActive,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// Update 更新配置
func (r *ConfigRepository) Update(config *model.Config) error {
	query := `
		UPDATE configs
		SET config_name = ?, config_data = ?, version = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		config.ConfigName,
		config.ConfigData,
		config.Version,
		config.IsActive,
		time.Now(),
		config.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	return nil
}

// Delete 删除配置
func (r *ConfigRepository) Delete(id int) error {
	query := "DELETE FROM configs WHERE id = ?"

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	return nil
}

// GetByAgentID 获取指定Agent的配置
func (r *ConfigRepository) GetByAgentID(agentID string) (*model.Config, error) {
	query := `
		SELECT id, config_name, config_type, target_agent_id, config_data, version, is_active, created_at, updated_at
		FROM configs
		WHERE (target_agent_id = ? OR config_type = 'global') AND is_active = 1
		ORDER BY config_type DESC
		LIMIT 1
	`

	var config model.Config

	err := r.db.QueryRow(query, agentID).Scan(
		&config.ID,
		&config.ConfigName,
		&config.ConfigType,
		&config.TargetAgentID,
		&config.ConfigData,
		&config.Version,
		&config.IsActive,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return &config, nil
}
