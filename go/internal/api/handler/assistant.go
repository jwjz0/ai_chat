package handler

import (
	"Voice_Assistant/internal/model"
	"Voice_Assistant/internal/service"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AssistantHandler struct {
	assistantService service.AssistantService
}

func NewAssistantHandler(assistantService service.AssistantService) *AssistantHandler {
	return &AssistantHandler{assistantService: assistantService}
}

// SelectAll 获取所有助手
func (h *AssistantHandler) SelectAll(c *gin.Context) {
	assistants, err := h.assistantService.SelectAll(c.Request.Context())
	if err != nil {
		log.Printf("获取助手列表失败: %v", err)
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "获取失败",
			Code:    http.StatusInternalServerError,
			Data:    nil,
		})
		return
	}
	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "获取成功",
		Code:    http.StatusOK,
		Data:    assistants,
	})
}

// DeleteByID 根据ID删除助手
func (h *AssistantHandler) DeleteByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "ID不能为空",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	if err := h.assistantService.DeleteByID(c.Request.Context(), id); err != nil {
		log.Printf("删除助手失败(ID: %s): %v", id, err)
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "删除失败",
			Code:    http.StatusInternalServerError,
			Data:    nil,
		})
		return
	}
	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "删除成功",
		Code:    http.StatusOK,
		Data:    nil,
	})
}

// Save 创建新助手
func (h *AssistantHandler) Save(c *gin.Context) {
	var assistant model.Assistant
	if err := c.ShouldBindJSON(&assistant); err != nil {
		log.Printf("解析请求失败: %v", err)
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "参数格式错误",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	if assistant.Name == "" {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "名称为必填项",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	saved, err := h.assistantService.Save(c.Request.Context(), &assistant)
	if err != nil {
		log.Printf("保存助手失败: %v", err)
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "创建失败",
			Code:    http.StatusInternalServerError,
			Data:    nil,
		})
		return
	}
	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "创建成功",
		Code:    http.StatusOK,
		Data:    saved,
	})
}

// UpdateByID 根据ID更新助手
func (h *AssistantHandler) UpdateByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "ID不能为空",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	var assistant model.Assistant
	if err := c.ShouldBindJSON(&assistant); err != nil {
		log.Printf("解析请求失败: %v", err)
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "参数格式错误",
			Code:    http.StatusBadRequest,
			Data:    nil,
		})
		return
	}

	updated, err := h.assistantService.UpdateByID(c.Request.Context(), id, &assistant)
	if err != nil {
		log.Printf("更新助手失败(ID: %s): %v", id, err)
		c.JSON(http.StatusOK, model.Result{
			Success: false,
			Msg:     "更新失败",
			Code:    http.StatusInternalServerError,
			Data:    nil,
		})
		return
	}
	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "更新成功",
		Code:    http.StatusOK,
		Data:    updated,
	})
}
