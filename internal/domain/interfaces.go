package domain

import (
	"context"
	"database/sql"
)

// WhatsAppService handles WhatsApp messaging operations
type WhatsAppService interface {
	SendMessage(ctx context.Context, phone, message string) error
	IsConnected() bool
}

// AIQueryService handles AI-powered database queries
type AIQueryService interface {
	PlanQuery(ctx context.Context, text string) (*SQLPlan, error)
	ExecuteQuery(ctx context.Context, plan *SQLPlan) (string, error)
}

// DatabaseService handles database operations
type DatabaseService interface {
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Close() error
}

// ConfigService handles application configuration
type ConfigService interface {
	GetDatabaseURL() string
	GetWhatsAppStorePath() string
	GetGeminiAPIKey() string
	GetAPIKey() string
	GetHTTPAddr() string
}
