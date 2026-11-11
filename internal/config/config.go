package config

import (
	"os"
	"strings"
)

// Config holds application configuration
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	MercadoPago MercadoPagoConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string
	Host string
	// CORSAllowedOrigins lists the browser origins allowed to call the API
	// (CORS_ALLOWED_ORIGINS, comma-separated).
	CORSAllowedOrigins []string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	DBName   string
	User     string
	Password string
	SSLMode  string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey   string
	TokenExpiry string
}

// MercadoPagoConfig holds MercadoPago configuration
type MercadoPagoConfig struct {
	AccessToken   string
	PublicKey     string
	WebhookSecret string
	// Marketplace OAuth + payouts (placeholders; fill in when MP app is registered)
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// Load loads configuration from environment variables
func Load() Config {
	// JWT_SECRET is required — no fallback in production
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET environment variable is required")
	}

	return Config{
		Server: ServerConfig{
			Port:               getEnv("SERVER_PORT", "8080"),
			Host:               getEnv("SERVER_HOST", "0.0.0.0"),
			CORSAllowedOrigins: splitList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:4325,http://localhost:4322")),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			DBName:   getEnv("DB_NAME", "kinetix-db"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			SecretKey:   jwtSecret,
			TokenExpiry: getEnv("JWT_TOKEN_EXPIRY", "24h"),
		},
		MercadoPago: MercadoPagoConfig{
			AccessToken:   getEnv("MP_ACCESS_TOKEN", ""),
			PublicKey:     getEnv("MP_PUBLIC_KEY", ""),
			WebhookSecret: getEnv("MP_WEBHOOK_SECRET", ""),
			ClientID:      getEnv("MP_CLIENT_ID", ""),
			ClientSecret:  getEnv("MP_CLIENT_SECRET", ""),
			RedirectURI:   getEnv("MP_REDIRECT_URI", "http://localhost:8085/api/v1/admin/organizers/callback"),
		},
	}
}

// getEnv gets environment variable with fallback to default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// splitList splits a comma-separated value, trimming blanks and dropping empty items.
func splitList(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
