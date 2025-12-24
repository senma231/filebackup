package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"doc-scanner-server/internal/model"
)

// FileRepository 文件数据访问层
type FileRepository struct {
	db *sql.DB
}

// NewFileRepository 创建新的文件仓库
func NewFileRepository(db *sql.DB) *FileRepository {
	return &FileRepository{db: db}
}

// Create 创建文件上传记录
func (r *FileRepository) Create(file *model.FileUpload) error {
	query := `
		INSERT INTO file_uploads (agent_id, local_path, remote_path, file_name, file_size, file_type, upload_status, upload_start_time, upload_end_time, retry_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		file.AgentID,
		file.LocalPath,
		file.RemotePath,
		file.FileName,
		file.FileSize,
		file.FileType,
		file.UploadStatus,
		file.UploadStartTime,
		file.UploadEndTime,
		file.RetryCount,
		file.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create file upload record: %w", err)
	}

	return nil
}

// GetByID 根据ID获取文件记录
func (r *FileRepository) GetByID(id int) (*model.FileUpload, error) {
	query := `
		SELECT id, agent_id, local_path, remote_path, file_name, file_size, file_type, upload_status, error_message, upload_start_time, upload_end_time, retry_count, created_at
		FROM file_uploads WHERE id = ?
	`

	var file model.FileUpload
	var uploadStartTime, uploadEndTime sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&file.ID,
		&file.AgentID,
		&file.LocalPath,
		&file.RemotePath,
		&file.FileName,
		&file.FileSize,
		&file.FileType,
		&file.UploadStatus,
		&file.ErrorMessage,
		&uploadStartTime,
		&uploadEndTime,
		&file.RetryCount,
		&file.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file upload: %w", err)
	}

	if uploadStartTime.Valid {
		file.UploadStartTime = &uploadStartTime.Time
	}

	if uploadEndTime.Valid {
		file.UploadEndTime = &uploadEndTime.Time
	}

	return &file, nil
}

// GetAll 获取所有文件记录（分页）
func (r *FileRepository) GetAll(page, perPage int, agentID, status string) ([]*model.FileUpload, int64, error) {
	offset := (page - 1) * perPage

	// 构建查询条件
	whereClause := ""
	args := []interface{}{}
	conditions := []string{}

	if agentID != "" {
		conditions = append(conditions, "agent_id = ?")
		args = append(args, agentID)
	}

	if status != "" && status != "all" {
		conditions = append(conditions, "upload_status = ?")
		args = append(args, status)
	}

	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 获取总数
	var total int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM file_uploads %s", whereClause)
	err := r.db.QueryRow(query, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count files: %w", err)
	}

	// 获取分页数据
	query = fmt.Sprintf(`
		SELECT id, agent_id, local_path, remote_path, file_name, file_size, file_type, upload_status, error_message, upload_start_time, upload_end_time, retry_count, created_at
		FROM file_uploads %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, perPage, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	var files []*model.FileUpload
	for rows.Next() {
		var file model.FileUpload
		var uploadStartTime, uploadEndTime sql.NullTime

		err := rows.Scan(
			&file.ID,
			&file.AgentID,
			&file.LocalPath,
			&file.RemotePath,
			&file.FileName,
			&file.FileSize,
			&file.FileType,
			&file.UploadStatus,
			&file.ErrorMessage,
			&uploadStartTime,
			&uploadEndTime,
			&file.RetryCount,
			&file.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan file: %w", err)
		}

		if uploadStartTime.Valid {
			file.UploadStartTime = &uploadStartTime.Time
		}

		if uploadEndTime.Valid {
			file.UploadEndTime = &uploadEndTime.Time
		}

		files = append(files, &file)
	}

	return files, total, nil
}

// UpdateStatus 更新上传状态
func (r *FileRepository) UpdateStatus(id int, status, errorMessage string) error {
	query := `
		UPDATE file_uploads
		SET upload_status = ?, error_message = ?, upload_end_time = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, status, errorMessage, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}

	return nil
}

// UpdateProgress 更新上传进度
func (r *FileRepository) UpdateProgress(agentID, fileName string, progress int) error {
	query := `
		UPDATE file_uploads
		SET upload_status = 'uploading'
		WHERE agent_id = ? AND file_name = ?
	`

	_, err := r.db.Exec(query, agentID, fileName)
	if err != nil {
		return fmt.Errorf("failed to update upload progress: %w", err)
	}

	return nil
}

// GetByAgentID 获取指定Agent的文件记录
func (r *FileRepository) GetByAgentID(agentID string, limit int) ([]*model.FileUpload, error) {
	query := `
		SELECT id, agent_id, local_path, remote_path, file_name, file_size, file_type, upload_status, error_message, upload_start_time, upload_end_time, retry_count, created_at
		FROM file_uploads
		WHERE agent_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	var files []*model.FileUpload
	for rows.Next() {
		var file model.FileUpload
		var uploadStartTime, uploadEndTime sql.NullTime

		err := rows.Scan(
			&file.ID,
			&file.AgentID,
			&file.LocalPath,
			&file.RemotePath,
			&file.FileName,
			&file.FileSize,
			&file.FileType,
			&file.UploadStatus,
			&file.ErrorMessage,
			&uploadStartTime,
			&uploadEndTime,
			&file.RetryCount,
			&file.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		if uploadStartTime.Valid {
			file.UploadStartTime = &uploadStartTime.Time
		}

		if uploadEndTime.Valid {
			file.UploadEndTime = &uploadEndTime.Time
		}

		files = append(files, &file)
	}

	return files, nil
}

// UpdateStatusByFileName 根据文件名更新状态
func (r *FileRepository) UpdateStatusByFileName(agentID, fileName, status, errorMessage string) error {
	query := `
		UPDATE file_uploads
		SET upload_status = ?, error_message = ?, upload_end_time = CASE WHEN ? = 'success' THEN CURRENT_TIMESTAMP ELSE upload_end_time END
		WHERE agent_id = ? AND file_name = ?
	`

	_, err := r.db.Exec(query, status, errorMessage, status, agentID, fileName)
	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}

	return nil
}

// CountByAgentAndStatus 统计指定Agent和状态的文件数量
func (r *FileRepository) CountByAgentAndStatus(agentID, status string) (int64, error) {
	var count int64
	query := `
		SELECT COUNT(*)
		FROM file_uploads
		WHERE agent_id = ? AND upload_status = ?
	`
	err := r.db.QueryRow(query, agentID, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count files by agent and status: %w", err)
	}
	return count, nil
}

// GetLastUploadTime 获取指定Agent的最后上传时间
func (r *FileRepository) GetLastUploadTime(agentID string) (*time.Time, error) {
	query := `
		SELECT MAX(upload_end_time)
		FROM file_uploads
		WHERE agent_id = ? AND upload_status = 'success'
	`
	var uploadTime sql.NullTime
	err := r.db.QueryRow(query, agentID).Scan(&uploadTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get last upload time: %w", err)
	}
	if uploadTime.Valid {
		return &uploadTime.Time, nil
	}
	return nil, nil
}
