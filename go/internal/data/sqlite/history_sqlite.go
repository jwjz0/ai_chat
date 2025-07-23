package sqlite

import (
	"Voice_Assistant/internal/model"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

// HistorySQLiteRepo 实现HistoryRepo接口
type HistorySQLiteRepo struct {
	db *sql.DB
}

// NewHistorySQLiteRepo 创建实例
func NewHistorySQLiteRepo(db *sql.DB) *HistorySQLiteRepo {
	return &HistorySQLiteRepo{db: db}
}

func (r *HistorySQLiteRepo) SelectByAssistantID(ctx context.Context, aid string) (*model.History, error) {
	query := "SELECT messages FROM histories WHERE assistant_id = ?"
	var messagesJSON string
	err := r.db.QueryRowContext(ctx, query, aid).Scan(&messagesJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err // 使用包级错误变量
		}
		return nil, fmt.Errorf("查询历史失败: %w", err)
	}

	// 关键修复：明确声明为[]model.Message切片（接收JSON数组）
	var messages []model.Message
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		// 打印详细错误信息（方便调试）
		return nil, fmt.Errorf("解析消息失败: 原始JSON=%s, 错误=%w", messagesJSON, err)
	}

	return &model.History{
		AssistantID: aid,
		Messages:    messages, // 正确赋值切片
	}, nil
}

// DeleteByAssistantID 按助手ID删除历史（核心修改：不存在时不报错）
func (r *HistorySQLiteRepo) DeleteByAssistantID(ctx context.Context, aid string) error {
	log.Printf("[SQLite] 删除助手 %s 的所有历史消息", aid)

	_, err := r.db.ExecContext(ctx, "DELETE FROM histories WHERE assistant_id = ?", aid)
	if err != nil {
		log.Printf("[SQLite] 删除历史失败: %v", err)
		return fmt.Errorf("删除历史失败: %w", err)
	}

	log.Printf("[SQLite] 成功删除助手 %s 的历史记录", aid)
	return nil
}

// SaveByAssistantID 追加消息到历史（恢复原设计）
func (r *HistorySQLiteRepo) SaveByAssistantID(ctx context.Context, aid string, msg model.Message) error {
	log.Printf("[SQLite] 开始保存助手 %s 的新消息", aid)

	// 1. 查询现有历史
	history, err := r.SelectByAssistantID(ctx, aid)
	var messages []model.Message

	if err == nil {
		messages = history.Messages
		log.Printf("[SQLite] 找到助手 %s 的历史记录，共 %d 条消息", aid, len(messages))
	} else if errors.Is(err, sql.ErrNoRows) {
		log.Printf("[SQLite] 助手 %s 无历史记录，将创建新记录", aid)
		messages = []model.Message{}
	} else {
		log.Printf("[SQLite] 查询历史记录失败: %v", err)
		return fmt.Errorf("查询历史失败: %w", err)
	}

	// 2. 追加新消息并序列化完整历史
	messages = append(messages, msg)
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		log.Printf("[SQLite] 序列化消息失败: %v", err)
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 3. 更新或插入完整历史
	if history != nil {
		// 更新现有记录
		_, err = r.db.ExecContext(ctx,
			"UPDATE histories SET messages = ? WHERE assistant_id = ?",
			messagesJSON, aid)
	} else {
		// 插入新记录
		_, err = r.db.ExecContext(ctx,
			"INSERT INTO histories (assistant_id, messages) VALUES (?, ?)",
			aid, messagesJSON)
	}

	if err != nil {
		log.Printf("[SQLite] 保存历史记录失败: %v", err)
		return fmt.Errorf("保存历史失败: %w", err)
	}

	log.Printf("[SQLite] 成功保存助手 %s 的新消息", aid)
	return nil
}

// UpdateAssistantTimestamp 更新助手时间戳
func (r *HistorySQLiteRepo) UpdateAssistantTimestamp(ctx context.Context, aid, timestamp string) error {
	log.Printf("[SQLite] 更新助手 %s 的时间戳为: %s", aid, timestamp)

	res, err := r.db.ExecContext(ctx,
		"UPDATE assistants SET time_stamp = ? WHERE id = ?",
		timestamp, aid,
	)

	if err != nil {
		log.Printf("[SQLite] 更新时间戳失败: %v", err)
		return fmt.Errorf("更新时间戳失败: %w", err)
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		log.Printf("[SQLite] 更新时间戳失败: 未找到助手 %s", aid)
		return errors.New("助手不存在")
	}

	log.Printf("[SQLite] 成功更新助手 %s 的时间戳", aid)
	return nil
}
