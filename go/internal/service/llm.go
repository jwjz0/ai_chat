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
	"net"
	"net/http"
	"strings"
	"time"

	"Voice_Assistant/internal/model"
)

// 博查通用搜索API结构
type BochaSearchRequest struct {
	Query     string `json:"query"`     // 搜索关键词
	Freshness string `json:"freshness"` // 时间范围
	Summary   bool   `json:"summary"`   // 显示摘要
	Count     int    `json:"count"`     // 结果数量
}

type BochaSearchResponse struct {
	Code  int    `json:"code"`   // 200=成功
	LogID string `json:"log_id"` // 日志ID
	Msg   string `json:"msg"`    // 响应消息
	Data  struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`          // 结果标题
				Url     string `json:"url"`           // 结果链接
				Snippet string `json:"snippet"`       // 结果摘要
				Time    string `json:"datePublished"` // 发布时间
			} `json:"value"`
		} `json:"webPages"`
	} `json:"data"`
}

// 工具调用结构
type ToolCall struct {
	Name       string             `json:"name"`
	Parameters BochaSearchRequest `json:"parameters"`
}

// 消息结构
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"` // 用于标识工具调用
}

// LLM服务接口
type LLMService interface {
	GenerateReply(ctx context.Context, prompt string, input string) (model.Output, model.Usage, error)
	StreamGenerate(ctx context.Context, messages []message) (<-chan string, <-chan error)
	StreamGenerateWithSearch(ctx context.Context, messages []message) (<-chan string, <-chan error)
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
}

func NewLLMService(apiKey, baseURL, modelName string, maxTokens int, timeoutSec int, bochaAPIKey string) LLMService {
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
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
		bochaAPIKey: bochaAPIKey,
	}
}

// 非流式生成
func (s *llmServiceImpl) GenerateReply(ctx context.Context, prompt string, input string) (model.Output, model.Usage, error) {
	reqBody := chatRequest{Model: s.modelName, Messages: []message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: input},
	}, MaxTokens: s.maxTokens}

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
		return model.Output{}, model.Usage{}, fmt.Errorf("API错误: %d, 内容: %s", resp.StatusCode, string(respBody))
	}

	var response chatResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return model.Output{}, model.Usage{}, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(response.Choices) == 0 {
		return model.Output{}, model.Usage{}, errors.New("无生成结果")
	}

	return model.Output{
			FinishReason: response.Choices[0].FinishReason,
			Content:      response.Choices[0].Message.Content,
		}, model.Usage{
			InputTokens:  response.Usage.PromptTokens,
			OutputTokens: response.Usage.CompletionTokens,
			TotalTokens:  response.Usage.TotalTokens,
		}, nil
}

// 流式生成
func (s *llmServiceImpl) StreamGenerate(ctx context.Context, messages []message) (<-chan string, <-chan error) {
	contentChan, errChan := make(chan string), make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		reqBody := chatRequest{Model: s.modelName, Messages: messages, MaxTokens: s.maxTokens, Stream: true}
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
				if line == "" || line == "data: [DONE]" {
					continue
				}
				if strings.HasPrefix(line, "data: ") {
					var streamResp struct {
						Choices []struct{ Delta struct{ Content string } } `json:"choices"`
						Error   *struct{ Message string }                  `json:"error"`
					}
					if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &streamResp); err != nil {
						continue
					}
					if streamResp.Error != nil {
						errChan <- fmt.Errorf("模型错误: %s", streamResp.Error.Message)
						return
					}
					if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
						contentChan <- streamResp.Choices[0].Delta.Content
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("读取流失败: %w", err)
		}
	}()

	return contentChan, errChan
}

// 带搜索的流式生成
func (s *llmServiceImpl) StreamGenerateWithSearch(ctx context.Context, messages []message) (<-chan string, <-chan error) {
	contentChan, errChan := make(chan string), make(chan error, 1)
	var firstResponse, userVisibleResponse strings.Builder

	go func() {
		defer close(contentChan)
		defer close(errChan)

		// 提取用户问题并分析
		lastUserMsg := ""
		for _, msg := range messages {
			if msg.Role == "user" {
				lastUserMsg = msg.Content
			}
		}
		needsSearch := s.needsRealTimeInfo(lastUserMsg) || s.hasRecentTimeKeyword(lastUserMsg)
		log.Printf("用户问题分析: 需要搜索=%v, 问题内容=%s", needsSearch, lastUserMsg)

		// 第一轮调用LLM判断是否需要搜索
		log.Println("第一轮调用LLM（判断是否需要搜索）")
		enhancedMessages := append(messages, message{
			Role: "system",
			Content: `如果用户问题涉及实时/动态信息，请使用格式：
<|FunctionCallBegin|>[{"name":"bocha_search","parameters":{"query":"[自然语言查询，非关键词]"}}]<|FunctionCallEnd|>

注意：
- "query"的值必须是完整的自然语言句子（如"2025年7月深圳房价走势"，而非"深圳 房价 2025"）；
- 包含时间、地点等关键信息（如用户问"最近北京天气"，生成"2025年7月北京近期天气情况"）。`,
		})

		preChan, preErrChan := s.StreamGenerate(ctx, enhancedMessages)
		for preChan != nil || preErrChan != nil {
			select {
			case chunk, ok := <-preChan:
				if !ok {
					preChan = nil
					break
				}
				firstResponse.WriteString(chunk)
				if !strings.Contains(chunk, "<|FunctionCallBegin|>") && !strings.Contains(chunk, "<|FunctionCallEnd|>") {
					userVisibleResponse.WriteString(chunk)
				}
			case err, ok := <-preErrChan:
				if !ok {
					preErrChan = nil
					break
				}
				log.Printf("第一轮调用错误: %v", err)
				contentChan <- "抱歉，处理请求时遇到问题，请稍后再试~"
				return
			case <-ctx.Done():
				log.Printf("上下文取消: %v", ctx.Err())
				return
			}
		}
		log.Printf("第一轮响应: %s", firstResponse.String())

		// 提取工具调用信息
		var toolCalls []ToolCall
		responseStr := firstResponse.String()
		startTag := "<|FunctionCallBegin|>"
		endTag := "<|FunctionCallEnd|>"

		startIdx := strings.Index(responseStr, startTag)
		endIdx := strings.Index(responseStr, endTag)

		if startIdx != -1 && endIdx != -1 {
			// 提取并清理JSON字符串
			jsonStr := responseStr[startIdx+len(startTag) : endIdx]
			jsonStr = strings.TrimSpace(jsonStr)

			log.Printf("提取的工具调用JSON: %s", jsonStr)

			// 尝试解析JSON
			if err := json.Unmarshal([]byte(jsonStr), &toolCalls); err != nil {
				// 尝试修复常见的JSON格式问题
				fixedJSON := s.fixJSONFormat(jsonStr)
				if fixedJSON != jsonStr {
					log.Printf("尝试修复JSON格式: %s", fixedJSON)
					if err := json.Unmarshal([]byte(fixedJSON), &toolCalls); err != nil {
						log.Printf("修复后仍解析失败: %v", err)
					} else {
						log.Printf("JSON解析成功（修复后）")
					}
				} else {
					log.Printf("解析工具调用失败: %v", err)
				}
			} else {
				log.Printf("JSON解析成功")
			}
		}

		// 强制搜索逻辑
		if len(toolCalls) == 0 && needsSearch {

			// 新逻辑：使用LLM生成自然语言query
			query, err := s.generateQueryByLLM(ctx, lastUserMsg)
			if err != nil {
				log.Printf("LLM生成query失败，使用备用方法: %v", err)
				query = s.generatePreciseQuery(lastUserMsg) // 降级方案
			} else {
				// 可选：增强query（补充时间/地点等信息）
				query = s.enhanceQuery(query)
			}

			log.Printf("未检测到工具调用，但需要实时信息，强制触发搜索，查询: %s", query)
			toolCalls = []ToolCall{{
				Name: "bocha_search",
				Parameters: BochaSearchRequest{
					Query:     query,
					Freshness: "oneWeek",
					Summary:   true,
					Count:     5,
				},
			}}
		}

		// 执行搜索并生成回答
		if len(toolCalls) > 0 {
			var searchResults []string
			for _, toolCall := range toolCalls {
				if toolCall.Name == "bocha_search" {
					// 确保搜索参数正确
					if toolCall.Parameters.Count <= 0 {
						toolCall.Parameters.Count = 10
					}
					if !toolCall.Parameters.Summary {
						toolCall.Parameters.Summary = true
					}
					log.Printf("执行搜索，查询: %s, 预期结果数量: %d",
						toolCall.Parameters.Query, toolCall.Parameters.Count)

					// 调用搜索API
					searchResp, err := s.callBochaAPI(ctx, toolCall.Parameters)
					if err != nil {
						log.Printf("搜索失败: %v", err)
						contentChan <- fmt.Sprintf("抱歉，获取最新信息失败：%v", err)
						return
					}

					// 处理搜索结果
					if len(searchResp.Data.WebPages.Value) == 0 {
						log.Println("警告：搜索结果为空")
						searchResults = append(searchResults, "未找到相关最新信息。")
					} else {
						log.Printf("获取到 %d 条搜索结果", len(searchResp.Data.WebPages.Value))
						searchResults = append(searchResults, s.formatSearchResult(searchResp))
					}
				}
			}

			if len(searchResults) > 0 {
				// 第二轮调用LLM生成最终回答
				log.Println("第二轮调用LLM（基于搜索结果生成回答）")
				newMessages := append(messages,
					message{Role: "assistant", Content: userVisibleResponse.String()},
					message{Role: "function", Name: "bocha_search", Content: strings.Join(searchResults, "\n\n")},
					message{Role: "system", Content: "请基于搜索结果自然流畅地回答用户问题"})

				finalChan, finalErrChan := s.StreamGenerate(ctx, newMessages)
				s.forwardStream(finalChan, finalErrChan, contentChan)
				return
			}
		}

		// 无需搜索，直接返回
		contentChan <- userVisibleResponse.String()
	}()

	return contentChan, errChan
}

// 判断是否需要实时信息
func (s *llmServiceImpl) needsRealTimeInfo(text string) bool {
	for _, kw := range []string{"最新", "现在", "当前", "正在", "近况"} {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// 检测近期时间关键词
func (s *llmServiceImpl) hasRecentTimeKeyword(text string) bool {
	for _, pattern := range []string{"近期", "这周", "本月", "今天", "昨日"} {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

// 生成精准搜索关键词（保留原方法作为备用）
func (s *llmServiceImpl) generatePreciseQuery(userMsg string) string {
	query := strings.TrimSpace(
		strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(userMsg, "你知道", ""),
					"请问", ""),
				"吗", ""),
			"？", ""))

	if strings.Contains(query, "近期") && !strings.Contains(query, "2025") {
		query += " 2025"
	}
	log.Printf("规则生成的搜索关键词: %s", query)
	return query
}

// 新增：通过LLM生成精准搜索query
func (s *llmServiceImpl) generateQueryByLLM(ctx context.Context, userMsg string) (string, error) {
	// 提示词：明确要求生成自然语言查询，而非关键词
	prompt := fmt.Sprintf(`请将用户问题转化为一个精准的搜索查询（用自然语言句子表达，而非关键词堆砌）。
要求：
1. 包含所有关键信息（如时间、地点、事件）；
2. 去除冗余语气词，但保留必要上下文；
3. 适合搜索引擎理解（例如用户问"最近广州的暴雨情况"，生成"2025年近期广州暴雨天气情况"）。

用户问题：%s
生成的搜索查询：`, userMsg)

	// 调用LLM生成query（使用非流式接口，确保一次性获取结果）
	output, _, err := s.GenerateReply(ctx, "", prompt)
	if err != nil {
		return "", fmt.Errorf("LLM生成query失败: %w", err)
	}

	query := strings.TrimSpace(output.Content)
	log.Printf("LLM生成的搜索查询: %s", query)
	return query, nil
}

func (s *llmServiceImpl) enhanceQuery(query string) string {
	if !strings.Contains(query, "2025") && (strings.Contains(query, "近期") || strings.Contains(query, "最近")) {
		query += " 2025年"
	}
	return query
}

// 尝试修复JSON格式问题
func (s *llmServiceImpl) fixJSONFormat(jsonStr string) string {
	// 简单的JSON格式修复（根据常见问题）
	// 1. 移除多余的方括号
	jsonStr = strings.Trim(jsonStr, "[]")
	jsonStr = fmt.Sprintf("[%s]", jsonStr)

	// 2. 移除多余的逗号
	jsonStr = strings.TrimRight(jsonStr, ",")
	jsonStr = strings.TrimRight(jsonStr, " ]") + "]"

	return jsonStr
}

// 调用博查搜索API
func (s *llmServiceImpl) callBochaAPI(ctx context.Context, req BochaSearchRequest) (*BochaSearchResponse, error) {
	if req.Query == "" {
		return nil, errors.New("搜索关键词不能为空")
	}
	if s.bochaAPIKey == "" {
		return nil, errors.New("博查API密钥未配置")
	}

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

	log.Printf("发送搜索请求: URL=%s, 参数=%+v", apiURL, req)
	resp, err := s.client.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("请求API失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	log.Printf("API响应状态码: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP错误: %d, 内容: %s", resp.StatusCode, string(respBody))
	}

	var searchResp BochaSearchResponse
	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 内容=%s", err, string(respBody))
	}

	if searchResp.Code != 200 {
		return nil, fmt.Errorf("业务错误: code=%d, 消息=%s", searchResp.Code, searchResp.Msg)
	}

	return &searchResp, nil
}

// 格式化搜索结果
func (s *llmServiceImpl) formatSearchResult(resp *BochaSearchResponse) string {
	var result strings.Builder
	result.WriteString("以下是相关最新信息：\n\n")

	for i, item := range resp.Data.WebPages.Value {
		result.WriteString(fmt.Sprintf(
			"%d. %s（发布时间：%s）\n摘要：%s\n\n",
			i+1, item.Name, item.Time, item.Snippet))
		log.Printf("%d. %s（发布时间：%s）\n摘要：%s\n\n", i+1, item.Name, item.Time, item.Snippet)
	}

	result.WriteString(fmt.Sprintf("共获取到 %d 条搜索结果。", len(resp.Data.WebPages.Value)))
	return result.String()
}

// 转发流式内容
func (s *llmServiceImpl) forwardStream(finalChan <-chan string, finalErrChan <-chan error, contentChan chan<- string) {
	for chunk := range finalChan {
		contentChan <- chunk
	}
	if err := <-finalErrChan; err != nil {
		log.Printf("生成回答失败: %v", err)
		contentChan <- "抱歉，生成回答时遇到问题，请重试~"
	}
}

// 内部请求/响应结构
type chatRequest struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
	Stream    bool      `json:"stream,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
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
