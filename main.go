package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"note-stacks-backend/config"
	"note-stacks-backend/handlers"
	"note-stacks-backend/middleware"
	"note-stacks-backend/repository"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load configuration from environment variables
	cfg := config.LoadConfig()

	// Initialize Redis client
	redisClient := initRedis(cfg)
	defer redisClient.Close()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// Initialize repository
	repo := repository.NewRedisRepository(redisClient)

	// Initialize handlers
	handler := handlers.NewHandler(repo)

	// Setup Gin router
	router := setupRouter(handler, cfg)

	// Start server
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// initRedis creates and configures a Redis client
// cfg: Configuration containing Redis connection details
// Returns a configured Redis client instance
func initRedis(cfg *config.Config) *redis.Client {
	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.RedisPassword,
		DB:           0, // Use default DB
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10, // Connection pool size
		MinIdleConns: 5,  // Minimum idle connections
	})

	return client
}

// setupRouter configures the Gin router with all routes and middleware
// handler: Handler for all endpoints
// cfg: Configuration including markdown directory path
// Returns a configured Gin engine instance
func setupRouter(handler *handlers.Handler, cfg *config.Config) *gin.Engine {
	// Set Gin to release mode for production
	// gin.SetMode(gin.ReleaseMode) // Uncomment for production

	router := gin.Default()

	// Setup CORS middleware
	router.Use(middleware.SetupCORS())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Serve static markdown files and images
	// This allows the frontend to load images from the markdown directory
	markdownDir := cfg.MarkdownDir
	if markdownDir == "" {
		// Default to relative path if not configured
		markdownDir = filepath.Join(".", "markdown-files")
	}

	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Check if markdown directory exists
	if _, err := os.Stat(markdownDir); err == nil {
		log.Printf("Serving static files from: %s", markdownDir)

		// Serve files at /static/* route
		// Example: /static/Linux/imgs/diagram.png -> D:\markdown - Copy\Linux\imgs\diagram.png
		router.Static("/static", markdownDir)

		log.Printf("Static file server enabled at /static")
	} else {
		log.Printf("Warning: Markdown directory not found at %s. Static file serving disabled.", markdownDir)
		log.Printf("Set MARKDOWN_DIR environment variable to enable image serving")
	}

	// API v1 routes
	api := router.Group("/api")
	{
		// Note routes
		api.POST("/notes", handler.CreateNote)
		api.GET("/notes", handler.GetAllNotes)
		api.GET("/notes/search", handler.SearchNotes)
		api.GET("/notes/:id", handler.GetNote)
		api.PUT("/notes/:id", handler.UpdateNote)
		api.DELETE("/notes/:id", handler.DeleteNote)
		api.POST("/views", handler.TrackView)
		api.GET("/notes/:id/views/count", handler.GetViewCount)

		// Cover routes (categories)
		api.POST("/covers", handler.CreateCover)
		api.GET("/covers", handler.GetAllCovers)
		api.GET("/covers/:id", handler.GetCover)
		api.GET("/covers/:id/notes", handler.GetNotesByCover)
		api.PUT("/covers/:id", handler.UpdateCover)
		api.DELETE("/covers/:id", handler.DeleteCover)

		// Calendar Event routes
		api.POST("/events", handler.CreateCalendarEvent)
		api.GET("/events", handler.GetAllCalendarEvents)
		api.GET("/events/:id", handler.GetCalendarEvent)
		api.PUT("/events/:id", handler.UpdateCalendarEvent)
		api.DELETE("/events/:id", handler.DeleteCalendarEvent)

	}

	return router
}
