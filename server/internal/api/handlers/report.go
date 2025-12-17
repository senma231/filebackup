package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"doc-scanner-server/internal/model"
	"doc-scanner-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// ReportHandler 报告处理器
type ReportHandler struct {
	agentRepo *repository.AgentRepository
	fileRepo  *repository.FileRepository
}

// NewReportHandler 创建新的报告处理器
func NewReportHandler(db *sql.DB) *ReportHandler {
	return &ReportHandler{
		agentRepo: repository.NewAgentRepository(db),
		fileRepo:  repository.NewFileRepository(db),
	}
}

// GetStats 获取系统统计信息
func (h *ReportHandler) GetStats(c *gin.Context) {
	stats := gin.H{}

	// 统计Agent数量
	totalAgents, _ := h.getTotalAgents()
	onlineAgents, _ := h.agentRepo.GetOnlineCount()
	stats["total_agents"] = totalAgents
	stats["online_agents"] = onlineAgents
	stats["offline_agents"] = totalAgents - onlineAgents

	// 统计文件上传
	totalFiles, totalSize, _ := h.getFileStats()
	stats["total_files"] = totalFiles
	stats["total_size"] = totalSize
	stats["total_size_formatted"] = formatBytes(totalSize)

	// 统计成功率
	successRate, _ := h.getUploadSuccessRate()
	stats["upload_success_rate"] = fmt.Sprintf("%.2f%%", successRate)

	// 最近的活跃Agent
	recentActiveAgents, _ := h.getRecentActiveAgents(10)
	stats["recent_active_agents"] = recentActiveAgents

	c.JSON(http.StatusOK, model.Success(stats))
}

// GetAgentStats 获取Agent统计
func (h *ReportHandler) GetAgentStats(c *gin.Context) {
	agentID := c.Param("agent_id")

	if agentID == "" {
		// 获取所有Agent统计
		agents, _, _ := h.agentRepo.GetAll(1, 100, "all")

		stats := []gin.H{}
		for _, agent := range agents {
			agentStats := h.getAgentIndividualStats(agent)
			stats = append(stats, agentStats)
		}

		c.JSON(http.StatusOK, model.Success(stats))
	} else {
		// 获取单个Agent统计
		agent, err := h.agentRepo.GetByID(agentID)
		if err != nil || agent == nil {
			c.JSON(http.StatusNotFound, model.Error(http.StatusNotFound, "Agent not found"))
			return
		}

		stats := h.getAgentIndividualStats(agent)
		c.JSON(http.StatusOK, model.Success(stats))
	}
}

// GetFileStats 获取文件统计
func (h *ReportHandler) GetFileStats(c *gin.Context) {
	agentID := c.DefaultQuery("agent_id", "")
	days := parseInt(c.DefaultQuery("days", "30"))

	stats := gin.H{}

	// 按日期统计
	dateStats, _ := h.getFileStatsByDate(agentID, days)
	stats["by_date"] = dateStats

	// 按文件类型统计
	typeStats, _ := h.getFileStatsByType(agentID)
	stats["by_type"] = typeStats

	// 按状态统计
	statusStats, _ := h.getFileStatsByStatus(agentID)
	stats["by_status"] = statusStats

	c.JSON(http.StatusOK, model.Success(stats))
}

// 辅助方法

// getTotalAgents 获取总Agent数
func (h *ReportHandler) getTotalAgents() (int, error) {
	agents, _, err := h.agentRepo.GetAll(1, 1000, "all")
	return len(agents), err
}

// getFileStats 获取文件统计
func (h *ReportHandler) getFileStats() (int, int64, error) {
	files, _, err := h.fileRepo.GetAll(1, 10000, "", "all")
	if err != nil {
		return 0, 0, err
	}

	var totalFiles int
	var totalSize int64
	for _, file := range files {
		totalFiles++
		totalSize += file.FileSize
	}

	return totalFiles, totalSize, nil
}

// getUploadSuccessRate 获取上传成功率
func (h *ReportHandler) getUploadSuccessRate() (float64, error) {
	files, _, err := h.fileRepo.GetAll(1, 10000, "", "all")
	if err != nil {
		return 0, err
	}

	if len(files) == 0 {
		return 0, nil
	}

	var successCount int
	for _, file := range files {
		if file.UploadStatus == model.FileStatusSuccess {
			successCount++
		}
	}

	return float64(successCount) / float64(len(files)) * 100, nil
}

// getRecentActiveAgents 获取最近活跃的Agent
func (h *ReportHandler) getRecentActiveAgents(limit int) ([]gin.H, error) {
	agents, _, err := h.agentRepo.GetAll(1, limit, "all")
	if err != nil {
		return nil, err
	}

	var recentAgents []gin.H
	for _, agent := range agents {
		if agent.IsOnline() {
			recentAgents = append(recentAgents, gin.H{
				"agent_id":      agent.AgentID,
				"email":         agent.Email,
				"email_prefix":  agent.EmailPrefix,
				"hostname":      agent.Hostname,
				"last_heartbeat": agent.LastHeartbeat,
				"status":        agent.Status,
			})
		}
	}

	return recentAgents, nil
}

// getAgentIndividualStats 获取单个Agent统计
func (h *ReportHandler) getAgentIndividualStats(agent *model.Agent) gin.H {
	stats := gin.H{
		"agent_id":       agent.AgentID,
		"email":          agent.Email,
		"email_prefix":   agent.EmailPrefix,
		"hostname":       agent.Hostname,
		"status":         agent.Status,
		"last_heartbeat": agent.LastHeartbeat,
		"created_at":     agent.CreatedAt,
	}

	// 获取该Agent的文件统计
	files, _, _ := h.fileRepo.GetAll(1, 10000, agent.AgentID, "all")

	var totalFiles, successFiles, failedFiles int
	var totalSize int64
	for _, file := range files {
		totalFiles++
		totalSize += file.FileSize
		if file.UploadStatus == model.FileStatusSuccess {
			successFiles++
		} else if file.UploadStatus == model.FileStatusFailed {
			failedFiles++
		}
	}

	stats["total_files"] = totalFiles
	stats["success_files"] = successFiles
	stats["failed_files"] = failedFiles
	stats["total_size"] = totalSize
	stats["total_size_formatted"] = formatBytes(totalSize)

	if totalFiles > 0 {
		stats["success_rate"] = fmt.Sprintf("%.2f%%", float64(successFiles)/float64(totalFiles)*100)
	} else {
		stats["success_rate"] = "0.00%"
	}

	return stats
}

// getFileStatsByDate 按日期统计文件
func (h *ReportHandler) getFileStatsByDate(agentID string, days int) ([]gin.H, error) {
	// TODO: 实现按日期统计
	return []gin.H{}, nil
}

// getFileStatsByType 按类型统计文件
func (h *ReportHandler) getFileStatsByType(agentID string) (gin.H, error) {
	files, _, err := h.fileRepo.GetAll(1, 10000, agentID, "all")
	if err != nil {
		return nil, err
	}

	stats := gin.H{}
	for _, file := range files {
		fileType := file.FileType
		if fileType == "" {
			fileType = "unknown"
		}

		if _, exists := stats[fileType]; !exists {
			stats[fileType] = gin.H{
				"count": 0,
				"size":  0,
			}
		}

		typeStats := stats[fileType].(gin.H)
		typeStats["count"] = typeStats["count"].(int) + 1
		typeStats["size"] = typeStats["size"].(int64) + file.FileSize
	}

	return stats, nil
}

// getFileStatsByStatus 按状态统计文件
func (h *ReportHandler) getFileStatsByStatus(agentID string) (gin.H, error) {
	files, _, err := h.fileRepo.GetAll(1, 10000, agentID, "all")
	if err != nil {
		return nil, err
	}

	stats := gin.H{
		model.FileStatusPending:    0,
		model.FileStatusUploading:  0,
		model.FileStatusSuccess:    0,
		model.FileStatusFailed:     0,
	}

	for _, file := range files {
		stats[file.UploadStatus] = stats[file.UploadStatus].(int) + 1
	}

	return stats, nil
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	if bytes >= GB {
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	} else if bytes >= MB {
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	} else if bytes >= KB {
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	} else {
		return fmt.Sprintf("%d B", bytes)
	}
}
