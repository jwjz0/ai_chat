package main

import (
	"Voice_Assistant/internal/config"
	"log"
	"net/http"
)

func main() {
	router := config.SetupApp()

	log.Fatal(http.ListenAndServe(config.AppConfig.Server.Port, router))
}
