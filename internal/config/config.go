package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	AppEnv   string
	AppHost  string
	HTTPPort string
	LogLevel string

	DB struct {
		Host     string
		Port     string
		User     string
		Password string
		Database string
		SSLMode  string
	}
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		AppHost:  getEnv("APP_HOST", "0.0.0.0"),
		HTTPPort: firstEnv("APP_PORT", "HTTP_PORT", "8091"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}
	cfg.DB.Host = getEnv("DB_HOST", "localhost")
	cfg.DB.Port = getEnv("DB_PORT", "5432")
	cfg.DB.User = getEnv("DB_USER", "postgres")
	cfg.DB.Password = getEnv("DB_PASSWORD", "postgres")
	cfg.DB.Database = getEnv("DB_DATABASE", "session_manager_service")
	cfg.DB.SSLMode = getEnv("DB_SSLMODE", "disable")
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.DB.Host == "" {
		return errors.New("config: DB_HOST is required")
	}
	if c.DB.User == "" {
		return errors.New("config: DB_USER is required")
	}
	if c.DB.Database == "" {
		return errors.New("config: DB_DATABASE is required")
	}
	if c.AppEnv == "production" && c.DB.Password == "" {
		return errors.New("config: in production DB_PASSWORD is required")
	}
	return nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Database, c.DB.SSLMode)
}

func (c *Config) DatabaseURL() string {
	pass := url.QueryEscape(c.DB.Password)
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DB.User, pass, c.DB.Host, c.DB.Port, c.DB.Database, c.DB.SSLMode)
}

func (c *Config) Addr() string {
	return c.AppHost + ":" + c.HTTPPort
}

func firstEnv(keysAndDef ...string) string {
	if len(keysAndDef) == 0 {
		return ""
	}
	def := keysAndDef[len(keysAndDef)-1]
	keys := keysAndDef[:len(keysAndDef)-1]
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return def
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
