package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/config"
	"github.com/harkandavez/split-bill-go/internal/handler"
	"github.com/harkandavez/split-bill-go/internal/router"
)

func main() {
	cfg := config.Load()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// init db
	db := config.InitDB(cfg)

	r := gin.Default()
	r.Use(handler.ErrorMiddleware())
	router.RegisterRoutes(r, db, cfg)

	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}

	log.Printf("%s running on port %s", cfg.AppName, port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
