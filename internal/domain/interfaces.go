package domain

import (
	"context"
	"database/sql"
	"time"
)

// WhatsAppService handles WhatsApp messaging operations
type WhatsAppService interface {
	SendMessage(ctx context.Context, phone, message string) error
	// SendMessageWithAutoRevoke mengirim pesan dan otomatis unsend setelah durasi tertentu
	SendMessageWithAutoRevoke(ctx context.Context, phone, message string, after time.Duration) error
	IsConnected() bool
}

// AIQueryService handles AI-powered database queries
type AIQueryService interface {
    PlanQuery(ctx context.Context, text string) (*SQLPlan, error)
    ExecuteQuery(ctx context.Context, plan *SQLPlan) (string, error)
    // AnswerWithDB: generate SQL plan, execute, feed DB data back into AI with basePrompt, return final answer
    AnswerWithDB(ctx context.Context, text string, basePrompt string) (string, error)
    // AnswerWithDBForUser: sama seperti AnswerWithDB, tetapi terlebih dahulu mengambil konteks user berdasarkan phone,
    // lalu mempersonalisasi jawaban dan memaksa query dibatasi pada user terkait bila tabel mendukung
    AnswerWithDBForUser(ctx context.Context, userPhone string, text string, basePrompt string) (string, error)
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
	GetOTPExpiryMinutes() int
	GetKPRPromptPath() string
}

// OTPService handles OTP generation, validation, and expiry
type OTPService interface {
	GenerateOTP(ctx context.Context, phone string, expirySeconds ...int) (*OTPResponse, error)
	ValidateOTP(ctx context.Context, phone, code string) (*OTPValidateResponse, error)
	CleanupExpiredOTPs(ctx context.Context) error
}

// KPRQAService handles KPR Q&A with optional DB context
type KPRQAService interface {
    Ask(ctx context.Context, text string) (string, error)
    // AskForUser: sama seperti Ask, tetapi menyertakan nomor pengguna untuk gating akses data
    AskForUser(ctx context.Context, phone, text string) (string, error)
}
