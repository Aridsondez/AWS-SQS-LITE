package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all environment configuration
type Config struct {
	Port                int
	DatabaseURL         string
	VisibilityTimeout   time.Duration
	ReceiveMax          int
	SweepInterval       time.Duration
	LogLevel            string
	DBConnectionTimeout time.Duration
}

// helper: read env var as int seconds → convert to duration
func getEnvAsDuration(name string, defaultVal time.Duration) time.Duration {
	if value, exists := os.LookupEnv(name); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return time.Duration(i) * time.Second
		}
	}
	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	if value, exists := os.LookupEnv(name); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultVal
}

func getEnv(name, defaultVal string) string {
	if value, exists := os.LookupEnv(name); exists {
		return value
	}
	return defaultVal
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:                getEnvAsInt("PORT", 8080),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		VisibilityTimeout:   getEnvAsDuration("VISIBILITY_TIMEOUT", 30*time.Second),
		ReceiveMax:          getEnvAsInt("RECEIVE_MAX", 10),
		SweepInterval:       getEnvAsDuration("SWEEP_INTERVAL", 60*time.Second),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		DBConnectionTimeout: getEnvAsDuration("DB_CONNECTION_TIMEOUT", 5*time.Second),
	}

	// Basic validation
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid PORT: %d", cfg.Port)
	}
	if cfg.ReceiveMax <= 0 {
		return nil, fmt.Errorf("invalid RECEIVE_MAX: %d", cfg.ReceiveMax)
	}

	return cfg, nil
}
