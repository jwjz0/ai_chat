package service

import (
	"Voice_Assistant/internal/model"
	"Voice_Assistant/internal/repository"
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

type HistoryService interface {
	SelectByAssistantID(ctx context.Context, assistantID string) (*model.History, error)
	ResetByAssistantID(ctx context.Context, assistantID string) error
	SaveByAssistantID(ctx context.Context, assistantID string, message model.Message) error
	ProcessMessage(ctx context.Context, assistantID string, input model.Input) (model.Message, error)
	StreamProcessMessage(ctx context.Context, assistantID string, input model.Input) (<-chan string, <-chan error, model.Usage, error)
}

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

func (s *historyServiceImpl) SelectByAssistantID(ctx context.Context, assistantID string) (*model.History, error) {
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

	history, err := s.historyRepo.SelectByAssistantID(ctx, assistantID)
	if err != nil {
		return nil, errors.New("查询历史记录失败: " + err.Error())
	}
	return history, nil
}

func (s *historyServiceImpl) ResetByAssistantID(ctx context.Context, assistantID string) error {
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

	if err := s.historyRepo.DeleteByAssistantID(ctx, assistantID); err != nil {
		if err.Error() == "未找到该助手的历史记录" {
		} else {
			return errors.New("删除历史对话失败: " + err.Error())
		}
	}

	defaultMessage := model.Message{
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

	if err := s.historyRepo.SaveByAssistantID(ctx, assistantID, defaultMessage); err != nil {
		return errors.New("添加默认消息失败: " + err.Error())
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	if err := s.historyRepo.UpdateAssistantTimestamp(ctx, assistantID, now); err != nil {
		return errors.New("更新助手时间戳失败: " + err.Error())
	}

	return nil
}

func (s *historyServiceImpl) SaveByAssistantID(ctx context.Context, assistantID string, message model.Message) error {
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

	now := time.Now().Format("2006-01-02 15:04:05")

	if err := s.historyRepo.SaveByAssistantID(ctx, assistantID, message); err != nil {
		return errors.New("保存历史记录失败: " + err.Error())
	}

	if err := s.historyRepo.UpdateAssistantTimestamp(ctx, assistantID, now); err != nil {
		return errors.New("更新助手时间戳失败: " + err.Error())
	}

	return nil
}

func (s *historyServiceImpl) ProcessMessage(ctx context.Context, assistantID string, input model.Input) (model.Message, error) {
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
	log.Printf("找到的助手信息: %v", assistant)

	if input.Prompt == "" {
		input.Prompt = assistant.Prompt
	}

	output, usage, err := s.llmService.GenerateReply(ctx, input.Prompt, input.Send)
	if err != nil {
		return model.Message{}, errors.New("生成回复失败: " + err.Error())
	}

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

// 修改 historyServiceImpl 的 StreamProcessMessage 方法
func (s *historyServiceImpl) StreamProcessMessage(ctx context.Context, assistantID string, input model.Input) (<-chan string, <-chan error, model.Usage, error) {
	contentChan := make(chan string)
	errChan := make(chan error, 1)
	fullContent := ""
	usage := model.Usage{InputTokens: len(input.Send) / 4}

	// 1. 校验助手存在性
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

	// 2. 获取历史对话记录（关键：多轮对话核心）
	history, err := s.historyRepo.SelectByAssistantID(ctx, assistantID)
	if err != nil {
		close(contentChan)
		close(errChan)
		return nil, nil, usage, fmt.Errorf("查询历史记录失败: %w", err)
	}

	// 3. 构建完整消息链（包含历史对话）
	messages := []message{
		{Role: "system", Content: input.Prompt}, // 系统提示（初始设定）
	}

	// 添加历史消息（关键：多轮对话上下文）
	for _, msg := range history.Messages {
		// 跳过空消息
		if msg.Input.Send != "" {
			messages = append(messages, message{
				Role:    "user",
				Content: msg.Input.Send, // 历史用户输入
			})
		}
		if msg.Output.Content != "" {
			messages = append(messages, message{
				Role:    "assistant",
				Content: msg.Output.Content, // 历史助手回复
			})
		}
	}

	// 添加当前用户输入
	messages = append(messages, message{
		Role:    "user",
		Content: input.Send,
	})

	// 4. 调用LLM流式接口（传入完整消息链）
	llmContentChan, llmErrChan := s.llmService.StreamGenerate(ctx, messages)

	// 5. 转发内容并收集完整回复
	go func() {
		defer close(contentChan)
		defer close(errChan)

		contentClosed := false
		errClosed := false

		for {
			if contentClosed && errClosed {
				// 流式结束后保存对话
				usage.OutputTokens = len(fullContent) / 4
				usage.TotalTokens = usage.InputTokens + usage.OutputTokens
				message := model.Message{
					Input:     input,
					Output:    model.Output{Content: fullContent, FinishReason: "stop"},
					Usage:     usage,
					GmtCreate: time.Now().Format("2006-01-02 15:04:05"),
				}
				if err := s.SaveByAssistantID(ctx, assistantID, message); err != nil {
					log.Printf("保存对话失败: %v", err)
				} else {
					log.Printf("对话保存成功")
				}
				return
			}

			select {
			case content, ok := <-llmContentChan:
				if !ok {
					contentClosed = true
					continue
				}
				fullContent += content
				contentChan <- content // 转发片段
			case err, ok := <-llmErrChan:
				if !ok {
					errClosed = true
					continue
				}
				errChan <- err // 转发错误
				return
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return contentChan, errChan, usage, nil
}
