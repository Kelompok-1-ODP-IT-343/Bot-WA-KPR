package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
)

// OTPEntry represents an OTP entry with expiry
type OTPEntry struct {
	Code      string
	Phone     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// OTPService implements domain.OTPService with in-memory storage and auto-expiry
type OTPService struct {
	otps          map[string]*OTPEntry // key: phone number
	mutex         sync.RWMutex
	defaultExpiry int // in seconds
}

// NewOTPService creates a new OTP service with default expiry time
func NewOTPService(defaultExpirySeconds int) *OTPService {
	service := &OTPService{
		otps:          make(map[string]*OTPEntry),
		defaultExpiry: defaultExpirySeconds,
	}

	// Start cleanup goroutine
	go service.startCleanupRoutine()

	return service
}

// GenerateOTP generates a new OTP for the given phone number
func (s *OTPService) GenerateOTP(ctx context.Context, phone string, expirySeconds ...int) (*domain.OTPResponse, error) {
	// Determine expiry time
	expiry := s.defaultExpiry
	if len(expirySeconds) > 0 && expirySeconds[0] > 0 {
		expiry = expirySeconds[0]
	}

	// Generate 6-digit OTP
	code, err := s.generateRandomCode(6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP code: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(expiry) * time.Second)

	// Store OTP
	s.mutex.Lock()
	s.otps[phone] = &OTPEntry{
		Code:      code,
		Phone:     phone,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	s.mutex.Unlock()

	return &domain.OTPResponse{
		Status:    "success",
		Phone:     phone,
		Code:      code,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		ExpiresIn: expiry,
	}, nil
}

// ValidateOTP validates the provided OTP code for the given phone number
func (s *OTPService) ValidateOTP(ctx context.Context, phone, code string) (*domain.OTPValidateResponse, error) {
	s.mutex.RLock()
	entry, exists := s.otps[phone]
	s.mutex.RUnlock()

	if !exists {
		return &domain.OTPValidateResponse{
			Status:  "error",
			Phone:   phone,
			Valid:   false,
			Message: "No OTP found for this phone number",
		}, nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired OTP
		s.mutex.Lock()
		delete(s.otps, phone)
		s.mutex.Unlock()

		return &domain.OTPValidateResponse{
			Status:  "error",
			Phone:   phone,
			Valid:   false,
			Message: "OTP has expired",
		}, nil
	}

	// Validate code
	if entry.Code != code {
		return &domain.OTPValidateResponse{
			Status:  "error",
			Phone:   phone,
			Valid:   false,
			Message: "Invalid OTP code",
		}, nil
	}

	// Valid OTP - remove it (one-time use)
	s.mutex.Lock()
	delete(s.otps, phone)
	s.mutex.Unlock()

	return &domain.OTPValidateResponse{
		Status:  "success",
		Phone:   phone,
		Valid:   true,
		Message: "OTP validated successfully",
	}, nil
}

// CleanupExpiredOTPs removes all expired OTPs from memory
func (s *OTPService) CleanupExpiredOTPs(ctx context.Context) error {
	now := time.Now()
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for phone, entry := range s.otps {
		if now.After(entry.ExpiresAt) {
			delete(s.otps, phone)
		}
	}

	return nil
}

// generateRandomCode generates a random numeric code of specified length
func (s *OTPService) generateRandomCode(length int) (string, error) {
	const digits = "0123456789"
	code := make([]byte, length)

	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}

	return string(code), nil
}

// startCleanupRoutine starts a background goroutine to cleanup expired OTPs
func (s *OTPService) startCleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second) // Cleanup every 30 seconds
	defer ticker.Stop()

	for range ticker.C {
		s.CleanupExpiredOTPs(context.Background())
	}
}

// GetActiveOTPs returns the count of active (non-expired) OTPs - for debugging
func (s *OTPService) GetActiveOTPs() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	now := time.Now()
	count := 0
	for _, entry := range s.otps {
		if now.Before(entry.ExpiresAt) {
			count++
		}
	}

	return count
}
