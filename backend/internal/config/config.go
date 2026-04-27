package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName              string
	Env                  string
	HTTPAddr             string
	DatabaseURL          string
	RedisAddr            string
	RedisPassword        string
	JWTSecret            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	CORSOrigins          []string
	MigrationsPath       string
	AutoMigrate          bool
	SeedDemo             bool
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		AppName:              get("APP_NAME", "Whisper"),
		Env:                  get("APP_ENV", "development"),
		HTTPAddr:             get("HTTP_ADDR", ":8080"),
		DatabaseURL:          get("DATABASE_URL", "postgres://whisper:whisper@localhost:5432/whisper?sslmode=disable"),
		RedisAddr:            get("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        get("REDIS_PASSWORD", ""),
		JWTSecret:            get("JWT_SECRET", "change-me-in-production"),
		AccessTokenDuration:  duration("ACCESS_TOKEN_MINUTES", 15) * time.Minute,
		RefreshTokenDuration: duration("REFRESH_TOKEN_DAYS", 30) * 24 * time.Hour,
		CORSOrigins:          split("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"),
		MigrationsPath:       get("MIGRATIONS_PATH", "migrations"),
		AutoMigrate:          boolean("AUTO_MIGRATE", true),
		SeedDemo:             boolean("SEED_DEMO", true),
	}
}

func get(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func split(key, fallback string) []string {
	raw := get(key, fallback)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func duration(key string, fallback int) time.Duration {
	value, err := strconv.Atoi(get(key, strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		return time.Duration(fallback)
	}
	return time.Duration(value)
}

func boolean(key string, fallback bool) bool {
	value := strings.ToLower(get(key, strconv.FormatBool(fallback)))
	return value == "1" || value == "true" || value == "yes"
}
