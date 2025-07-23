package api

import (
	"Voice_Assistant/internal/api/handler"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(assistantHandler *handler.AssistantHandler, historyHandler *handler.HistoryHandler) http.Handler {
	r := gin.Default()

	// CORS 中间件
	// CORS 中间件（适配SSE）
	r.Use(func(c *gin.Context) {
		// 允许前端域名（开发环境可用*，生产环境需指定具体域名）
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		// 允许的方法（包含OPTIONS预检请求）
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		// 允许的请求头（包含SSE需要的Accept）
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		// 允许暴露的响应头（SSE可能需要）
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Type, X-Accel-Buffering")
		// 允许携带凭证（如Cookie，按需开启）
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// 处理OPTIONS预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent) // 204 No Content
			return
		}

		c.Next()
	})

	apiV1a := r.Group("/api/voice-robot/v1/assistant")
	{
		apiV1a.GET("", assistantHandler.SelectAll)
		apiV1a.DELETE("/:id", assistantHandler.DeleteByID)
		apiV1a.POST("", assistantHandler.Save)
		apiV1a.PATCH("/:id", assistantHandler.UpdateByID)
	}

	apiV1h := r.Group("/api/voice-robot/v1/history")
	{
		apiV1h.GET("/:assistant_id", historyHandler.SelectByAssistantID)
		apiV1h.DELETE("/:assistant_id", historyHandler.ResetByAssistantID)
		apiV1h.POST("/:assistant_id", historyHandler.SaveByAssistantID)
		apiV1h.POST("/:assistant_id/stream-process", historyHandler.StreamProcessMessage)
	}

	return r
}
