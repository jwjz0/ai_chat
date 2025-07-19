package repository

import (
	"Voice_Assistant/internal/model"
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
)

// AssistantRepo 定义数据访问接口（仅数据读写操作）
type AssistantRepo interface {
	// 读取所有数据
	SelectAll(ctx context.Context) ([]model.Assistant, error)
	// 根据ID删除（仅执行删除操作，不处理业务规则）
	DeleteByID(ctx context.Context, id string) error
	// 保存数据（仅写入，不处理业务校验）
	Save(ctx context.Context, assistant *model.Assistant) (*model.Assistant, error)
	// 更新数据（仅执行更新，不处理业务逻辑）
	UpdateByID(ctx context.Context, id string, assistant *model.Assistant) (*model.Assistant, error)
}

// assistantRepoImpl 实现数据访问逻辑（与文件交互）
type assistantRepoImpl struct {
	filePath      string       // 助手数据文件路径
	historiesPath string       // 历史记录文件路径（用于级联删除）
	mu            sync.RWMutex // 保证文件操作线程安全
}

func NewAssistantRepo(filePath string, historiesPath string) AssistantRepo {
	return &assistantRepoImpl{
		filePath:      filePath,
		historiesPath: historiesPath,
	}
}

// SelectAll 读取所有助手数据（仅文件读取，无业务逻辑）
func (r *assistantRepoImpl) SelectAll(ctx context.Context) ([]model.Assistant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.Assistant{}, nil // 文件不存在视为空数据
		}
		return nil, err // 其他错误返回
	}

	var assistants []model.Assistant
	if err := json.Unmarshal(data, &assistants); err != nil {
		return nil, err
	}
	return assistants, nil
}

// DeleteByID 按ID删除助手及关联历史（仅执行删除操作，不校验业务规则）
func (r *assistantRepoImpl) DeleteByID(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. 删除助手数据
	assistants, err := r.loadAssistants()
	if err != nil {
		return err
	}
	// 过滤掉要删除的ID
	newAssistants := make([]model.Assistant, 0, len(assistants))
	found := false
	for _, a := range assistants {
		if a.ID == id {
			found = true
			continue
		}
		newAssistants = append(newAssistants, a)
	}
	if !found {
		return errors.New("assistant not found") // 仅返回数据层错误
	}
	// 写入更新后的助手数据
	if err := r.saveAssistants(newAssistants); err != nil {
		return err
	}

	// 2. 级联删除关联的历史记录（数据层操作，无业务逻辑）
	histories, err := r.loadHistories()
	if err != nil {
		return err
	}
	// 过滤掉关联的历史
	newHistories := make([]model.History, 0, len(histories))
	for _, h := range histories {
		if h.AssistantID != id {
			newHistories = append(newHistories, h)
		}
	}
	// 写入更新后的历史数据
	if err := r.saveHistories(newHistories); err != nil {
		return err
	}

	return nil
}

// Save 保存助手数据（仅写入文件，不处理业务校验）
func (r *assistantRepoImpl) Save(ctx context.Context, assistant *model.Assistant) (*model.Assistant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	assistants, err := r.loadAssistants()
	if err != nil {
		return nil, err
	}
	// 直接添加（ID和时间由Service层处理）
	assistants = append(assistants, *assistant)
	if err := r.saveAssistants(assistants); err != nil {
		return nil, err
	}
	return assistant, nil
}

// UpdateByID 更新助手数据（仅执行更新，不处理业务规则）
func (r *assistantRepoImpl) UpdateByID(ctx context.Context, id string, assistant *model.Assistant) (*model.Assistant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	assistants, err := r.loadAssistants()
	if err != nil {
		return nil, err
	}

	// 查找并更新
	for i, a := range assistants {
		if a.ID == id {
			// 直接替换（更新内容由Service层处理）
			assistants[i] = *assistant
			if err := r.saveAssistants(assistants); err != nil {
				return nil, err
			}
			return assistant, nil
		}
	}
	return nil, errors.New("assistant not found")
}

// 以下为私有工具方法（仅数据层内部使用）
func (r *assistantRepoImpl) loadAssistants() ([]model.Assistant, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.Assistant{}, nil
		}
		return nil, err
	}
	var assistants []model.Assistant
	if err := json.Unmarshal(data, &assistants); err != nil {
		return nil, err
	}
	return assistants, nil
}

func (r *assistantRepoImpl) saveAssistants(assistants []model.Assistant) error {
	data, err := json.MarshalIndent(assistants, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}

func (r *assistantRepoImpl) loadHistories() ([]model.History, error) {
	data, err := os.ReadFile(r.historiesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.History{}, nil
		}
		return nil, err
	}
	var histories []model.History
	if err := json.Unmarshal(data, &histories); err != nil {
		return nil, err
	}
	return histories, nil
}

func (r *assistantRepoImpl) saveHistories(histories []model.History) error {
	data, err := json.MarshalIndent(histories, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.historiesPath, data, 0644)
}
