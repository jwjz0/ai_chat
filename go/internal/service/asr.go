package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// ASRService 定义ASR服务接口
type ASRService interface {
	StreamRecognize(ctx context.Context, audioStream io.Reader) (<-chan string, <-chan error)
}

// asrServiceImpl 腾讯云ASR实现（基于WebSocket）
type asrServiceImpl struct {
	secretID  string
	secretKey string
	region    string
	engine    string
}

func NewASRService(secretID, secretKey, region, engine string) ASRService {
	return &asrServiceImpl{
		secretID:  secretID,
		secretKey: secretKey,
		region:    region,
		engine:    engine,
	}
}

// 腾讯云ASR WebSocket请求参数
type asrRequest struct {
	Action          string `json:"action"`                // 固定为"StartRecognition"或"ContinueRecognition"
	Version         string `json:"version"`               // 版本，固定为"2020-09-28"
	Seq             int    `json:"seq"`                   // 序列号，递增
	Timestamp       int64  `json:"timestamp"`             // 时间戳
	SecretId        string `json:"secretid"`              // 腾讯云SecretId
	EngineModelType string `json:"enginemodeltype"`       // 引擎模型
	VoiceId         string `json:"voiceid"`               // 语音唯一标识（可选）
	AudioFormat     string `json:"audioformat"`           // 音频格式，如"pcm"
	SampleRate      int    `json:"samplerate"`            // 采样率，如16000
	ChannelNum      int    `json:"channelnum"`            // 声道数，固定1
	Data            string `json:"data,omitempty"`        // 音频数据（base64编码）
	End             int    `json:"end,omitempty"`         // 是否结束，1表示结束
	HotwordId       string `json:"hotwordid,omitempty"`   // 热词表ID（可选）
	NeedVad         int    `json:"needvad,omitempty"`     // 是否开启 vad，1表示开启
	FilterDirty     int    `json:"filterdirty,omitempty"` // 是否过滤脏词
	FilterModal     int    `json:"filtermodel,omitempty"` // 是否过滤语气词
	FilterPunc      int    `json:"filterpunc,omitempty"`  // 是否过滤标点符号
}

// 腾讯云ASR WebSocket响应
type asrResponse struct {
	Code      int    `json:"code"`      // 错误码，0表示成功
	Message   string `json:"message"`   // 错误信息
	Seq       int    `json:"seq"`       // 序列号
	Result    string `json:"result"`    // 识别结果
	VoiceId   string `json:"voiceid"`   // 语音标识
	End       int    `json:"end"`       // 是否最后一片结果
	Timestamp int64  `json:"timestamp"` // 时间戳
}

// StreamRecognize 处理流式语音识别（基于WebSocket）
func (s *asrServiceImpl) StreamRecognize(ctx context.Context, audioStream io.Reader) (<-chan string, <-chan error) {
	resultChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errChan)

		// 1. 构建WebSocket连接URL
		u := url.URL{
			Scheme: "wss",
			Host:   fmt.Sprintf("asr.%s.tencentcloudapi.com", s.region),
			Path:   "/",
		}

		// 2. 建立WebSocket连接
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			errChan <- fmt.Errorf("建立WebSocket连接失败: %w", err)
			return
		}
		defer conn.Close()

		// 3. 发送开始识别请求
		seq := 1
		startReq := asrRequest{
			Action:          "StartRecognition",
			Version:         "2020-09-28",
			Seq:             seq,
			Timestamp:       time.Now().Unix(),
			SecretId:        s.secretID,
			EngineModelType: s.engine,
			AudioFormat:     "pcm", // 假设前端发送PCM格式
			SampleRate:      16000, // 16k采样率
			ChannelNum:      1,     // 单声道
			NeedVad:         1,     // 开启VAD
		}
		seq++

		if err := conn.WriteJSON(startReq); err != nil {
			errChan <- fmt.Errorf("发送开始请求失败: %w", err)
			return
		}

		// 4. 启动协程接收识别结果
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					var resp asrResponse
					if err := conn.ReadJSON(&resp); err != nil {
						errChan <- fmt.Errorf("读取响应失败: %w", err)
						return
					}

					if resp.Code != 0 {
						errChan <- fmt.Errorf("ASR错误: 代码=%d, 信息=%s", resp.Code, resp.Message)
						return
					}

					if resp.Result != "" {
						log.Printf("ASR识别结果: %s", resp.Result)
						resultChan <- resp.Result
					}

					if resp.End == 1 {
						// 识别结束
						return
					}
				}
			}
		}()

		// 5. 读取音频流并发送
		buffer := make([]byte, 3200) // 16k采样率下100ms的PCM数据（16bit*1声道）
		for {
			select {
			case <-ctx.Done():
				// 发送结束标记
				conn.WriteJSON(asrRequest{
					Action:    "ContinueRecognition",
					Version:   "2020-09-28",
					Seq:       seq,
					Timestamp: time.Now().Unix(),
					End:       1, // 标记结束
				})
				errChan <- ctx.Err()
				return
			default:
				n, err := audioStream.Read(buffer)
				if err == io.EOF {
					// 发送结束标记
					conn.WriteJSON(asrRequest{
						Action:    "ContinueRecognition",
						Version:   "2020-09-28",
						Seq:       seq,
						Timestamp: time.Now().Unix(),
						End:       1, // 标记结束
					})
					return
				}
				if err != nil {
					errChan <- fmt.Errorf("读取音频流失败: %w", err)
					return
				}

				// 发送音频片段（base64编码）
				audioData := base64.StdEncoding.EncodeToString(buffer[:n])
				req := asrRequest{
					Action:    "ContinueRecognition",
					Version:   "2020-09-28",
					Seq:       seq,
					Timestamp: time.Now().Unix(),
					SecretId:  s.secretID,
					Data:      audioData,
				}
				seq++

				if err := conn.WriteJSON(req); err != nil {
					errChan <- fmt.Errorf("发送音频数据失败: %w", err)
					return
				}

				// 控制发送速率（避免过快发送）
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return resultChan, errChan
}
