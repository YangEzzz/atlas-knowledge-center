package main

import (
	"log"

	"ty-notebook-api/internal/api/routes"
	"ty-notebook-api/internal/config"
	"ty-notebook-api/internal/database"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}

	if cfg.DBEnabled {
		database.Connect(cfg)
	} else {
		log.Println("Database connection skipped (DB_ENABLED=false)")
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOriginsOrDefault(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	routes.Register(r)

	log.Printf("Starting ty-notebook API on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
