package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL       string
	WhatsAppStorePath string
	GeminiAPIKey      string
	APIKey            string
	HTTPAddr          string
	OTPExpiryMinutes  int
	KPRPromptPath     string
	GeminiCanSeeData  bool
	SQLAuditPath      string
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

	otpExpiryMinutes := 5 // default 5 minutes
	if otpEnv := os.Getenv("OTP_EXPIRY_MINUTES"); otpEnv != "" {
		if parsed, err := strconv.Atoi(otpEnv); err == nil && parsed > 0 {
			otpExpiryMinutes = parsed
		}
	}

	promptPath := os.Getenv("KPR_PROMPT_PATH")
	if promptPath == "" {
		promptPath = "kpr_prompt.txt"
	}

	// Privacy flag: control whether Gemini can see DB data strings in prompt
	// Default: false (Gemini should NOT see raw data strings)
	geminiCanSeeData := false
	if v := os.Getenv("GEMINI_CAN_SEE_DATA"); v != "" {
		// Accept common truthy values: true/1/yes/on
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "on":
			geminiCanSeeData = true
		}
	}

	auditPath := os.Getenv("SQL_AUDIT_PATH")
	if strings.TrimSpace(auditPath) == "" {
		auditPath = "sql_audit.jsonl"
	}

	return &Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		WhatsAppStorePath: storePath,
		GeminiAPIKey:      os.Getenv("GEMINI_API_KEY"),
		APIKey:            os.Getenv("API_KEY"),
		HTTPAddr:          httpAddr,
		OTPExpiryMinutes:  otpExpiryMinutes,
		KPRPromptPath:     promptPath,
		GeminiCanSeeData:  geminiCanSeeData,
		SQLAuditPath:      auditPath,
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

func (c *Config) GetOTPExpiryMinutes() int {
	return c.OTPExpiryMinutes
}

func (c *Config) GetKPRPromptPath() string {
	return c.KPRPromptPath
}

func (c *Config) GetGeminiCanSeeData() bool {
	return c.GeminiCanSeeData
}

func (c *Config) GetSQLAuditPath() string {
	return c.SQLAuditPath
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY is required")
	}
	return nil
}
