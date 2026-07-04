package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type AuthConfig struct {
	AccessTokenTTL   int
	RefreshTokenTTL  int
	RateLimitPerIP   int
	RateLimitPerAcct int
	JWKSPath         string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/collabboard?sslmode=disable"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
		},
		Auth: AuthConfig{
			AccessTokenTTL:   getEnvInt("ACCESS_TOKEN_TTL", 900),
			RefreshTokenTTL:  getEnvInt("REFRESH_TOKEN_TTL", 604800),
			RateLimitPerIP:   getEnvInt("RATE_LIMIT_PER_IP", 100),
			RateLimitPerAcct: getEnvInt("RATE_LIMIT_PER_ACCT", 1000),
			JWKSPath:         getEnv("JWKS_PATH", "./jwks.json"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
