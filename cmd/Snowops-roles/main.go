package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/MSTimX/Snowops-roles/internal/database"
	"github.com/MSTimX/Snowops-roles/internal/handlers"
	"github.com/MSTimX/Snowops-roles/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: failed to load .env file: %v", err)
	}

	database.Init()
	database.Migrate()
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "7070"
	}

	address := port
	if !strings.Contains(port, ":") {
		address = ":" + port
	}

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"*"},
		ExposeHeaders: []string{
			"Content-Type",
		},
		MaxAge: 12 * time.Hour,
	}))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	rolesGroup := router.Group("/roles")
	rolesGroup.Use(middleware.JWTAuthMiddleware())
	handlers.RegisterRoutes(rolesGroup)

	log.Printf("starting server on port %s", port)
	log.Println("App started")

	if err := router.Run(address); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
