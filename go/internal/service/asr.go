package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// ASRService 定义ASR服务接口
type ASRService interface {
	StreamRecognize(ctx context.Context, audioStream io.Reader) (<-chan string, <-chan error)
}

// asrServiceImpl 腾讯云ASR实现（基于WebSocket）
type asrServiceImpl struct {
	appID     string // 已传入，但未使用到签名和连接参数
	secretID  string
	secretKey string
	region    string
	engine    string
}

func NewASRService(appID, secretID, secretKey, region, engine string) ASRService {
	if appID == "" || secretID == "" || secretKey == "" || region == "" {
		panic("ASR服务初始化失败: AppId、SecretId、SecretKey和Region均为必填参数")
	}
	return &asrServiceImpl{
		appID:     appID,
		secretID:  secretID,
		secretKey: secretKey,
		region:    region,
		engine:    engine,
	}
}

// 腾讯云ASR WebSocket请求参数（按文档v2版本）
type asrRequest struct {
	Action          string `json:"action"`
	Version         string `json:"version"`
	Seq             int    `json:"seq"`
	Timestamp       int64  `json:"timestamp"`
	Region          string `json:"region"`
	EngineModelType string `json:"enginemodeltype"`
	VoiceId         string `json:"voiceid"`
	AudioFormat     string `json:"audioformat"`
	SampleRate      int    `json:"samplerate"`
	ChannelNum      int    `json:"channelnum"`
	Data            string `json:"data,omitempty"`
	End             int    `json:"end,omitempty"`
	NeedVad         int    `json:"needvad"`
}

// 腾讯云ASR WebSocket响应
type asrResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Seq       int    `json:"seq"`
	Result    string `json:"result"`
	End       int    `json:"end"`
	Timestamp int64  `json:"timestamp"`
}

// 生成腾讯云API签名（关键修复：添加AppId，按字典序排序）
func (s *asrServiceImpl) generateSignature(nonce int) string {
	// 1. 签名参数必须包含AppId（必填！），并按字典序排序
	params := map[string]string{
		"Action":    "StartRecognition",
		"AppId":     s.appID, // 新增：添加AppId
		"Nonce":     strconv.Itoa(nonce),
		"Region":    s.region,
		"SecretId":  s.secretID,
		"Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		"Version":   "2020-09-28",
	}

	// 2. 按字典序排序参数（关键：腾讯云要求严格按字母顺序）
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys) // 排序后：Action, AppId, Nonce, Region, SecretId, Timestamp, Version

	// 3. 拼接参数为"key=value&key=value"（值需URL编码）
	var paramStr string
	for i, k := range keys {
		if i > 0 {
			paramStr += "&"
		}
		paramStr += fmt.Sprintf("%s=%s", k, url.QueryEscape(params[k]))
	}

	// 4. 签名原文格式："GETasr.tencentcloudapi.com?" + 排序后的参数串（无空格！）
	signatureBase := fmt.Sprintf("GETasr.tencentcloudapi.com?%s", paramStr)

	// 5. HMAC-SHA1加密 + Base64编码
	mac := hmac.New(sha1.New, []byte(s.secretKey))
	mac.Write([]byte(signatureBase))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// StreamRecognize 流式语音识别（修复WebSocket连接参数）
func (s *asrServiceImpl) StreamRecognize(ctx context.Context, audioStream io.Reader) (<-chan string, <-chan error) {
	resultChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errChan)

		// 1. 生成签名参数
		nonce := int(time.Now().UnixNano() % 1000000)
		timestamp := time.Now().Unix()
		signature := s.generateSignature(nonce) // 此时签名已包含AppId

		// 2. 构建WebSocket连接URL（必须包含AppId参数）
		u := url.URL{
			Scheme: "wss",
			Host:   "asr.tencentcloudapi.com",
			Path:   "/",
			RawQuery: url.Values{
				"Action":    {"StartRecognition"},
				"AppId":     {s.appID}, // 新增：URL中添加AppId
				"Nonce":     {strconv.Itoa(nonce)},
				"Region":    {s.region},
				"SecretId":  {s.secretID},
				"Timestamp": {strconv.FormatInt(timestamp, 10)},
				"Version":   {"2020-09-28"},
				"Signature": {signature}, // 包含AppId的签名
			}.Encode(),
		}

		// 3. 建立WebSocket连接（此时参数完整）
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			// 调试：打印完整URL（仅调试用，生产环境删除）
			log.Printf("WebSocket连接URL: %s", u.String())
			errChan <- fmt.Errorf("WebSocket连接失败: %w", err)
			return
		}
		defer conn.Close()

		// 4. 发送启动请求（StartRecognition）
		seq := 1
		startReq := asrRequest{
			Action:          "StartRecognition",
			Version:         "2020-09-28",
			Seq:             seq,
			Timestamp:       timestamp,
			Region:          s.region,
			EngineModelType: s.engine,
			AudioFormat:     "opus",
			SampleRate:      16000,
			ChannelNum:      1,
			NeedVad:         1,
		}
		if err := conn.WriteJSON(startReq); err != nil {
			errChan <- fmt.Errorf("发送启动请求失败: %w", err)
			return
		}
		seq++

		// 5. 接收识别结果（不变）
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
						errChan <- fmt.Errorf("ASR错误: %d-%s", resp.Code, resp.Message)
						return
					}

					if resp.Result != "" {
						log.Printf("ASR识别: %s", resp.Result)
						resultChan <- resp.Result
					}

					if resp.End == 1 {
						return
					}
				}
			}
		}()

		// 6. 发送音频流（不变）
		buffer := make([]byte, 3200)
		for {
			select {
			case <-ctx.Done():
				conn.WriteJSON(asrRequest{
					Action:    "ContinueRecognition",
					Seq:       seq,
					Timestamp: time.Now().Unix(),
					End:       1,
				})
				return
			default:
				n, err := audioStream.Read(buffer)
				if err == io.EOF {
					conn.WriteJSON(asrRequest{
						Action:    "ContinueRecognition",
						Seq:       seq,
						Timestamp: time.Now().Unix(),
						End:       1,
					})
					return
				}
				if err != nil {
					errChan <- fmt.Errorf("读取音频失败: %w", err)
					return
				}

				req := asrRequest{
					Action:    "ContinueRecognition",
					Seq:       seq,
					Timestamp: time.Now().Unix(),
					Data:      base64.StdEncoding.EncodeToString(buffer[:n]),
				}
				if err := conn.WriteJSON(req); err != nil {
					errChan <- fmt.Errorf("发送音频失败: %w", err)
					return
				}
				seq++

				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return resultChan, errChan
}
