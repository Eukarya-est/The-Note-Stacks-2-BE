package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	RedisHost     string
	RedisPort     string
	RedisPassword string
	ServerPort    string
	MarkdownDir   string // Path to markdown files directory for static serving

	// Elasticsearch configuration
	ElasticsearchEnabled  bool
	ElasticsearchURL      string
	ElasticsearchUsername string
	ElasticsearchPassword string
}

// LoadConfig loads configuration from environment variables with fallback defaults
// Returns a Config struct with all necessary application settings
func LoadConfig() *Config {
	esEnabled, _ := strconv.ParseBool(getEnv("ELASTICSEARCH_ENABLED", "false"))

	return &Config{
		RedisHost:     getEnv("REDIS_HOST", "redis"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getSecret("REDIS_PASSWORD"),
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		MarkdownDir:   getEnv("MARKDOWN_DIR", "../Markdown"), // Default path for local development

		// Elasticsearch settings
		ElasticsearchEnabled:  esEnabled,
		ElasticsearchURL:      getEnv("ELASTICSEARCH_URL", "http://note-stacks-elasticsearch:9200"),
		ElasticsearchUsername: getSecret("ELASTICSEARCH_USERNAME"),
		ElasticsearchPassword: getSecret("ELASTICSEARCH_PASSWORD"),
	}
}

// getEnv retrieves an environment variable or returns a default value if not set
// key: the environment variable name to look up
// defaultVal: the value to return if the environment variable is not set
func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

// getSecret retrieves a secret from file or environment variable
// Priority:
// 1. {KEY}_FILE environment variable (path to secret file)
// 2. {KEY} environment variable (direct value)
// 3. Empty string (no secret)
func getSecret(key string) string {
	// Try to load from file first (Docker Secrets pattern)
	filePathKey := key + "_FILE"
	if filePath := os.Getenv(filePathKey); filePath != "" {
		// #nosec G304 - Secret file paths are trusted from environment variables
		// Validate the file path to prevent directory traversal
		cleanPath := filepath.Clean(filePath)

		// Ensure the path is absolute or starts with /run/secrets (Docker Secrets)
		if !filepath.IsAbs(cleanPath) && !strings.HasPrefix(cleanPath, "/run/secrets") {
			log.Printf("Warning: Secret file path must be absolute or in /run/secrets: %s", filePath)
			return getEnv(key, "")
		}

		content, err := os.ReadFile(cleanPath)
		if err != nil {
			log.Printf("Warning: Could not read secret file %s: %v", cleanPath, err)
			// Fall back to environment variable
		} else {
			// Trim whitespace/newlines from file content
			return strings.TrimSpace(string(content))
		}
	}

	// Fall back to environment variable
	return getEnv(key, "")
}
