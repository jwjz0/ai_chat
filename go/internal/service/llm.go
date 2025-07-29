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

// 工具定义（通用搜索描述）
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

// 博查搜索相关结构（修正JSON标签，匹配API响应）
type BochaSearchRequest struct {
	Query     string `json:"query"`     // 搜索关键词
	Freshness string `json:"freshness"` // 新鲜度（如"oneDay"、"oneWeek"）
	Summary   bool   `json:"summary"`   // 是否返回摘要
	Count     int    `json:"count"`     // 返回结果数量
}

// 核心修正：调整结构体字段的json标签，与API返回的小写驼峰键名匹配
type BochaSearchResponse struct {
	Code  int    `json:"code"`
	LogID string `json:"log_id"`
	Msg   string `json:"msg"`
	Data  struct {
		WebPages struct { // 对应API的"data.webPages"
			Value []struct { // 对应API的"data.webPages.value"
				Name          string `json:"name"`
				Url           string `json:"url"`
				Snippet       string `json:"snippet"`
				DatePublished string `json:"datePublished"` // 修正字段名，匹配API的"datePublished"
			} `json:"value"`
		} `json:"webPages"` // 关键修正：标签改为小写"webPages"
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

// 初始化函数（通用搜索工具）
func NewLLMService(apiKey, baseURL, modelName string, maxTokens int, timeoutSec int, bochaAPIKey string) LLMService {
	tools := []Tool{
		{
			Type: "function",
			Function: Function{
				Name: "bocha_search",
				Description: "使用博查全领域高级搜索API获取实时信息，支持任意领域查询。" +
					"查询时请确保关键词具体明确（如“杭州余杭区 2025年7月29日 天气”）。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "具体搜索关键词，尽量包含时间、地点等关键信息",
						},
						"freshness": map[string]interface{}{
							"type":        "string",
							"description": "信息新鲜度，可选值：oneDay（1天内）、oneWeek（1周内）、oneMonth（1月内）",
							"default":     "oneDay",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "返回结果数量（最大50）",
							"default":     5,
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

// 流式生成基础实现
func (s *llmServiceImpl) StreamGenerate(ctx context.Context, messages []Message, tools []Tool) (<-chan string, <-chan error) {
	contentChan, errChan := make(chan string), make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		// 系统提示：引导LLM处理空结果
		enhancedMessages := append([]Message{
			{
				Role: "system",
				Content: "若搜索结果为空，请明确告知用户当前未找到相关信息，" +
					"并建议用户调整关键词或补充更多细节后重试。",
			},
		}, messages...)

		reqBody := map[string]interface{}{
			"model":      s.modelName,
			"messages":   enhancedMessages,
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

// 带搜索功能的流式生成
func (s *llmServiceImpl) StreamGenerateWithSearch(ctx context.Context, messages []Message) (<-chan string, <-chan error) {
	contentChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			close(contentChan)
			close(errChan)
			log.Println("所有流式数据处理完成")
		}()

		log.Println("开始第一次LLM调用（判断是否需要工具）")
		streamChan, streamErrChan := s.StreamGenerate(ctx, messages, s.tools)

		toolCalls, assistantMsg, err := s.parseToolCalls(streamChan, streamErrChan, contentChan)
		if err != nil {
			errChan <- fmt.Errorf("第一次调用解析失败: %w", err)
			return
		}

		if len(toolCalls) == 0 {
			log.Println("无需工具调用，第一次调用流式内容已完成")
			return
		}

		log.Printf("检测到%d个工具调用，执行搜索后发起第二次调用", len(toolCalls))
		messages = append(messages, assistantMsg)
		toolResults := s.executeTools(ctx, toolCalls)
		messages = append(messages, toolResults...)

		log.Println("开始第二次LLM调用（生成最终回答）")
		finalChan, finalErrChan := s.StreamGenerate(ctx, messages, s.tools)

		if err := s.forwardStream(finalChan, finalErrChan, contentChan); err != nil {
			errChan <- fmt.Errorf("第二次调用转发失败: %w", err)
			return
		}

		log.Println("第二次LLM调用流式内容处理完成")
	}()

	return contentChan, errChan
}

// 解析流式响应
func (s *llmServiceImpl) parseToolCalls(streamChan <-chan string, errChan <-chan error, contentChan chan<- string) ([]ToolCall, Message, error) {
	type partialTool struct {
		id        string
		name      string
		arguments strings.Builder
	}
	partials := make(map[int]*partialTool)
	var assistantMsg Message
	assistantMsg.Role = "assistant"

	for chunk := range streamChan {
		var resp struct {
			Choices []struct {
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(chunk), &resp); err != nil {
			log.Printf("解析流式chunk失败（非致命）: %v, chunk: %s", err, chunk)
			continue
		}

		for _, choice := range resp.Choices {
			if choice.Delta.Content != "" {
				log.Printf("第一次调用流式内容: %s", choice.Delta.Content)
				contentChan <- choice.Delta.Content
				assistantMsg.Content += choice.Delta.Content
			}

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
		return nil, assistantMsg, fmt.Errorf("第一次调用流式错误: %w", err)
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

	return toolCalls, assistantMsg, nil
}

// 执行工具调用
func (s *llmServiceImpl) executeTools(ctx context.Context, calls []ToolCall) []Message {
	var results []Message
	log.Printf("开始执行%d个工具调用", len(calls))

	for i, call := range calls {
		log.Printf("执行第%d个工具调用: %s", i+1, call.Function.Name)

		if call.Function.Name != "bocha_search" {
			results = append(results, Message{
				Role:       "tool",
				Content:    fmt.Sprintf("不支持的工具: %s", call.Function.Name),
				ToolCallID: call.ID,
			})
			continue
		}

		// 解析参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
			results = append(results, Message{
				Role:       "tool",
				Content:    fmt.Sprintf("参数解析错误: %v", err),
				ToolCallID: call.ID,
			})
			continue
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

		// 构建搜索请求
		freshness := "oneDay"
		if f, ok := params["freshness"].(string); ok && f != "" {
			freshness = f
		}

		count := 10
		if c, ok := params["count"].(float64); ok && c > 0 {
			count = int(c)
		}

		searchReq := BochaSearchRequest{
			Query:     query,
			Freshness: freshness,
			Count:     count,
			Summary:   true,
		}

		// 执行搜索（最多重试3次）
		var resp *BochaSearchResponse
		var err error
		maxRetries := 3
		for retry := 0; retry < maxRetries; retry++ {
			// 输出每次请求参数
			reqJSON, _ := json.Marshal(searchReq)
			log.Printf("第%d次尝试调用博查API，请求参数: %s", retry+1, string(reqJSON))

			resp, err = s.callBochaAPI(ctx, searchReq)
			if err != nil {
				log.Printf("第%d次搜索失败（将重试）：%v", retry+1, err)
				time.Sleep(1 * time.Second)
				continue
			}
			// 修正：正确判断结果数量（依赖解析后的resp）
			if len(resp.Data.WebPages.Value) > 0 {
				log.Printf("第%d次搜索成功，获取到%d条结果", retry+1, len(resp.Data.WebPages.Value))
				break
			}
			log.Printf("第%d次搜索无结果（将重试）：%s", retry+1, query)
			time.Sleep(1 * time.Second)
		}

		// 处理结果
		if err != nil {
			results = append(results, Message{
				Role:       "tool",
				Content:    fmt.Sprintf("搜索失败（已重试3次）: %v", err),
				ToolCallID: call.ID,
			})
			continue
		}

		// 输出API原始响应（用于调试）
		respJSON, _ := json.Marshal(resp)
		log.Printf("博查API最终响应: %s", string(respJSON))

		formattedResult := s.formatSearchResult(resp)
		results = append(results, Message{
			Role:       "tool",
			Content:    formattedResult,
			ToolCallID: call.ID,
		})
		log.Printf("工具调用完成，获取到%d条结果", len(resp.Data.WebPages.Value))
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

	// 输出完整请求信息
	log.Printf("调用博查API，URL: %s，请求体: %s", apiURL, string(reqBytes))

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

	// 输出完整响应信息（用于调试）
	log.Printf("博查API响应状态码: %d，响应体: %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP错误: %d, 内容: %s", resp.StatusCode, string(respBody))
	}

	var searchResp BochaSearchResponse
	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w，响应体: %s", err, string(respBody))
	}

	if searchResp.Code != 200 {
		return nil, fmt.Errorf("业务错误: %s", searchResp.Msg)
	}

	return &searchResp, nil
}

// 格式化搜索结果（通用处理）
func (s *llmServiceImpl) formatSearchResult(resp *BochaSearchResponse) string {
	var result strings.Builder

	// 无结果提示
	if len(resp.Data.WebPages.Value) == 0 {
		result.WriteString("未找到相关搜索结果，请尝试调整关键词或补充更多细节后重试。")
		return result.String()
	}

	// 有结果时正常格式化
	result.WriteString("以下是搜索结果：\n\n")
	for i, item := range resp.Data.WebPages.Value {
		result.WriteString(fmt.Sprintf("%d. %s\n发布时间: %s\n摘要: %s\n链接: %s\n\n",
			i+1, item.Name, item.DatePublished, item.Snippet, item.Url))
	}
	return result.String()
}

// 转发流式结果
func (s *llmServiceImpl) forwardStream(finalChan <-chan string, finalErrChan <-chan error, contentChan chan<- string) error {
	hasContent := false
	for chunk := range finalChan {
		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(chunk), &streamResp); err != nil {
			log.Printf("第二次调用解析chunk失败（非致命）: %v, chunk: %s", err, chunk)
			continue
		}

		for _, choice := range streamResp.Choices {
			if choice.Delta.Content != "" {
				hasContent = true
				log.Printf("第二次调用流式内容: %s", choice.Delta.Content)
				contentChan <- choice.Delta.Content
			}
		}
	}

	if !hasContent {
		log.Println("第二次调用LLM未返回内容")
		contentChan <- "抱歉，暂时无法获取相关信息。请尝试调整问题或提供更多细节。"
	}

	if err := <-finalErrChan; err != nil {
		return fmt.Errorf("第二次调用流式错误: %w", err)
	}

	return nil
}
