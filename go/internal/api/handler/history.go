package handler

import (
	"Voice_Assistant/internal/model"
	"Voice_Assistant/internal/service"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HistoryHandler struct {
	historyService service.HistoryService
}

func NewHistoryHandler(historyService service.HistoryService) *HistoryHandler {
	return &HistoryHandler{historyService: historyService}
}

func (h *HistoryHandler) SelectByAssistantID(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "助手ID不能为空",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	history, err := h.historyService.SelectByAssistantID(c.Request.Context(), assistantID)
	if err != nil {
		log.Printf("查询历史记录失败: %v", err)
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     err.Error(),
			Code:    http.StatusInternalServerError,
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "查询成功",
		Code:    http.StatusOK,
		Data:    history,
	})
}

func (h *HistoryHandler) ResetByAssistantID(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "助手ID不能为空",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	if err := h.historyService.ResetByAssistantID(c.Request.Context(), assistantID); err != nil {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "重置对话失败: " + err.Error(),
			Code:    http.StatusInternalServerError,
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "对话已重置",
		Code:    http.StatusOK,
		Data:    nil,
	})
}

func (h *HistoryHandler) SaveByAssistantID(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "助手ID不能为空",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	var req struct {
		Input  model.Input
		Output model.Output
		Usage  model.Usage
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "消息格式错误",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	message := model.Message{
		Input:     req.Input,
		Output:    req.Output,
		Usage:     req.Usage,
		GmtCreate: time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := h.historyService.SaveByAssistantID(c.Request.Context(), assistantID, message); err != nil {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "保存消息失败: " + err.Error(),
			Code:    http.StatusInternalServerError,
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "消息添加成功",
		Code:    http.StatusOK,
		Data:    message,
	})
}

func (h *HistoryHandler) ProcessMessage(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "助手ID不能为空",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	var input model.Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "消息格式错误",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	message, err := h.historyService.ProcessMessage(c.Request.Context(), assistantID, input)
	if err != nil {
		log.Printf("处理消息失败: %v", err)
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "处理消息失败: " + err.Error(),
			Code:    http.StatusInternalServerError,
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "消息处理成功",
		Code:    http.StatusOK,
		Data:    message,
	})
}

func (h *HistoryHandler) StreamProcessMessage(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Msg:     "助手ID不能为空",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var input model.Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Msg:     "消息格式错误: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.(http.Flusher).Flush()

	contentChan, errChan, usage, err := h.historyService.StreamProcessMessage(c.Request.Context(), assistantID, input)
	if err != nil {
		log.Printf("handler调用服务失败: %v", err)
		c.Writer.WriteString(fmt.Sprintf("data: {\"error\": \"%s\"}\n\n", err.Error()))
		c.Writer.(http.Flusher).Flush()
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			// 客户端主动中止请求，服务层会处理保存逻辑
			return
		case content, ok := <-contentChan:
			if !ok {
				doneData, _ := json.Marshal(map[string]interface{}{
					"done":  true,
					"usage": usage,
				})
				c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", doneData))
				c.Writer.(http.Flusher).Flush()
				return
			}
			data, _ := json.Marshal(map[string]string{"content": content})
			c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", data))
			c.Writer.(http.Flusher).Flush()
		case err := <-errChan:
			if err == nil {
				log.Printf("回答结束")
				return
			}
			// 区分上下文取消错误，此时服务层已保存内容
			if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
				return
			}
			data, _ := json.Marshal(map[string]string{"error": err.Error()})
			c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", data))
			c.Writer.(http.Flusher).Flush()
			return
		case <-ticker.C:
			c.Writer.WriteString("data: {\"heartbeat\": true}\n\n")
			c.Writer.(http.Flusher).Flush()
		}
	}
}
