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

func (s *historyServiceImpl) StreamProcessMessage(ctx context.Context, assistantID string, input model.Input) (<-chan string, <-chan error, model.Usage, error) {
	contentChan := make(chan string)
	errChan := make(chan error, 1)
	fullContent := ""
	usage := model.Usage{InputTokens: len(input.Send) / 4}
	var isAborted bool = false

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

	history, err := s.historyRepo.SelectByAssistantID(ctx, assistantID)
	if err != nil {
		close(contentChan)
		close(errChan)
		return nil, nil, usage, fmt.Errorf("查询历史记录失败: %w", err)
	}

	messages := []message{
		{Role: "system", Content: input.Prompt},
	}

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

	messages = append(messages, message{
		Role:    "user",
		Content: input.Send,
	})

	llmContentChan, llmErrChan := s.llmService.StreamGenerate(ctx, messages)

	go func() {
		defer func() {
			// 无论正常结束还是中止，都保存消息
			finishReason := "stop"
			content := fullContent
			if isAborted {
				finishReason = "abort"
				content += "（已中止）"
			}

			finalUsage := model.Usage{
				InputTokens:  usage.InputTokens,
				OutputTokens: len(fullContent) / 4,
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

			if err := s.SaveByAssistantID(ctx, assistantID, message); err != nil {
				log.Printf("保存对话失败: %v", err)
			} else {
				log.Printf("对话保存成功 (是否中止: %v)", isAborted)
			}

			close(contentChan)
			close(errChan)
		}()

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
				// 上下文取消，标记为中止
				isAborted = true
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return contentChan, errChan, usage, nil
}
