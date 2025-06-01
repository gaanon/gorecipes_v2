package config

import (
	"fmt"
	"os"
	"strconv"
)

// DBConfig holds the database connection parameters.
// We'll expand this to load from environment variables or a file later.
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string // e.g., "disable", "require", "verify-full"
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// getEnvAsInt reads an environment variable as an integer or returns a default value.
func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return fallback
	}
	valueInt, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback
	}
	return valueInt
}

// DefaultDBConfig returns a database configuration, loading values from environment variables with fallbacks.
func DefaultDBConfig() DBConfig {
	return DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "your_db_user"),
		Password: getEnv("DB_PASSWORD", "your_db_password"),
		DBName:   getEnv("DB_NAME", "recipes_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// ConnectionString generates a PostgreSQL connection string from the DBConfig.
func (cfg DBConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
}
