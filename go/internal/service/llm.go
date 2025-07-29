package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// 工具定义
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// 工具调用响应结构
type ToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// 消息结构
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// 博查搜索相关结构
type BochaSearchRequest struct {
	Query     string `json:"query"`
	Freshness string `json:"freshness,omitempty"`
	Summary   bool   `json:"summary"`
	Count     int    `json:"count"`
}

type BochaSearchResponse struct {
	Code  int    `json:"code"`
	LogID string `json:"log_id"`
	Msg   string `json:"msg"`
	Data  struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`
				Url     string `json:"url"`
				Snippet string `json:"snippet"`
				Time    string `json:"datePublished"`
			} `json:"value"`
		} `json:"webPages"`
	} `json:"data"`
}

// LLM服务接口
type LLMService interface {
	GenerateReply(ctx context.Context, prompt string, input string) (string, error)
	StreamGenerate(ctx context.Context, messages []Message, tools []Tool) (<-chan string, <-chan error)
	StreamGenerateWithSearch(ctx context.Context, messages []Message) (<-chan string, <-chan error)
}

// LLM服务实现
type llmServiceImpl struct {
	apiKey      string
	baseURL     string
	modelName   string
	maxTokens   int
	timeout     time.Duration
	client      *http.Client
	bochaAPIKey string
	tools       []Tool
}

// 初始化函数
func NewLLMService(apiKey, baseURL, modelName string, maxTokens int, timeoutSec int, bochaAPIKey string) LLMService {
	tools := []Tool{
		{
			Type: "function",
			Function: Function{
				Name:        "bocha_search",
				Description: "获取实时信息、动态数据或最新资讯时使用",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "搜索关键词（完整自然语言句子，含时间、地点）",
						},
						"freshness": map[string]interface{}{
							"type":        "string",
							"description": "时间范围（如'oneWeek'）",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "结果数量(最少5个)",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}

	return &llmServiceImpl{
		apiKey:    apiKey,
		baseURL:   baseURL,
		modelName: modelName,
		maxTokens: maxTokens,
		timeout:   time.Duration(timeoutSec) * time.Second,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		bochaAPIKey: bochaAPIKey,
		tools:       tools,
	}
}

// 非流式生成
func (s *llmServiceImpl) GenerateReply(ctx context.Context, prompt string, input string) (string, error) {
	reqBody := map[string]interface{}{
		"model": s.modelName,
		"messages": []Message{
			{Role: "system", Content: prompt},
			{Role: "user", Content: input},
		},
		"max_tokens": s.maxTokens,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API错误: %d, 内容: %s", resp.StatusCode, string(respBody))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", errors.New("无生成结果")
	}

	return response.Choices[0].Message.Content, nil
}

// 流式生成
func (s *llmServiceImpl) StreamGenerate(ctx context.Context, messages []Message, tools []Tool) (<-chan string, <-chan error) {
	contentChan, errChan := make(chan string), make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		reqBody := map[string]interface{}{
			"model":      s.modelName,
			"messages":   messages,
			"max_tokens": s.maxTokens,
			"stream":     true,
			"tools":      tools,
		}
		reqBytes, err := json.Marshal(reqBody)
		if err != nil {
			errChan <- fmt.Errorf("序列化失败: %w", err)
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
			errChan <- fmt.Errorf("API错误: %d, 内容: %s", resp.StatusCode, string(body))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				resp.Body.Close()
				errChan <- ctx.Err()
				return
			default:
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !strings.HasPrefix(line, "data: [DONE]") {
					contentChan <- strings.TrimPrefix(line, "data: ")
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("读取流失败: %w", err)
		}
	}()

	return contentChan, errChan
}

func (s *llmServiceImpl) StreamGenerateWithSearch(ctx context.Context, messages []Message) (<-chan string, <-chan error) {
	contentChan, errChan := make(chan string), make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		// 第一次调用明确标记（移除冗余换行符）
		log.Println("======================================")
		log.Println("===== 开始第一次LLM调用（判断是否需要工具） =====")
		log.Println("======================================")

		streamChan, streamErrChan := s.StreamGenerate(ctx, messages, s.tools)
		toolCalls, assistantMsg, finalResponse := s.parseToolCalls(streamChan, streamErrChan)

		log.Println("======================================")
		log.Println("===== 第一次LLM调用结束 =====")
		log.Println("======================================")

		if len(toolCalls) == 0 {
			log.Println("无需二次调用，直接返回结果")
			contentChan <- finalResponse
			return
		}

		// 明确标记进入第二次调用（移除冗余换行符）
		log.Println("======================================")
		log.Println("===== 检测到工具调用，准备第二次LLM调用 =====")
		log.Println("======================================")

		messages = append(messages, assistantMsg)
		log.Printf("工具调用详情: %+v\n", toolCalls)

		toolResults := s.executeTools(ctx, toolCalls)
		messages = append(messages, toolResults...)

		// 开始第二次调用标记（移除冗余换行符）
		log.Println("======================================")
		log.Println("===== 开始第二次LLM调用（生成最终回答） =====")
		log.Println("======================================")

		finalChan, finalErrChan := s.StreamGenerate(ctx, messages, s.tools)
		s.forwardStream(finalChan, finalErrChan, contentChan)

		log.Println("======================================")
		log.Println("===== 第二次LLM调用结束 =====")
		log.Println("======================================")
	}()

	return contentChan, errChan
}

// 解析流式响应
func (s *llmServiceImpl) parseToolCalls(streamChan <-chan string, errChan <-chan error) ([]ToolCall, Message, string) {
	type partialTool struct {
		id        string
		name      string
		arguments strings.Builder
	}
	partials := make(map[int]*partialTool)
	var finalContent string
	var fullResponse strings.Builder
	var assistantMsg Message
	assistantMsg.Role = "assistant"

	for chunk := range streamChan {
		fullResponse.WriteString(chunk)

		var resp struct {
			Choices []struct {
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(chunk), &resp); err != nil {
			continue
		}

		for _, choice := range resp.Choices {
			finalContent += choice.Delta.Content
			assistantMsg.Content += choice.Delta.Content

			for _, tc := range choice.Delta.ToolCalls {
				pt, exists := partials[tc.Index]
				if !exists {
					pt = &partialTool{}
					partials[tc.Index] = pt
				}
				if tc.ID != "" {
					pt.id = tc.ID
				}
				if tc.Function.Name != "" {
					pt.name = tc.Function.Name
				}
				pt.arguments.WriteString(tc.Function.Arguments)
				assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, tc)
			}
		}
	}

	if err := <-errChan; err != nil {
		log.Printf("解析工具调用错误: %v", err)
	}

	// 打印第一次调用的完整结果（格式化后的JSON）
	log.Println("第一次LLM调用完整返回结果:")
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(fullResponse.String()), "", "  "); err != nil {
		log.Println(fullResponse.String()) // 格式化失败时直接打印原始内容
	} else {
		log.Println(prettyJSON.String())
	}

	var toolCalls []ToolCall
	for idx, pt := range partials {
		if pt.name == "" {
			continue
		}
		toolCalls = append(toolCalls, ToolCall{
			Index: idx,
			ID:    pt.id,
			Type:  "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      pt.name,
				Arguments: pt.arguments.String(),
			},
		})
	}

	return toolCalls, assistantMsg, finalContent
}

// 执行工具调用
func (s *llmServiceImpl) executeTools(ctx context.Context, calls []ToolCall) []Message {
	var results []Message
	for _, call := range calls {
		if call.Function.Name != "bocha_search" {
			results = append(results, Message{
				Role:       "tool",
				Content:    fmt.Sprintf("不支持的工具: %s", call.Function.Name),
				ToolCallID: call.ID,
			})
			continue
		}

		var params map[string]interface{}
		argsStr := call.Function.Arguments
		if argsStr != "" {
			if err := json.Unmarshal([]byte(argsStr), &params); err != nil {
				results = append(results, Message{
					Role:       "tool",
					Content:    fmt.Sprintf("参数解析错误: %v，原始参数: %s", err, argsStr),
					ToolCallID: call.ID,
				})
				continue
			}
		} else {
			params = make(map[string]interface{})
		}

		query, ok := params["query"].(string)
		if !ok || strings.TrimSpace(query) == "" {
			results = append(results, Message{
				Role:       "tool",
				Content:    "错误：搜索关键词不能为空",
				ToolCallID: call.ID,
			})
			continue
		}

		searchReq := BochaSearchRequest{
			Query:   query,
			Count:   5,
			Summary: true,
		}
		if freshness, ok := params["freshness"].(string); ok {
			searchReq.Freshness = freshness
		}
		if count, ok := params["count"].(float64); ok {
			searchReq.Count = int(count)
		}

		resp, err := s.callBochaAPI(ctx, searchReq)
		if err != nil {
			results = append(results, Message{
				Role:       "tool",
				Content:    fmt.Sprintf("搜索失败: %v", err),
				ToolCallID: call.ID,
			})
			continue
		}

		results = append(results, Message{
			Role:       "tool",
			Content:    s.formatSearchResult(resp),
			ToolCallID: call.ID,
		})
	}
	return results
}

// 调用博查API
func (s *llmServiceImpl) callBochaAPI(ctx context.Context, req BochaSearchRequest) (*BochaSearchResponse, error) {
	apiURL := "https://api.bochaai.com/v1/web-search"
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.Header.Set("Authorization", "Bearer "+s.bochaAPIKey)

	resp, err := s.client.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("请求API失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP错误: %d, 内容: %s", resp.StatusCode, string(respBody))
	}

	var searchResp BochaSearchResponse
	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if searchResp.Code != 200 {
		return nil, fmt.Errorf("业务错误: %s", searchResp.Msg)
	}

	return &searchResp, nil
}

// 格式化搜索结果（每条结果整行打印）
func (s *llmServiceImpl) formatSearchResult(resp *BochaSearchResponse) string {
	var result strings.Builder
	result.WriteString("以下是搜索结果：\n\n")

	// 打印搜索结果（每条结果一行）
	log.Println("搜索结果：")
	for i, item := range resp.Data.WebPages.Value {
		// 单条结果格式：序号. 名称 [发布时间] 摘要 - 链接
		itemStr := fmt.Sprintf("%d. %s [%s] %s - %s",
			i+1, item.Name, item.Time, item.Snippet, item.Url)
		log.Println(itemStr)

		// 同时构建返回给LLM的结果字符串
		result.WriteString(fmt.Sprintf("%d. %s\n发布时间: %s\n摘要: %s\n链接: %s\n\n",
			i+1, item.Name, item.Time, item.Snippet, item.Url))
	}

	return result.String()
}

// 转发流式结果并打印第二次调用返回值
func (s *llmServiceImpl) forwardStream(finalChan <-chan string, finalErrChan <-chan error, contentChan chan<- string) {
	var finalAnswer strings.Builder
	var fullResponse strings.Builder // 收集完整的流式响应

	for chunk := range finalChan {
		fullResponse.WriteString(chunk) // 收集原始响应

		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(chunk), &streamResp); err == nil {
			for _, choice := range streamResp.Choices {
				if choice.Delta.Content != "" {
					contentChan <- choice.Delta.Content
					finalAnswer.WriteString(choice.Delta.Content)
				}
			}
		}
	}

	// 打印第二次调用的完整结果
	log.Println("第二次LLM调用完整返回结果:")
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(fullResponse.String()), "", "  "); err != nil {
		log.Println(fullResponse.String()) // 格式化失败时直接打印
	} else {
		log.Println(prettyJSON.String())
	}

	log.Println("第二次LLM调用生成的最终回答:")
	log.Println(finalAnswer.String())

	if err := <-finalErrChan; err != nil {
		log.Printf("最终生成错误: %v", err)
		contentChan <- "生成回答失败"
	}
}
