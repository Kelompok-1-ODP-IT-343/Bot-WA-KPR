package config

import (
	"fmt"
	"os"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL       string
	WhatsAppStorePath string
	GeminiAPIKey      string
	APIKey            string
	HTTPAddr          string
}

func NewConfig() domain.ConfigService {
	// Load .env if present
	_ = godotenv.Load()

	storePath := os.Getenv("WHATSAPP_STORE_PATH")
	if storePath == "" {
		storePath = "whatsmeow.db"
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	return &Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		WhatsAppStorePath: storePath,
		GeminiAPIKey:      os.Getenv("GEMINI_API_KEY"),
		APIKey:            os.Getenv("API_KEY"),
		HTTPAddr:          httpAddr,
	}
}

func (c *Config) GetDatabaseURL() string {
	return c.DatabaseURL
}

func (c *Config) GetWhatsAppStorePath() string {
	return c.WhatsAppStorePath
}

func (c *Config) GetGeminiAPIKey() string {
	return c.GeminiAPIKey
}

func (c *Config) GetAPIKey() string {
	return c.APIKey
}

func (c *Config) GetHTTPAddr() string {
	return c.HTTPAddr
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY is required")
	}
	return nil
}
