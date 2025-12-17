package repository

import (
	"database/sql"
	"fmt"
	"time"
)

// NewDB 创建新的数据库连接
func NewDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)  // 连接最大生命周期

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Migrate 运行数据库迁移
func Migrate(db *sql.DB) error {
	// 创建agents表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id VARCHAR(64) UNIQUE NOT NULL,
			email VARCHAR(255),
			email_prefix VARCHAR(255),
			hostname VARCHAR(255) NOT NULL,
			ip_address VARCHAR(45) NOT NULL,
			os_version VARCHAR(255),
			agent_version VARCHAR(50),
			status VARCHAR(20) DEFAULT 'online',
			last_heartbeat DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create agents table: %w", err)
	}

	// 创建configs表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			config_name VARCHAR(255) NOT NULL,
			config_type VARCHAR(50) NOT NULL,
			target_agent_id VARCHAR(64),
			config_data TEXT NOT NULL,
			version INTEGER DEFAULT 1,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (target_agent_id) REFERENCES agents(agent_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create configs table: %w", err)
	}

	// 创建file_uploads表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS file_uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id VARCHAR(64) NOT NULL,
			local_path VARCHAR(1024) NOT NULL,
			remote_path VARCHAR(1024) NOT NULL,
			file_name VARCHAR(255) NOT NULL,
			file_size BIGINT NOT NULL,
			file_type VARCHAR(50),
			upload_status VARCHAR(20) DEFAULT 'pending',
			error_message TEXT,
			upload_start_time DATETIME,
			upload_end_time DATETIME,
			retry_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (agent_id) REFERENCES agents(agent_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create file_uploads table: %w", err)
	}

	// 创建heartbeats表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS heartbeats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id VARCHAR(64) NOT NULL,
			status VARCHAR(20) NOT NULL,
			cpu_usage DECIMAL(5,2),
			memory_usage BIGINT,
			disk_usage BIGINT,
			scan_count INTEGER DEFAULT 0,
			upload_count INTEGER DEFAULT 0,
			error_count INTEGER DEFAULT 0,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (agent_id) REFERENCES agents(agent_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create heartbeats table: %w", err)
	}

	// 创建system_logs表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS system_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			log_level VARCHAR(20) NOT NULL,
			module VARCHAR(50) NOT NULL,
			message TEXT NOT NULL,
			agent_id VARCHAR(64),
			stack_trace TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (agent_id) REFERENCES agents(agent_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create system_logs table: %w", err)
	}

	// 创建file_stats表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS file_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id VARCHAR(64) NOT NULL,
			stat_date DATE NOT NULL,
			total_files INTEGER DEFAULT 0,
			total_size BIGINT DEFAULT 0,
			doc_count INTEGER DEFAULT 0,
			excel_count INTEGER DEFAULT 0,
			ppt_count INTEGER DEFAULT 0,
			pdf_count INTEGER DEFAULT 0,
			txt_count INTEGER DEFAULT 0,
			uploaded_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (agent_id) REFERENCES agents(agent_id),
			UNIQUE(agent_id, stat_date)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create file_stats table: %w", err)
	}

	// 创建索引
	if err := createIndexes(db); err != nil {
		return err
	}

	return nil
}

// createIndexes 创建数据库索引
func createIndexes(db *sql.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status)",
		"CREATE INDEX IF NOT EXISTS idx_agents_last_heartbeat ON agents(last_heartbeat)",
		"CREATE INDEX IF NOT EXISTS idx_configs_type ON configs(config_type)",
		"CREATE INDEX IF NOT EXISTS idx_configs_agent ON configs(target_agent_id)",
		"CREATE INDEX IF NOT EXISTS idx_configs_active ON configs(is_active)",
		"CREATE INDEX IF NOT EXISTS idx_file_uploads_agent ON file_uploads(agent_id)",
		"CREATE INDEX IF NOT EXISTS idx_file_uploads_status ON file_uploads(upload_status)",
		"CREATE INDEX IF NOT EXISTS idx_file_uploads_created ON file_uploads(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_heartbeats_agent ON heartbeats(agent_id)",
		"CREATE INDEX IF NOT EXISTS idx_heartbeats_timestamp ON heartbeats(timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs(log_level)",
		"CREATE INDEX IF NOT EXISTS idx_system_logs_agent ON system_logs(agent_id)",
		"CREATE INDEX IF NOT EXISTS idx_system_logs_created ON system_logs(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_file_stats_agent ON file_stats(agent_id)",
		"CREATE INDEX IF NOT EXISTS idx_file_stats_date ON file_stats(stat_date)",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// Close 关闭数据库连接
func Close(db *sql.DB) error {
	return db.Close()
}
