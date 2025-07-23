package service

import (
	"Voice_Assistant/internal/model"
	"Voice_Assistant/internal/repository"
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
)

// AssistantService 定义业务接口（包含业务逻辑）
type AssistantService interface {
	// 查询所有助手（可添加业务过滤逻辑）
	SelectAll(ctx context.Context) ([]model.Assistant, error)
	// 按ID删除（包含业务校验，如是否允许删除）
	DeleteByID(ctx context.Context, id string) error
	// 保存助手（包含数据校验、ID生成、时间设置等业务逻辑）
	Save(ctx context.Context, assistant *model.Assistant) (*model.Assistant, error)
	// 更新助手（包含业务校验，如更新权限、数据合法性）
	UpdateByID(ctx context.Context, id string, assistant *model.Assistant) (*model.Assistant, error)
}

// assistantServiceImpl 实现业务逻辑
type assistantServiceImpl struct {
	assistantRepo  repository.AssistantRepo // 依赖数据访问层
	historyService HistoryService
}

func NewAssistantService(assistantRepo repository.AssistantRepo, historyService HistoryService) AssistantService {
	return &assistantServiceImpl{
		assistantRepo:  assistantRepo,
		historyService: historyService, // 注入依赖
	}
}

// SelectAll 查询所有助手
func (s *assistantServiceImpl) SelectAll(ctx context.Context) ([]model.Assistant, error) {
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return nil, errors.New("failed to get assistants: " + err.Error()) // 包装业务错误
	}
	return assistants, nil
}

// DeleteByID 按ID删除（业务校验：如ID格式是否合法、是否存在）
func (s *assistantServiceImpl) DeleteByID(ctx context.Context, id string) error {
	// 业务校验1：ID格式是否合法（UUID格式）
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("invalid assistant ID format")
	}
	// 业务校验2：是否存在该助手（可查库确认，避免无效删除）
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return errors.New("failed to check assistant existence: " + err.Error())
	}
	exists := false
	for _, a := range assistants {
		if a.ID == id {
			exists = true
			break
		}
	}
	if !exists {
		return errors.New("assistant not found (business)")
	}
	// 执行删除（调用数据层）
	if err := s.assistantRepo.DeleteByID(ctx, id); err != nil {
		return errors.New("failed to delete assistant: " + err.Error())
	}
	return nil
}

// Save 保存助手（业务逻辑：数据校验、ID生成、时间设置）
func (s *assistantServiceImpl) Save(ctx context.Context, assistant *model.Assistant) (*model.Assistant, error) {
	// 业务校验1：必填字段检查
	if assistant == nil {
		return nil, errors.New("assistant cannot be nil")
	}
	if assistant.Name == "" {
		return nil, errors.New("assistant name is required")
	}
	if assistant.Prompt == "" {
		return nil, errors.New("assistant prompt is required")
	}

	// 业务逻辑：生成UUID（业务层负责ID生成，而非数据层）
	id := uuid.New().String()
	assistant.ID = id

	// 业务逻辑：设置时间（创建时间和修改时间）
	now := time.Now().Format("2006-01-02 15:04:05")
	assistant.GmtCreate = now
	assistant.GmtModified = now
	assistant.TimeStamp = now

	// 调用数据层保存
	saved, err := s.assistantRepo.Save(ctx, assistant)
	if err != nil {
		return nil, errors.New("failed to save assistant: " + err.Error())
	}

	// 添加默认欢迎消息
	defaultMessage := model.Message{
		Input: model.Input{
			Prompt: saved.Prompt,
			Send:   "",
		},
		Output: model.Output{
			FinishReason: "stop",
			Content:      "欢迎使用" + saved.Name + "！我已准备好为你提供帮助~",
		},
		Usage: model.Usage{
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
		},
		GmtCreate: saved.GmtCreate, // 与助手创建时间一致
	}

	if err := s.historyService.SaveByAssistantID(ctx, saved.ID, defaultMessage); err != nil {
		// 注意：默认消息添加失败不影响助手创建，仅记录警告日志
		log.Printf("警告：助手创建成功，但默认消息添加失败: %v", err)
	}

	return saved, nil
}

// UpdateByID 更新助手（业务逻辑：校验合法性、更新时间）
func (s *assistantServiceImpl) UpdateByID(ctx context.Context, id string, assistant *model.Assistant) (*model.Assistant, error) {
	// 业务校验1：ID格式合法
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("invalid assistant ID format")
	}
	// 业务校验2：更新的数据是否合法（如名称不为空）
	if assistant.Name == "" {
		return nil, errors.New("assistant name cannot be empty")
	}

	// 业务逻辑：查询原数据（确保存在）
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return nil, errors.New("failed to get original assistant: " + err.Error())
	}
	var original model.Assistant
	found := false
	for _, a := range assistants {
		if a.ID == id {
			original = a
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("assistant not found")
	}

	// 业务逻辑：更新字段（只允许更新指定字段，避免非法修改）
	updated := model.Assistant{
		ID:          id,                                       // ID不可改
		Name:        assistant.Name,                           // 允许更新名称
		Description: assistant.Description,                    //允许更新描述
		Prompt:      assistant.Prompt,                         // 允许更新Prompt
		GmtCreate:   original.GmtCreate,                       // 创建时间不可改
		GmtModified: time.Now().Format("2006-01-02 15:04:05"), // 更新修改时间
		TimeStamp:   time.Now().Format("2006-01-02 15:04:05"), // 同步时间戳
	}

	// 调用数据层执行更新
	result, err := s.assistantRepo.UpdateByID(ctx, id, &updated)
	if err != nil {
		return nil, errors.New("failed to update assistant: " + err.Error())
	}
	return result, nil
}
