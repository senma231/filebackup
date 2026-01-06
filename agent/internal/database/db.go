package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB Agent本地数据库
type DB struct {
	*sql.DB
}

// FileRecord 已上传文件记录
type FileRecord struct {
	ID           int64
	FilePath     string    // 文件完整路径
	FileSize     int64     // 文件大小
	ModTime      time.Time // 文件修改时间
	RemotePath   string    // 远程路径
	UploadTime   time.Time // 上传时间
	LastCheckTime time.Time // 最后检查时间
}

// Open 打开或创建数据库
func Open(dbPath string) (*DB, error) {
	// 确保数据库目录存在
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// 打开数据库
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{DB: sqlDB}

	// 初始化表结构
	if err := db.initTables(); err != nil {
		sqlDB.Close()
		return nil, err
	}

	return db, nil
}

// initTables 初始化数据库表
func (db *DB) initTables() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS uploaded_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL UNIQUE,
		file_size INTEGER NOT NULL,
		mod_time DATETIME NOT NULL,
		remote_path TEXT NOT NULL,
		upload_time DATETIME NOT NULL,
		last_check_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(file_path)
	);

	CREATE INDEX IF NOT EXISTS idx_file_path ON uploaded_files(file_path);
	CREATE INDEX IF NOT EXISTS idx_mod_time ON uploaded_files(mod_time);
	CREATE INDEX IF NOT EXISTS idx_upload_time ON uploaded_files(upload_time);
	`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// IsFileUploaded 检查文件是否已上传且未修改
func (db *DB) IsFileUploaded(filePath string, modTime time.Time) (bool, error) {
	var recordModTime time.Time
	query := `SELECT mod_time FROM uploaded_files WHERE file_path = ?`

	err := db.QueryRow(query, filePath).Scan(&recordModTime)
	if err == sql.ErrNoRows {
		// 文件未上传过
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to query file: %w", err)
	}

	// 比较修改时间（精确到秒）
	recordModTimeSec := recordModTime.Truncate(time.Second)
	modTimeSec := modTime.Truncate(time.Second)

	return recordModTimeSec.Equal(modTimeSec), nil
}

// RecordUpload 记录文件上传
func (db *DB) RecordUpload(filePath string, fileSize int64, modTime time.Time, remotePath string) error {
	query := `
		INSERT OR REPLACE INTO uploaded_files (file_path, file_size, mod_time, remote_path, upload_time, last_check_time)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := db.Exec(query, filePath, fileSize, modTime, remotePath, now, now)
	if err != nil {
		return fmt.Errorf("failed to record upload: %w", err)
	}

	return nil
}

// UpdateCheckTime 更新文件最后检查时间
func (db *DB) UpdateCheckTime(filePath string) error {
	query := `UPDATE uploaded_files SET last_check_time = ? WHERE file_path = ?`
	_, err := db.Exec(query, time.Now(), filePath)
	return err
}

// GetUploadedCount 获取已上传文件总数
func (db *DB) GetUploadedCount() (int64, error) {
	var count int64
	err := db.QueryRow(`SELECT COUNT(*) FROM uploaded_files`).Scan(&count)
	return count, err
}

// CleanOldRecords 清理长时间未检查的记录（文件可能已被删除）
func (db *DB) CleanOldRecords(daysOld int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -daysOld)
	result, err := db.Exec(`DELETE FROM uploaded_files WHERE last_check_time < ?`, cutoffTime)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
