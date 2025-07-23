package main

import (
	"Voice_Assistant/internal/config"
	"log"
	"net/http"
)

func main() {
	// 1. 初始化应用并获取路由和配置
	router, cfg, err := config.SetupApp()
	if err != nil {
		log.Fatalf("应用初始化失败: %v", err)
	}

	log.Fatal(http.ListenAndServe(cfg.Server.Port, router))
}
