package service

import (
	"Voice_Assistant/internal/model"
	"Voice_Assistant/internal/repository"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type HistoryService interface {
	SelectByAssistantID(ctx context.Context, assistantID string) (*model.History, error)
	ResetByAssistantID(ctx context.Context, assistantID string) error
	SaveByAssistantID(ctx context.Context, assistantID string, message model.Message) error
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

// 按助手ID查询历史
func (s *historyServiceImpl) SelectByAssistantID(ctx context.Context, assistantID string) (*model.History, error) {
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询助手失败: %w", err)
	}
	for _, a := range assistants {
		if a.ID == assistantID {
			history, err := s.historyRepo.SelectByAssistantID(ctx, assistantID)
			if err != nil {
				return nil, fmt.Errorf("查询历史失败: %w", err)
			}
			return history, nil
		}
	}
	return nil, errors.New("助手不存在")
}

// 重置对话
func (s *historyServiceImpl) ResetByAssistantID(ctx context.Context, assistantID string) error {
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return fmt.Errorf("查询助手失败: %w", err)
	}
	for _, a := range assistants {
		if a.ID == assistantID {
			if err := s.historyRepo.DeleteByAssistantID(ctx, assistantID); err != nil {
				return fmt.Errorf("删除历史失败: %w", err)
			}
			// 添加重置消息
			msg := model.Message{
				Input:     model.Input{Prompt: a.Prompt},
				Output:    model.Output{Content: "对话已重置"},
				GmtCreate: time.Now().Format("2006-01-02 15:04:05"),
			}
			return s.historyRepo.SaveByAssistantID(ctx, assistantID, msg)
		}
	}
	return errors.New("助手不存在")
}

// 保存历史
func (s *historyServiceImpl) SaveByAssistantID(ctx context.Context, assistantID string, message model.Message) error {
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return fmt.Errorf("查询助手失败: %w", err)
	}
	for _, a := range assistants {
		if a.ID == assistantID {
			if err := s.historyRepo.SaveByAssistantID(ctx, assistantID, message); err != nil {
				return fmt.Errorf("保存失败: %w", err)
			}
			return s.historyRepo.UpdateAssistantTimestamp(ctx, assistantID, time.Now().Format("2006-01-02 15:04:05"))
		}
	}
	return errors.New("助手不存在")
}

// 流式处理（核心）
func (s *historyServiceImpl) StreamProcessMessage(ctx context.Context, assistantID string, input model.Input) (<-chan string, <-chan error, model.Usage, error) {
	contentChan := make(chan string)
	errChan := make(chan error, 1)
	var fullContent strings.Builder
	usage := model.Usage{}
	var wg sync.WaitGroup
	wg.Add(1)

	// 1. 获取助手信息
	assistant, err := s.getAssistant(ctx, assistantID)
	if err != nil {
		errChan <- err
		close(contentChan)
		close(errChan)
		return nil, nil, usage, err
	}

	// 2. 构建消息列表
	messages := []Message{
		{Role: "system", Content: "你是一个智能助手，会根据用户输入来挑选,如果是一些实时性的问答或者你只要通过调用提供的tools可以提升对话质量的就一定要调用。" + assistant.Prompt}, // 使用优化后的系统提示
	}
	// 追加历史消息
	history, err := s.historyRepo.SelectByAssistantID(ctx, assistantID)
	if err == nil && history != nil {
		for _, msg := range history.Messages {
			if msg.Input.Send != "" {
				messages = append(messages, Message{Role: "user", Content: msg.Input.Send})
			}
			if msg.Output.Content != "" {
				messages = append(messages, Message{Role: "assistant", Content: msg.Output.Content})
			}
		}
	}
	// 追加当前输入
	messages = append(messages, Message{Role: "user", Content: input.Send})

	// 3. 调用LLM服务
	llmChan, llmErrChan := s.llmService.StreamGenerateWithSearch(ctx, messages)

	// 4. 处理流式内容
	go func() {
		defer wg.Done()
		for chunk := range llmChan {
			fullContent.WriteString(chunk)
			contentChan <- chunk
		}
	}()

	// 5. 处理错误
	go func() {
		if err := <-llmErrChan; err != nil {
			errChan <- err
		}
	}()

	// 6. 确保保存历史
	go func() {
		wg.Wait()
		message := model.Message{
			Input:     input,
			Output:    model.Output{Content: fullContent.String()},
			Usage:     usage,
			GmtCreate: time.Now().Format("2006-01-02 15:04:05"),
		}
		if err := s.SaveByAssistantID(ctx, assistantID, message); err != nil {
			log.Printf("保存历史警告: %v", err)
		} else {
			log.Printf("历史保存成功，长度: %d", fullContent.Len())
		}
		close(contentChan)
		close(errChan)
	}()

	return contentChan, errChan, usage, nil
}

// 辅助：获取助手
func (s *historyServiceImpl) getAssistant(ctx context.Context, assistantID string) (*model.Assistant, error) {
	assistants, err := s.assistantRepo.SelectAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询助手失败: %w", err)
	}
	for _, a := range assistants {
		if a.ID == assistantID {
			return &a, nil
		}
	}
	return nil, errors.New("助手不存在")
}
