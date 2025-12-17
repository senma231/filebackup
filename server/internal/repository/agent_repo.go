package repository

import (
	"database/sql"
	"fmt"
	"time"

	"doc-scanner-server/internal/model"
)

// AgentRepository Agent数据访问层
type AgentRepository struct {
	db *sql.DB
}

// NewAgentRepository 创建新的Agent仓库
func NewAgentRepository(db *sql.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

// Create 创建Agent
func (r *AgentRepository) Create(agent *model.Agent) error {
	query := `
		INSERT INTO agents (agent_id, email, email_prefix, hostname, ip_address, os_version, agent_version, status, last_heartbeat, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		agent.AgentID,
		agent.Email,
		agent.EmailPrefix,
		agent.Hostname,
		agent.IPAddress,
		agent.OSVersion,
		agent.AgentVersion,
		agent.Status,
		agent.LastHeartbeat,
		agent.CreatedAt,
		agent.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

// GetByID 根据ID获取Agent
func (r *AgentRepository) GetByID(agentID string) (*model.Agent, error) {
	query := `
		SELECT id, agent_id, email, email_prefix, hostname, ip_address, os_version, agent_version, status, last_heartbeat, created_at, updated_at
		FROM agents WHERE agent_id = ?
	`

	var agent model.Agent
	var lastHeartbeat sql.NullTime

	err := r.db.QueryRow(query, agentID).Scan(
		&agent.ID,
		&agent.AgentID,
		&agent.Email,
		&agent.EmailPrefix,
		&agent.Hostname,
		&agent.IPAddress,
		&agent.OSVersion,
		&agent.AgentVersion,
		&agent.Status,
		&lastHeartbeat,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if lastHeartbeat.Valid {
		agent.LastHeartbeat = &lastHeartbeat.Time
	}

	return &agent, nil
}

// GetAll 获取所有Agent
func (r *AgentRepository) GetAll(page, perPage int, status string) ([]*model.Agent, int64, error) {
	offset := (page - 1) * perPage

	// 构建查询条件
	whereClause := ""
	args := []interface{}{}
	if status != "all" {
		whereClause = "WHERE status = ?"
		args = append(args, status)
	}

	// 获取总数
	var total int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM agents %s", whereClause)
	err := r.db.QueryRow(query, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count agents: %w", err)
	}

	// 获取分页数据
	query = fmt.Sprintf(`
		SELECT id, agent_id, email, email_prefix, hostname, ip_address, os_version, agent_version, status, last_heartbeat, created_at, updated_at
		FROM agents %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, perPage, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []*model.Agent
	for rows.Next() {
		var agent model.Agent
		var lastHeartbeat sql.NullTime

		err := rows.Scan(
			&agent.ID,
			&agent.AgentID,
			&agent.Email,
			&agent.EmailPrefix,
			&agent.Hostname,
			&agent.IPAddress,
			&agent.OSVersion,
			&agent.AgentVersion,
			&agent.Status,
			&lastHeartbeat,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan agent: %w", err)
		}

		if lastHeartbeat.Valid {
			agent.LastHeartbeat = &lastHeartbeat.Time
		}

		agents = append(agents, &agent)
	}

	return agents, total, nil
}

// Update 更新Agent
func (r *AgentRepository) Update(agent *model.Agent) error {
	query := `
		UPDATE agents
		SET email = ?, email_prefix = ?, hostname = ?, ip_address = ?, os_version = ?, agent_version = ?, status = ?, last_heartbeat = ?, updated_at = ?
		WHERE agent_id = ?
	`

	_, err := r.db.Exec(query,
		agent.Email,
		agent.EmailPrefix,
		agent.Hostname,
		agent.IPAddress,
		agent.OSVersion,
		agent.AgentVersion,
		agent.Status,
		agent.LastHeartbeat,
		agent.UpdatedAt,
		agent.AgentID,
	)

	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	return nil
}

// UpdateHeartbeat 更新心跳
func (r *AgentRepository) UpdateHeartbeat(agentID string, stats model.SystemStats) error {
	now := time.Now()

	// 更新Agent状态
	query := `
		UPDATE agents
		SET status = 'online', last_heartbeat = ?, updated_at = ?
		WHERE agent_id = ?
	`

	_, err := r.db.Exec(query, now, now, agentID)
	if err != nil {
		return fmt.Errorf("failed to update agent heartbeat: %w", err)
	}

	// 记录心跳日志
	heartbeat := &model.Heartbeat{
		AgentID:     agentID,
		Status:      "online",
		CPUUsage:    stats.CPUUsage,
		MemoryUsage: stats.MemoryUsage,
		DiskUsage:   stats.DiskUsage,
		ScanCount:   stats.ScanCount,
		UploadCount: stats.UploadCount,
		ErrorCount:  stats.ErrorCount,
		Timestamp:   now,
	}

	return r.CreateHeartbeat(heartbeat)
}

// CreateHeartbeat 创建心跳记录
func (r *AgentRepository) CreateHeartbeat(heartbeat *model.Heartbeat) error {
	query := `
		INSERT INTO heartbeats (agent_id, status, cpu_usage, memory_usage, disk_usage, scan_count, upload_count, error_count, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		heartbeat.AgentID,
		heartbeat.Status,
		heartbeat.CPUUsage,
		heartbeat.MemoryUsage,
		heartbeat.DiskUsage,
		heartbeat.ScanCount,
		heartbeat.UploadCount,
		heartbeat.ErrorCount,
		heartbeat.Timestamp,
		heartbeat.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create heartbeat: %w", err)
	}

	return nil
}

// GetHeartbeats 获取心跳记录
func (r *AgentRepository) GetHeartbeats(agentID string, limit int) ([]*model.Heartbeat, error) {
	query := `
		SELECT id, agent_id, status, cpu_usage, memory_usage, disk_usage, scan_count, upload_count, error_count, timestamp, created_at
		FROM heartbeats
		WHERE agent_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query heartbeats: %w", err)
	}
	defer rows.Close()

	var heartbeats []*model.Heartbeat
	for rows.Next() {
		var hb model.Heartbeat

		err := rows.Scan(
			&hb.ID,
			&hb.AgentID,
			&hb.Status,
			&hb.CPUUsage,
			&hb.MemoryUsage,
			&hb.DiskUsage,
			&hb.ScanCount,
			&hb.UploadCount,
			&hb.ErrorCount,
			&hb.Timestamp,
			&hb.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan heartbeat: %w", err)
		}

		heartbeats = append(heartbeats, &hb)
	}

	return heartbeats, nil
}

// Delete 删除Agent
func (r *AgentRepository) Delete(agentID string) error {
	query := "DELETE FROM agents WHERE agent_id = ?"

	_, err := r.db.Exec(query, agentID)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// GetOnlineCount 获取在线Agent数量
func (r *AgentRepository) GetOnlineCount() (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM agents
		WHERE status = 'online' AND last_heartbeat > datetime('now', '-5 minutes')
	`

	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get online count: %w", err)
	}

	return count, nil
}
