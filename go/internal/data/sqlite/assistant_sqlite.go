package sqlite

import (
	"Voice_Assistant/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// AssistantSQLiteRepo 实现AssistantRepo接口
type AssistantSQLiteRepo struct {
	db *sql.DB
}

// NewAssistantSQLiteRepo 创建实例
func NewAssistantSQLiteRepo(db *sql.DB) *AssistantSQLiteRepo {
	return &AssistantSQLiteRepo{db: db}
}

// SelectAll 查询所有助手
func (r *AssistantSQLiteRepo) SelectAll(ctx context.Context) ([]model.Assistant, error) {
	query := "SELECT id, name, description, prompt, gmt_create, gmt_modified, time_stamp FROM assistants;"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询助手失败: %w", err)
	}
	defer rows.Close()

	var assistants []model.Assistant
	for rows.Next() {
		var a model.Assistant
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Description, &a.Prompt,
			&a.GmtCreate, &a.GmtModified, &a.TimeStamp,
		); err != nil {
			return nil, fmt.Errorf("扫描助手数据失败: %w", err)
		}
		assistants = append(assistants, a)
	}
	return assistants, rows.Err()
}

// DeleteByID 按ID删除助手（级联删除历史）
func (r *AssistantSQLiteRepo) DeleteByID(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 删除助手
	res, err := tx.ExecContext(ctx, "DELETE FROM assistants WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除助手失败: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return errors.New("助手不存在")
	}

	// 提交事务（历史记录通过外键级联删除）
	return tx.Commit()
}

// Save 保存新助手
func (r *AssistantSQLiteRepo) Save(ctx context.Context, a *model.Assistant) (*model.Assistant, error) {
	query := `
	INSERT INTO assistants (id, name, description, prompt, gmt_create, gmt_modified, time_stamp)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.Name, a.Description, a.Prompt,
		a.GmtCreate, a.GmtModified, a.TimeStamp,
	)
	if err != nil {
		return nil, fmt.Errorf("保存助手失败: %w", err)
	}
	return a, nil
}

// UpdateByID 按ID更新助手
func (r *AssistantSQLiteRepo) UpdateByID(ctx context.Context, id string, a *model.Assistant) (*model.Assistant, error) {
	query := `
	UPDATE assistants SET name = ?, description = ?, prompt = ?, 
	gmt_create = ?, gmt_modified = ?, time_stamp = ? 
	WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		a.Name, a.Description, a.Prompt,
		a.GmtCreate, a.GmtModified, a.TimeStamp, id,
	)
	if err != nil {
		return nil, fmt.Errorf("更新助手失败: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, errors.New("助手不存在")
	}
	return a, nil
}
