package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	DBDriver           string
	DBConnectionString string
	APIPort            int
	RBACMode           string
	CORSAllowedOrigins []string
	LogLevel           string
	ClusterRole        string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	c := &Config{
		DBDriver:           envOrDefault("DB_DRIVER", "sqlite"),
		DBConnectionString: envOrDefault("DB_CONNECTION_STRING", "assethub.db"),
		APIPort:            envOrDefaultInt("API_PORT", 8080),
		RBACMode:           envOrDefault("RBAC_MODE", "header"),
		LogLevel:           envOrDefault("LOG_LEVEL", "info"),
		ClusterRole:        envOrDefault("CLUSTER_ROLE", "development"),
	}

	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origins != "" {
		c.CORSAllowedOrigins = strings.Split(origins, ",")
		for i := range c.CORSAllowedOrigins {
			c.CORSAllowedOrigins[i] = strings.TrimSpace(c.CORSAllowedOrigins[i])
		}
	}

	return c
}

// AllowedStages returns the lifecycle stages visible for this cluster role.
func (c *Config) AllowedStages() []string {
	switch c.ClusterRole {
	case "production":
		return []string{"production"}
	case "testing":
		return []string{"testing", "production"}
	default: // development
		return []string{"development", "testing", "production"}
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
