package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"Voice_Assistant/internal/model"
)

type llmServiceImpl struct {
	apiKey    string
	baseURL   string
	modelName string
	maxTokens int
	timeout   time.Duration
	client    *http.Client
}

func NewLLMService(apiKey, baseURL, modelName string, maxTokens int, timeoutSec int) LLMService {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: false,
	}

	return &llmServiceImpl{
		apiKey:    apiKey,
		baseURL:   baseURL,
		modelName: modelName,
		maxTokens: maxTokens,
		timeout:   time.Duration(timeoutSec) * time.Second,
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
			Transport: &http.Transport{
				DialContext:           dialer.DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

type chatRequest struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
	Stream    bool      `json:"stream,omitempty"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (s *llmServiceImpl) GenerateReply(ctx context.Context, prompt string, input string) (model.Output, model.Usage, error) {
	messages := []message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: input},
	}

	reqBody := chatRequest{
		Model:     s.modelName,
		Messages:  messages,
		MaxTokens: s.maxTokens,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return model.Output{}, model.Usage{}, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return model.Output{}, model.Usage{}, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return model.Output{}, model.Usage{}, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return model.Output{}, model.Usage{}, fmt.Errorf("API返回非成功状态: %d, 响应体: %s", resp.StatusCode, string(respBody))
	}

	var response chatResponse
	if err := json.NewDecoder(bytes.NewBuffer(respBody)).Decode(&response); err != nil {
		return model.Output{}, model.Usage{}, fmt.Errorf("解析响应失败: %w, 响应体: %s", err, string(respBody))
	}

	if len(response.Choices) == 0 {
		return model.Output{}, model.Usage{}, fmt.Errorf("未获取到生成结果, 响应体: %s", string(respBody))
	}

	output := model.Output{
		FinishReason: response.Choices[0].FinishReason,
		Content:      response.Choices[0].Message.Content,
	}

	usage := model.Usage{
		InputTokens:  response.Usage.PromptTokens,
		OutputTokens: response.Usage.CompletionTokens,
		TotalTokens:  response.Usage.TotalTokens,
	}

	return output, usage, nil
}

func (s *llmServiceImpl) StreamGenerate(ctx context.Context, messages []message) (<-chan string, <-chan error) {
	contentChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		reqBody := chatRequest{
			Model:     s.modelName,
			Messages:  messages,
			MaxTokens: s.maxTokens,
			Stream:    true,
		}

		reqBytes, err := json.Marshal(reqBody)
		if err != nil {
			errChan <- fmt.Errorf("序列化请求失败: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewBuffer(reqBytes))
		if err != nil {
			errChan <- fmt.Errorf("创建请求失败: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.apiKey)

		resp, err := s.client.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("发送请求失败: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("LLM返回错误: 状态码=%d, 内容=%s", resp.StatusCode, string(body))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		for scanner.Scan() {
			// 检查是否已中止
			select {
			case <-ctx.Done():
				resp.Body.Close() // 关闭响应体，停止接收新数据
				errChan <- ctx.Err()
				return
			default:
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			if line == "data: [DONE]" {
				break
			}

			if strings.HasPrefix(line, "data: ") {
				dataStr := strings.TrimPrefix(line, "data: ")
				var streamResp struct {
					Choices []struct {
						Delta struct{ Content string } `json:"delta"`
					} `json:"choices"`
					Error *struct{ Message string } `json:"error"`
				}

				if err := json.Unmarshal([]byte(dataStr), &streamResp); err != nil {
					errChan <- fmt.Errorf("解析响应失败: %w", err)
					return
				}

				if streamResp.Error != nil {
					errChan <- fmt.Errorf("大模型错误: %s", streamResp.Error.Message)
					return
				}

				if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
					contentChan <- streamResp.Choices[0].Delta.Content
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("读取流失败: %w", err)
		}
	}()

	return contentChan, errChan
}
