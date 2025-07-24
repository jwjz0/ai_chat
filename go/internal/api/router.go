package api

import (
	"Voice_Assistant/internal/api/handler"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(assistantHandler *handler.AssistantHandler, historyHandler *handler.HistoryHandler) http.Handler {
	r := gin.Default()

	// CORS 中间件（适配SSE）
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Type, X-Accel-Buffering")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
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
