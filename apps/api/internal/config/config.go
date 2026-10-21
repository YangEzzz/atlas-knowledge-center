package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AccountKeyID        string
	AccountKeys         string
	AccountRevealPolicy string
	Port                string
	GinMode             string
	AllowedOrigins      []string

	DBEnabled   bool
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	AutoMigrate bool

	FrontendURL        string
	CookieSecure       bool
	HeliosBaseURL      string
	HeliosInternalURL  string
	HeliosClientID     string
	HeliosClientSecret string
	HeliosRedirectURI  string
	HeliosProjectID    string
	HeliosAllowedRoles []string

	FileServiceBaseURL     string
	FileServiceUploadPath  string
	FileServiceToken       string
	MaxUploadSizeMB        int
	HeliosImageInternalURL string
	HeliosImagePublicURL   string
}

func Load() Config {
	if err := godotenv.Load(".env-production"); err != nil {
		log.Println(".env-production not found, trying .env")
		_ = godotenv.Load(".env")
	}

	cfg := Config{
		AccountKeyID:        getEnv("ACCOUNT_KEY_ID", ""),
		AccountKeys:         getEnv("ACCOUNT_KEYS", ""),
		AccountRevealPolicy: getEnv("ACCOUNT_REVEAL_POLICY", ""),
		Port:                getEnv("PORT", "8080"),
		GinMode:             getEnv("GIN_MODE", "debug"),
		AllowedOrigins:      splitCSV(getEnv("ALLOWED_ORIGINS", "")),

		DBEnabled:   getBoolEnv("DB_ENABLED", false),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "ty_notebook"),
		DBPassword:  getEnv("DB_PASSWORD", "change_me"),
		DBName:      getEnv("DB_NAME", "atlas_db"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
		AutoMigrate: getBoolEnv("AUTO_MIGRATE", false),

		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5176"),
		CookieSecure:       getBoolEnv("COOKIE_SECURE", true),
		HeliosBaseURL:      getEnv("HELIOS_BASE_URL", "https://helios-api.team-tool.top"),
		HeliosInternalURL:  getEnv("HELIOS_INTERNAL_URL", "http://127.0.0.1:3456"),
		HeliosClientID:     getEnv("HELIOS_CLIENT_ID", ""),
		HeliosClientSecret: getEnv("HELIOS_CLIENT_SECRET", ""),
		HeliosProjectID:    getEnv("HELIOS_PROJECT_ID", "atlas-knowledge-center"),
		HeliosAllowedRoles: splitCSV(getEnv("HELIOS_ALLOWED_ROLES", "admin,owner,productor,developer")),

		FileServiceBaseURL:    getEnv("FILE_SERVICE_BASE_URL", ""),
		FileServiceUploadPath: getEnv("FILE_SERVICE_UPLOAD_PATH", "/api/files/upload"),
		FileServiceToken:      getEnv("FILE_SERVICE_TOKEN", ""),
		MaxUploadSizeMB:       getIntEnv("MAX_UPLOAD_SIZE_MB", 10),
	}
	var err error
	cfg.HeliosImageInternalURL = getEnv("HELIOS_IMAGE_INTERNAL_URL", cfg.HeliosInternalURL)
	cfg.HeliosImagePublicURL = getEnv("HELIOS_IMAGE_PUBLIC_URL", cfg.HeliosBaseURL)
	cfg.HeliosRedirectURI, err = resolveSSORedirect(cfg.FrontendURL, getEnv("PUBLIC_API_URL", ""), getEnv("HELIOS_REDIRECT_URI", ""))
	if err != nil {
		log.Fatal(err)
	}
	for _, raw := range []string{cfg.FrontendURL, cfg.HeliosBaseURL, cfg.HeliosRedirectURI} {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || (u.Scheme != "https" && !(u.Scheme == "http" && loopback(u.Hostname()))) {
			log.Fatal("SSO URLs must use HTTPS, or HTTP on loopback for development")
		}
	}
	u, _ := url.Parse(cfg.FrontendURL)
	if !cfg.CookieSecure && !loopback(u.Hostname()) {
		log.Fatal("COOKIE_SECURE=false is only allowed on loopback")
	}
	return cfg
}

func resolveSSORedirect(frontendURL, publicAPIURL, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	base := frontendURL
	if publicAPIURL != "" {
		base = publicAPIURL
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(u.Scheme == "http" && loopback(u.Hostname()))) {
		return "", fmt.Errorf("SSO callback requires a trusted HTTPS origin or HTTP loopback")
	}
	u.Path = "/api/v1/auth/sso/callback"
	return u.String(), nil
}

func loopback(host string) bool { return host == "localhost" || host == "127.0.0.1" || host == "::1" }

func (c Config) AllowedOriginsOrDefault() []string {
	if len(c.AllowedOrigins) == 0 {
		return []string{c.FrontendURL}
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
