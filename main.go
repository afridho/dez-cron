package main

import (
	"log"
	"os"

	"github.com/afridhozega/dez-cron/db"
	"github.com/afridhozega/dez-cron/handlers"
	"github.com/afridhozega/dez-cron/scheduler"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Set default log output to Stdout so Railway logs them as normal (not error)
	log.SetOutput(os.Stdout)

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; proceeding with system env vars")
	}

	// Connect to MongoDB
	db.ConnectDB()

	// Initialize Scheduler
	scheduler.Init()

	// Setup Router
	r := gin.Default()

	// Setup CORS
	r.Use(cors.Default())

	// Register API Routes
	handlers.RegisterRoutes(r)
	
	// Register Web UI Routes
	handlers.RegisterWebRoutes(r)

	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	r.Run(":" + port)
}
