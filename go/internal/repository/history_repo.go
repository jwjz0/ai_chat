package repository

import (
	"Voice_Assistant/internal/data/sqlite"
	"Voice_Assistant/internal/model"
	"context"
	"database/sql"
)

// HistoryRepo 历史记录数据访问接口
type HistoryRepo interface {
	SelectByAssistantID(ctx context.Context, assistantID string) (*model.History, error)
	DeleteByAssistantID(ctx context.Context, assistantID string) error
	SaveByAssistantID(ctx context.Context, assistantID string, message model.Message) error
	UpdateAssistantTimestamp(ctx context.Context, assistantID string, timestamp string) error
}

// NewHistoryRepo 创建历史仓库实例（依赖注入）
func NewHistoryRepo(db *sql.DB) HistoryRepo {
	return sqlite.NewHistorySQLiteRepo(db)
}
