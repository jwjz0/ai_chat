package config

import (
	"Voice_Assistant/internal/api"
	"Voice_Assistant/internal/api/handler"
	"Voice_Assistant/internal/data/sqlite"
	"Voice_Assistant/internal/repository"
	"Voice_Assistant/internal/service"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Data struct {
		DBPath string `yaml:"db_path"` // SQLite数据库路径（例如："./data/assistant.db"）
	} `yaml:"data"`
	LLM struct {
		APIKey     string `yaml:"api_key"`
		BaseURL    string `yaml:"base_url"`
		ModelName  string `yaml:"model_name"`
		MaxTokens  int    `yaml:"max_tokens"`
		TimeoutSec int    `yaml:"timeout_sec"`
	} `yaml:"llm"`
}

// LoadConfig 加载配置文件
func LoadConfig() (*Config, error) {
	configPath := filepath.Join("internal", "config", "application.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 替换环境变量占位符（如${DASHSCOPE_API_KEY}）
	cfg.LLM.APIKey = replaceEnvVar(cfg.LLM.APIKey)
	return &cfg, nil
}

// 替换环境变量占位符
func replaceEnvVar(value string) string {
	if len(value) > 2 && value[0] == '$' && value[1] == '{' {
		envKey := value[2 : len(value)-1]
		if envVal := os.Getenv(envKey); envVal != "" {
			return envVal
		}
	}
	return value
}

// SetupApp 初始化应用（核心入口）
func SetupApp() (http.Handler, *Config, error) {
	// 1. 加载配置
	cfg, err := LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 2. 创建数据库目录（确保目录存在）
	dbDir := filepath.Dir(cfg.Data.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 3. 初始化SQLite数据库（关键修复：调用sqlite.InitDB创建表）
	if err := sqlite.InitDB(cfg.Data.DBPath); err != nil {
		return nil, nil, fmt.Errorf("初始化数据库失败: %w", err)
	}
	db := sqlite.DB // 使用sqlite包中初始化好的全局连接

	// 4. 初始化数据仓库（依赖数据库连接）
	assistantRepo := repository.NewAssistantRepo(db)
	historyRepo := repository.NewHistoryRepo(db)

	// 5. 初始化大模型服务
	llmService := service.NewLLMService(
		cfg.LLM.APIKey,
		cfg.LLM.BaseURL,
		cfg.LLM.ModelName,
		cfg.LLM.MaxTokens,
		cfg.LLM.TimeoutSec,
	)

	// 6. 初始化业务服务
	historyService := service.NewHistoryService(historyRepo, assistantRepo, llmService)
	assistantService := service.NewAssistantService(assistantRepo, historyService)

	// 7. 初始化API处理器
	assistantHandler := handler.NewAssistantHandler(assistantService)
	historyHandler := handler.NewHistoryHandler(historyService)

	// 8. 初始化路由
	router := api.SetupRouter(assistantHandler, historyHandler)
	return router, cfg, nil
}
