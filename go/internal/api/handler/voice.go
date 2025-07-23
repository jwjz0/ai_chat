package handler

import (
	"io"
	"log"
	"net/http"

	"Voice_Assistant/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 升级HTTP连接为WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许跨域（生产环境需限制来源）
		return true
	},
	ReadBufferSize:  4096, // 读取缓冲区大小
	WriteBufferSize: 4096, // 写入缓冲区大小
}

type VoiceHandler struct {
	asrService service.ASRService
}

func NewVoiceHandler(asrService service.ASRService) *VoiceHandler {
	return &VoiceHandler{
		asrService: asrService,
	}
}

// HandleWebSocket 处理语音识别的WebSocket连接
func (h *VoiceHandler) HandleWebSocket(c *gin.Context) {
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("升级WebSocket失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法建立WebSocket连接"})
		return
	}
	defer conn.Close()

	// 创建音频流读取器（从WebSocket读取音频数据）
	audioStream := &websocketAudioReader{conn: conn}

	// 开始流式识别
	resultChan, errChan := h.asrService.StreamRecognize(c.Request.Context(), audioStream)

	// 处理识别结果并回传给客户端
	go func() {
		for result := range resultChan {
			log.Printf("ASR识别文本: %s", result)
			// 实时将识别结果通过WebSocket返回给前端
			if err := conn.WriteMessage(websocket.TextMessage, []byte(result)); err != nil {
				log.Printf("发送识别结果失败: %v", err)
				return
			}
		}
	}()

	// 处理错误
	select {
	case err := <-errChan:
		log.Printf("ASR错误: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("错误: "+err.Error()))
	case <-c.Request.Context().Done():
		log.Println("客户端断开连接")
	}
}

// websocketAudioReader 实现io.Reader接口，从WebSocket读取音频数据
type websocketAudioReader struct {
	conn *websocket.Conn
	buf  []byte // 缓存未读完的音频数据
}

// Read 从WebSocket读取音频数据（实现io.Reader接口）
func (r *websocketAudioReader) Read(p []byte) (int, error) {
	// 如果缓存中有数据，先读取缓存
	if len(r.buf) > 0 {
		n := copy(p, r.buf)
		r.buf = r.buf[n:]
		return n, nil
	}

	// 从WebSocket读取新数据
	mt, data, err := r.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	// 只处理二进制消息（音频数据）
	if mt != websocket.BinaryMessage {
		return 0, io.EOF
	}

	// 将数据复制到输出缓冲区
	n := copy(p, data)
	// 剩余数据存入缓存
	r.buf = data[n:]

	return n, nil
}
