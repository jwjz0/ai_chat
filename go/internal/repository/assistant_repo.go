package repository

import (
	"Voice_Assistant/internal/data/sqlite"
	"Voice_Assistant/internal/model"
	"context"
	"database/sql"
)

// AssistantRepo 助手数据访问接口
type AssistantRepo interface {
	SelectAll(ctx context.Context) ([]model.Assistant, error)
	DeleteByID(ctx context.Context, id string) error
	Save(ctx context.Context, assistant *model.Assistant) (*model.Assistant, error)
	UpdateByID(ctx context.Context, id string, assistant *model.Assistant) (*model.Assistant, error)
}

// NewAssistantRepo 创建助手仓库实例（依赖注入）
func NewAssistantRepo(db *sql.DB) AssistantRepo {
	return sqlite.NewAssistantSQLiteRepo(db)
}
