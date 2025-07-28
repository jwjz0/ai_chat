package handler

import (
	"Voice_Assistant/internal/model"
	"Voice_Assistant/internal/service"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type HistoryHandler struct {
	historyService service.HistoryService
}

func NewHistoryHandler(historyService service.HistoryService) *HistoryHandler {
	return &HistoryHandler{historyService: historyService}
}

// SelectByAssistantID 查询历史记录
func (h *HistoryHandler) SelectByAssistantID(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if !isValidUUID(assistantID) {
		c.JSON(http.StatusBadRequest, model.Result{Success: false, Msg: "无效的助手ID"})
		return
	}

	history, err := h.historyService.SelectByAssistantID(c.Request.Context(), assistantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{Success: false, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Result{Success: true, Data: history})
}

// ResetByAssistantID 重置对话历史
func (h *HistoryHandler) ResetByAssistantID(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if !isValidUUID(assistantID) {
		c.JSON(http.StatusBadRequest, model.Result{Success: false, Msg: "无效的助手ID"})
		return
	}

	if err := h.historyService.ResetByAssistantID(c.Request.Context(), assistantID); err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{Success: false, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Result{Success: true, Msg: "对话已重置"})
}

// SaveByAssistantID 手动保存消息
func (h *HistoryHandler) SaveByAssistantID(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if !isValidUUID(assistantID) {
		c.JSON(http.StatusBadRequest, model.Result{Success: false, Msg: "无效的助手ID"})
		return
	}

	var req struct {
		Input  model.Input  `json:"input"`
		Output model.Output `json:"output"`
		Usage  model.Usage  `json:"usage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: false, Msg: "格式错误: " + err.Error()})
		return
	}

	message := model.Message{
		Input:     req.Input,
		Output:    req.Output,
		Usage:     req.Usage,
		GmtCreate: time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := h.historyService.SaveByAssistantID(c.Request.Context(), assistantID, message); err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{Success: false, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Result{Success: true, Data: message})
}

// StreamProcessMessage 流式处理消息（修复未使用变量）
func (h *HistoryHandler) StreamProcessMessage(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	var input model.Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: false, Msg: "格式错误: " + err.Error()})
		return
	}

	// 设置SSE响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.(http.Flusher).Flush()

	// 调用服务层（修复：直接使用llmErrChan，不定义未使用的变量）
	contentChan, llmErrChan, _, err := h.historyService.StreamProcessMessage(c.Request.Context(), assistantID, input)
	if err != nil {
		c.Writer.WriteString(fmt.Sprintf("data: {\"error\":\"%s\"}\n\n", err.Error()))
		c.Writer.(http.Flusher).Flush()
		return
	}

	// 转发流式内容
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for chunk := range contentChan {
			data, _ := json.Marshal(map[string]string{"content": chunk})
			c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", data))
			c.Writer.(http.Flusher).Flush()
		}
	}()

	// 处理错误（直接使用llmErrChan）
	go func() {
		if err := <-llmErrChan; err != nil {
			data, _ := json.Marshal(map[string]string{"error": err.Error()})
			c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", data))
			c.Writer.(http.Flusher).Flush()
		}
	}()

	// 等待所有内容处理完成
	wg.Wait()
}
