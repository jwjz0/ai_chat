package handler

import (
	"Voice_Assistant/internal/model"
	"Voice_Assistant/internal/service"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

type AssistantHandler struct {
	assistantService service.AssistantService
}

func NewAssistantHandler(assistantService service.AssistantService) *AssistantHandler {
	return &AssistantHandler{assistantService: assistantService}
}

func isValidUUID(id string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(id)
}

// SelectAll 获取所有助手
func (h *AssistantHandler) SelectAll(c *gin.Context) {
	assistants, err := h.assistantService.SelectAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{
			Success: false,
			Msg:     err.Error(),
			Data:    nil,
		})
		return
	}
	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "获取成功",
		Data:    assistants,
	})
}

// DeleteByID 根据ID删除助手
func (h *AssistantHandler) DeleteByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Msg:     "ID不能为空",
			Data:    nil,
		})
		return
	}

	if !isValidUUID(id) {
		c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Msg:     "ID格式不正确",
			Data:    nil,
		})
		return
	}

	if err := h.assistantService.DeleteByID(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{
			Success: false,
			Msg:     err.Error(),
			Data:    nil,
		})
		return
	}
	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "删除成功",
		Data:    nil,
	})
}

// Save 创建新助手
func (h *AssistantHandler) Save(c *gin.Context) {
	var assistant model.Assistant
	if err := c.ShouldBindJSON(&assistant); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Msg:     "参数格式错误",
			Data:    nil,
		})
		return
	}

	if assistant.Name == "" {
		c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Msg:     "名称为必填项",
			Data:    nil,
		})
		return
	}

	saved, err := h.assistantService.Save(c.Request.Context(), &assistant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{
			Success: false,
			Msg:     err.Error(),
			Data:    nil,
		})
		return
	}
	c.JSON(http.StatusCreated, model.Result{
		Success: true,
		Msg:     "创建成功",
		Data:    saved,
	})
}

// UpdateByID 根据ID更新助手
func (h *AssistantHandler) UpdateByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Msg:     "ID不能为空",
			Data:    nil,
		})
		return
	}

	if !isValidUUID(id) {
		c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Msg:     "ID格式不正确",
			Data:    nil,
		})
		return
	}

	var assistant model.Assistant
	if err := c.ShouldBindJSON(&assistant); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Msg:     "参数格式错误",
			Data:    nil,
		})
		return
	}

	updated, err := h.assistantService.UpdateByID(c.Request.Context(), id, &assistant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{
			Success: false,
			Msg:     err.Error(),
			Data:    nil,
		})
		return
	}
	c.JSON(http.StatusOK, model.Result{
		Success: true,
		Msg:     "更新成功",
		Data:    updated,
	})
}
