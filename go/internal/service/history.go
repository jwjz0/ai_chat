// service/history_service.go - 完整修复版本

package service

import (
	"Voice_Assistant/internal/model"
	"Voice_Assistant/internal/repository"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// HistoryService 历史服务接口
type HistoryService interface {
	SelectByAssistantID(ctx context.Context, assistantID string) (*model.History, error)
	ResetByAssistantID(ctx context.Context, assistantID string) error // 重置对话
	SaveByAssistantID(ctx context.Context, assistantID string, message model.Message) error
	ProcessMessage(ctx context.Context, assistantID string, input model.Input) (model.Message, error)
	StreamProcessMessage(ctx context.Context, assistantID string, input model.Input) (<-chan string, <-chan error, model.Usage, error)
}

type LLMService interface {
	GenerateReply(ctx context.Context, prompt, send string) (model.Output, model.Usage, error)
	StreamGenerate(ctx context.Context, messages []message) (<-chan string, <-chan error)
}

// historyServiceImpl 实现历史服务
type historyServiceImpl struct {
	historyRepo   repository.HistoryRepo
	assistantRepo repository.AssistantRepo
	llmService    LLMService
}

func NewHistoryService(historyRepo repository.HistoryRepo, assistantRepo repository.AssistantRepo, llmService LLMService) HistoryService {
	return &historyServiceImpl{
		historyRepo:   historyRepo,
		assistantRepo: assistantRepo,
		llmService:    llmService,
	}
}

// SelectByAssistantID 按助手ID查询历史
func (s *historyServiceImpl) SelectByAssistantID(ctx context.Context, assistantID string) (*model.History, error) {
	// 验证助手存在
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return nil, errors.New("查询助手失败: " + err.Error())
	}
	exists := false
	for _, a := range assistants {
		if a.ID == assistantID {
			exists = true
			break
		}
	}
	if !exists {
		return nil, errors.New("助手不存在")
	}

	// 查询历史记录
	history, err := s.historyRepo.SelectByAssistantID(ctx, assistantID)
	if err != nil {
		return nil, errors.New("查询历史记录失败: " + err.Error())
	}
	return history, nil
}

// ResetByAssistantID 重置对话
func (s *historyServiceImpl) ResetByAssistantID(ctx context.Context, assistantID string) error {
	// 1. 验证助手存在
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return errors.New("查询助手失败: " + err.Error())
	}
	var targetAssistant model.Assistant
	exists := false
	for _, a := range assistants {
		if a.ID == assistantID {
			targetAssistant = a
			exists = true
			break
		}
	}
	if !exists {
		return errors.New("助手不存在")
	}

	// 2. 删除原历史记录
	if err := s.historyRepo.DeleteByAssistantID(ctx, assistantID); err != nil {
		return errors.New("删除历史记录时发生错误: " + err.Error())
	}

	// 3. 添加重置后的默认消息
	resetMessage := model.Message{
		Input: model.Input{
			Prompt: targetAssistant.Prompt,
			Send:   "",
		},
		Output: model.Output{
			FinishReason: "stop",
			Content:      "对话已重置，欢迎再次使用" + targetAssistant.Name + "！",
		},
		Usage: model.Usage{
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
		},
		GmtCreate: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 保存默认消息
	if err := s.historyRepo.SaveByAssistantID(ctx, assistantID, resetMessage); err != nil {
		return errors.New("添加默认消息失败: " + err.Error())
	}

	// 4. 更新时间戳
	now := time.Now().Format("2006-01-02 15:04:05")
	if err := s.historyRepo.UpdateAssistantTimestamp(ctx, assistantID, now); err != nil {
		return errors.New("更新助手时间戳失败: " + err.Error())
	}

	return nil
}

// SaveByAssistantID 保存历史记录
func (s *historyServiceImpl) SaveByAssistantID(ctx context.Context, assistantID string, message model.Message) error {
	// 使用独立上下文验证助手存在（避免主上下文取消影响）
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return errors.New("查询助手失败: " + err.Error())
	}
	exists := false
	for _, a := range assistants {
		if a.ID == assistantID {
			exists = true
			break
		}
	}
	if !exists {
		return errors.New("助手不存在")
	}

	// 保存历史记录
	if err := s.historyRepo.SaveByAssistantID(ctx, assistantID, message); err != nil {
		return errors.New("保存历史记录失败: " + err.Error())
	}

	// 更新时间戳
	now := time.Now().Format("2006-01-02 15:04:05")
	if err := s.historyRepo.UpdateAssistantTimestamp(ctx, assistantID, now); err != nil {
		return errors.New("更新助手时间戳失败: " + err.Error())
	}

	return nil
}

// ProcessMessage 处理消息（生成回复）
func (s *historyServiceImpl) ProcessMessage(ctx context.Context, assistantID string, input model.Input) (model.Message, error) {
	// 验证助手存在
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		log.Printf("查询助手失败: %v", err)
		return model.Message{}, errors.New("查询助手失败: " + err.Error())
	}

	var assistant model.Assistant
	exists := false
	for _, a := range assistants {
		if a.ID == assistantID {
			assistant = a
			exists = true
			break
		}
	}
	if !exists {
		return model.Message{}, errors.New("助手不存在")
	}

	// 补全prompt
	if input.Prompt == "" {
		input.Prompt = assistant.Prompt
	}

	// 调用大模型生成回复
	output, usage, err := s.llmService.GenerateReply(ctx, input.Prompt, input.Send)
	if err != nil {
		return model.Message{}, errors.New("生成回复失败: " + err.Error())
	}

	// 封装消息并保存
	message := model.Message{
		Input:     input,
		Output:    output,
		Usage:     usage,
		GmtCreate: time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := s.SaveByAssistantID(ctx, assistantID, message); err != nil {
		return model.Message{}, err
	}

	return message, nil
}

// StreamProcessMessage 流式处理消息
func (s *historyServiceImpl) StreamProcessMessage(ctx context.Context, assistantID string, input model.Input) (<-chan string, <-chan error, model.Usage, error) {
	contentChan := make(chan string)
	errChan := make(chan error, 1)
	fullContent := ""
	usage := model.Usage{InputTokens: len(input.Send) / 4} // 粗略估算
	var isAborted bool = false

	// 添加等待组用于同步消息保存
	var saveWg sync.WaitGroup
	saveWg.Add(1)

	// 验证助手存在（使用原始上下文）
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		close(contentChan)
		close(errChan)
		return nil, nil, usage, fmt.Errorf("查询助手失败: %w", err)
	}
	found := false
	for _, a := range assistants {
		if a.ID == assistantID {
			found = true
			break
		}
	}
	if !found {
		close(contentChan)
		close(errChan)
		return nil, nil, usage, errors.New("助手不存在")
	}

	// 查询历史记录（使用原始上下文）
	history, err := s.historyRepo.SelectByAssistantID(ctx, assistantID)
	if err != nil {
		close(contentChan)
		close(errChan)
		return nil, nil, usage, fmt.Errorf("查询历史记录失败: %w", err)
	}

	// 构建大模型输入消息
	messages := []message{
		{Role: "system", Content: input.Prompt},
	}

	// 追加历史消息
	for _, msg := range history.Messages {
		if msg.Input.Send != "" {
			messages = append(messages, message{
				Role:    "user",
				Content: msg.Input.Send,
			})
		}
		if msg.Output.Content != "" {
			messages = append(messages, message{
				Role:    "assistant",
				Content: msg.Output.Content,
			})
		}
	}

	// 追加当前输入
	messages = append(messages, message{
		Role:    "user",
		Content: input.Send,
	})

	// 调用流式生成
	llmContentChan, llmErrChan := s.llmService.StreamGenerate(ctx, messages)

	// 处理流式输出
	go func() {
		defer func() {
			// 使用独立的上下文执行保存操作
			saveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// 保存最终消息
			finishReason := "stop"
			content := fullContent
			if isAborted {
				finishReason = "abort"
				content += "（已中止）"
			}

			finalUsage := model.Usage{
				InputTokens:  usage.InputTokens,
				OutputTokens: len(fullContent) / 4, // 粗略估算
				TotalTokens:  usage.InputTokens + len(fullContent)/4,
			}

			message := model.Message{
				Input: input,
				Output: model.Output{
					FinishReason: finishReason,
					Content:      content,
				},
				Usage:     finalUsage,
				GmtCreate: time.Now().Format("2006-01-02 15:04:05"),
			}

			// 使用独立上下文保存消息
			if err := s.SaveByAssistantID(saveCtx, assistantID, message); err != nil {
				log.Printf("保存对话失败: %v", err)
				errChan <- err
			} else {
				log.Printf("对话保存成功 (是否中止: %v, 内容长度: %d)", isAborted, len(fullContent))
			}

			// 标记保存完成
			saveWg.Done()

			close(contentChan)
			close(errChan)
		}()

		// 接收流式输出
		for {
			select {
			case content, ok := <-llmContentChan:
				if !ok {
					return
				}
				fullContent += content
				contentChan <- content
			case err, ok := <-llmErrChan:
				if !ok {
					return
				}
				errChan <- err
				return
			case <-ctx.Done():
				// 正确标记为中止状态
				isAborted = true
				log.Printf("上下文取消，标记消息为中止状态，当前内容长度: %d", len(fullContent))
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return contentChan, errChan, usage, nil
}
