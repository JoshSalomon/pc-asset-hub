package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might be set
	for _, key := range []string{"DB_DRIVER", "DB_CONNECTION_STRING", "API_PORT", "RBAC_MODE", "CORS_ALLOWED_ORIGINS", "LOG_LEVEL"} {
		t.Setenv(key, "")
	}

	cfg := Load()

	assert.Equal(t, "sqlite", cfg.DBDriver)
	assert.Equal(t, "assethub.db", cfg.DBConnectionString)
	assert.Equal(t, 8080, cfg.APIPort)
	assert.Equal(t, "header", cfg.RBACMode)
	assert.Nil(t, cfg.CORSAllowedOrigins)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_CONNECTION_STRING", "host=localhost user=test dbname=test")
	t.Setenv("API_PORT", "9090")
	t.Setenv("RBAC_MODE", "k8s")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, http://localhost:5173")
	t.Setenv("LOG_LEVEL", "debug")

	cfg := Load()

	assert.Equal(t, "postgres", cfg.DBDriver)
	assert.Equal(t, "host=localhost user=test dbname=test", cfg.DBConnectionString)
	assert.Equal(t, 9090, cfg.APIPort)
	assert.Equal(t, "k8s", cfg.RBACMode)
	assert.Equal(t, []string{"http://localhost:3000", "http://localhost:5173"}, cfg.CORSAllowedOrigins)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestLoad_InvalidPortFallsBackToDefault(t *testing.T) {
	t.Setenv("API_PORT", "not-a-number")
	// Clear others
	t.Setenv("DB_DRIVER", "")
	t.Setenv("DB_CONNECTION_STRING", "")
	t.Setenv("RBAC_MODE", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("LOG_LEVEL", "")

	cfg := Load()

	assert.Equal(t, 8080, cfg.APIPort)
}

func TestLoad_SingleCORSOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")

	cfg := Load()

	assert.Equal(t, []string{"http://localhost:3000"}, cfg.CORSAllowedOrigins)
}

// T-CV.30: CLUSTER_ROLE env var defaults to "development"
func TestTCV30_ClusterRoleDefaults(t *testing.T) {
	t.Setenv("CLUSTER_ROLE", "")

	cfg := Load()
	assert.Equal(t, "development", cfg.ClusterRole)
}

// T-CV.31: AllowedStages returns correct stages for each clusterRole value
func TestTCV31_AllowedStages(t *testing.T) {
	tests := []struct {
		role     string
		expected []string
	}{
		{"development", []string{"development", "testing", "production"}},
		{"testing", []string{"testing", "production"}},
		{"production", []string{"production"}},
		{"", []string{"development", "testing", "production"}}, // default
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			cfg := &Config{ClusterRole: tt.role}
			if tt.role == "" {
				cfg.ClusterRole = "development"
			}
			assert.Equal(t, tt.expected, cfg.AllowedStages())
		})
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_KEY", "custom")
	assert.Equal(t, "custom", envOrDefault("TEST_KEY", "default"))

	t.Setenv("TEST_KEY", "")
	assert.Equal(t, "default", envOrDefault("TEST_KEY", "default"))
}

func TestEnvOrDefaultInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	assert.Equal(t, 42, envOrDefaultInt("TEST_INT", 0))

	t.Setenv("TEST_INT", "")
	assert.Equal(t, 99, envOrDefaultInt("TEST_INT", 99))

	t.Setenv("TEST_INT", "abc")
	assert.Equal(t, 99, envOrDefaultInt("TEST_INT", 99))
}
