package repository

import (
	"Voice_Assistant/internal/model"
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
)

// HistoryRepo 历史记录数据访问接口
type HistoryRepo interface {
	SelectByAssistantID(ctx context.Context, assistantID string) (*model.History, error)
	DeleteByAssistantID(ctx context.Context, assistantID string) error
	SaveByAssistantID(ctx context.Context, assistantID string, message model.Message) error
	UpdateAssistantTimestamp(ctx context.Context, assistantID string, timestamp string) error
}

// historyRepoImpl 历史记录数据访问实现
type historyRepoImpl struct {
	filePath       string       // 历史记录文件路径
	assistantsPath string       // 助手文件路径（用于更新时间戳）
	mu             sync.RWMutex // 线程安全锁
}

// NewHistoryRepo 创建历史记录仓库实例
func NewHistoryRepo(filePath string, assistantsPath string) HistoryRepo {
	return &historyRepoImpl{
		filePath:       filePath,
		assistantsPath: assistantsPath,
	}
}

// SelectByAssistantID 按助手ID查询历史
func (r *historyRepoImpl) SelectByAssistantID(ctx context.Context, assistantID string) (*model.History, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("历史记录不存在")
		}
		return nil, err
	}

	var histories []model.History
	if err := json.Unmarshal(data, &histories); err != nil {
		return nil, err
	}

	// 查找对应历史
	for _, h := range histories {
		if h.AssistantID == assistantID {
			return &h, nil
		}
	}

	return nil, errors.New("未找到该助手的历史记录")
}

// DeleteByAssistantID 按助手ID删除历史
func (r *historyRepoImpl) DeleteByAssistantID(ctx context.Context, assistantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("历史记录文件不存在")
		}
		return err
	}

	var histories []model.History
	if err := json.Unmarshal(data, &histories); err != nil {
		return err
	}

	// 过滤历史记录
	newHistories := make([]model.History, 0, len(histories))
	found := false
	for _, h := range histories {
		if h.AssistantID == assistantID {
			found = true
			continue
		}
		newHistories = append(newHistories, h)
	}

	if !found {
		return errors.New("未找到该助手的历史记录")
	}

	// 保存更新后的历史
	data, err = json.MarshalIndent(newHistories, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}

// SaveByAssistantID 保存消息到历史记录
func (r *historyRepoImpl) SaveByAssistantID(ctx context.Context, assistantID string, message model.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 读取现有历史
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte("[]") // 初始化空列表
		} else {
			return err
		}
	}

	var histories []model.History
	if err := json.Unmarshal(data, &histories); err != nil {
		return err
	}

	// 查找并追加消息
	foundIndex := -1
	for i, h := range histories {
		if h.AssistantID == assistantID {
			foundIndex = i
			break
		}
	}

	if foundIndex != -1 {
		// 追加到已有历史
		histories[foundIndex].Messages = append(histories[foundIndex].Messages, message)
	} else {
		// 新建历史记录
		histories = append(histories, model.History{
			AssistantID: assistantID,
			Messages:    []model.Message{message},
		})
	}

	// 保存更新后的历史
	data, err = json.MarshalIndent(histories, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}

// UpdateAssistantTimestamp 更新助手时间戳
func (r *historyRepoImpl) UpdateAssistantTimestamp(ctx context.Context, assistantID string, timestamp string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 读取助手数据
	data, err := os.ReadFile(r.assistantsPath)
	if err != nil {
		return err
	}

	var assistants []model.Assistant
	if err := json.Unmarshal(data, &assistants); err != nil {
		return err
	}

	// 查找并更新时间戳
	for i := range assistants {
		if assistants[i].ID == assistantID {
			assistants[i].TimeStamp = timestamp
			break
		}
	}

	// 保存更新后的助手数据
	data, err = json.MarshalIndent(assistants, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.assistantsPath, data, 0644)
}
