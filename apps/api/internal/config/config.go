package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	GinMode        string
	AllowedOrigins []string

	DBEnabled   bool
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	AutoMigrate bool

	AuthLoginURL   string
	JWTSecret      string
	JWTExpiresHour int

	FileServiceBaseURL    string
	FileServiceUploadPath string
	FileServiceToken      string
	MaxUploadSizeMB       int
}

func Load() Config {
	if err := godotenv.Load(".env-production"); err != nil {
		log.Println(".env-production not found, trying .env")
		_ = godotenv.Load(".env")
	}

	return Config{
		Port:           getEnv("PORT", "8080"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		AllowedOrigins: splitCSV(getEnv("ALLOWED_ORIGINS", "")),

		DBEnabled:   getBoolEnv("DB_ENABLED", false),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "ty_notebook"),
		DBPassword:  getEnv("DB_PASSWORD", "change_me"),
		DBName:      getEnv("DB_NAME", "notebook_db"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
		AutoMigrate: getBoolEnv("AUTO_MIGRATE", false),

		AuthLoginURL:   getEnv("AUTH_LOGIN_URL", ""),
		JWTSecret:      getEnv("JWT_SECRET", "replace-with-secure-secret"),
		JWTExpiresHour: getIntEnv("JWT_EXPIRES_HOURS", 24),

		FileServiceBaseURL:    getEnv("FILE_SERVICE_BASE_URL", ""),
		FileServiceUploadPath: getEnv("FILE_SERVICE_UPLOAD_PATH", "/api/files/upload"),
		FileServiceToken:      getEnv("FILE_SERVICE_TOKEN", ""),
		MaxUploadSizeMB:       getIntEnv("MAX_UPLOAD_SIZE_MB", 10),
	}
}

func (c Config) AllowedOriginsOrDefault() []string {
	if len(c.AllowedOrigins) == 0 {
		return []string{"*"}
	}
	return c.AllowedOrigins
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getIntEnv(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
