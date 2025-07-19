package config

import (
	"Voice_Assistant/internal/api"
	"Voice_Assistant/internal/api/handler"
	"Voice_Assistant/internal/repository"
	"Voice_Assistant/internal/service"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构体
type Config struct {
	Server struct {
		Port string `yaml:"port"` // 服务器端口
	} `yaml:"server"`
	Data struct {
		AssistantsPath string `yaml:"assistants_path"` // 助手数据文件路径
		HistoriesPath  string `yaml:"histories_path"`  // 历史记录文件路径
	} `yaml:"data"`
	LLM struct { // 新增LLM配置
		APIKey     string `yaml:"api_key"`     // 通义千问API密钥
		BaseURL    string `yaml:"base_url"`    // API基础URL
		ModelName  string `yaml:"model_name"`  // 模型名称
		MaxTokens  int    `yaml:"max_tokens"`  // 最大生成tokens
		TimeoutSec int    `yaml:"timeout_sec"` // 超时时间(秒)
	} `yaml:"llm"`
}

var AppConfig Config

// 初始化配置
func init() {
	configPath := filepath.Join("internal", "config", "application.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		panic("配置文件读取失败: " + err.Error())
	}

	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		panic("配置文件解析失败: " + err.Error())
	}

	// 替换环境变量占位符
	AppConfig.LLM.APIKey = replaceEnvVar(AppConfig.LLM.APIKey)
}

// 工具函数：替换环境变量占位符（如${ENV_NAME} -> 环境变量值）
func replaceEnvVar(value string) string {
	if len(value) > 2 && value[0] == '$' && value[1] == '{' {
		envKey := value[2 : len(value)-1]
		if envVal := os.Getenv(envKey); envVal != "" {
			return envVal
		}
	}
	return value
}

// SetupApp 初始化应用
func SetupApp() http.Handler {
	defer log.Printf("服务器端口为 %s \n", AppConfig.Server.Port)

	// 初始化仓库
	assistantRepo := repository.NewAssistantRepo(
		AppConfig.Data.AssistantsPath,
		AppConfig.Data.HistoriesPath,
	)
	historyRepo := repository.NewHistoryRepo(
		AppConfig.Data.HistoriesPath,
		AppConfig.Data.AssistantsPath,
	)

	// 初始化大模型服务
	llmService := service.NewLLMService(
		AppConfig.LLM.APIKey,
		AppConfig.LLM.BaseURL,
		AppConfig.LLM.ModelName,
		AppConfig.LLM.MaxTokens,
		AppConfig.LLM.TimeoutSec,
	)

	// 初始化业务服务（注入大模型服务）
	historyService := service.NewHistoryService(historyRepo, assistantRepo, llmService)
	assistantService := service.NewAssistantService(assistantRepo, historyService)

	// 初始化处理器
	assistantHandler := handler.NewAssistantHandler(assistantService)
	historyHandler := handler.NewHistoryHandler(historyService)

	// 初始化路由
	return api.SetupRouter(assistantHandler, historyHandler)
}
