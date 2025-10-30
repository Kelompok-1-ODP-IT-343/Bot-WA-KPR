package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/services"
)

type OTPHandler struct {
	otpService      domain.OTPService
	whatsappService domain.WhatsAppService
	cfg             domain.ConfigService
}

func NewOTPHandler(otpService domain.OTPService, whatsappService domain.WhatsAppService, cfg domain.ConfigService) *OTPHandler {
	return &OTPHandler{
		otpService:      otpService,
		whatsappService: whatsappService,
		cfg:             cfg,
	}
}

// GenerateOTP handles POST /api/otp/generate
func (h *OTPHandler) GenerateOTP(w http.ResponseWriter, r *http.Request) {
	// Validate API key
	if !h.validateAPIKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.OTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate phone number
	if req.Phone == "" {
		http.Error(w, "Phone number is required", http.StatusBadRequest)
		return
	}

	// Generate OTP with custom expiry if provided
	var resp *domain.OTPResponse
	var err error

	if req.ExpiryTime > 0 {
		resp, err = h.otpService.GenerateOTP(r.Context(), req.Phone, req.ExpiryTime)
	} else {
		// Use default expiry from config (convert minutes to seconds)
		defaultExpiry := h.cfg.GetOTPExpiryMinutes() * 60
		resp, err = h.otpService.GenerateOTP(r.Context(), req.Phone, defaultExpiry)
	}

	if err != nil {
		log.Printf("Failed to generate OTP: %v", err)
		http.Error(w, "Failed to generate OTP", http.StatusInternalServerError)
		return
	}

	// Send OTP via WhatsApp
	message := fmt.Sprintf("🔐 Kode OTP Anda: *%s*\n\n⏰ Berlaku selama %d detik\n\n⚠️ Jangan bagikan kode ini kepada siapapun!",
		resp.Code, resp.ExpiresIn)

	if h.whatsappService.IsConnected() {
		if err := h.whatsappService.SendMessage(r.Context(), req.Phone, message); err != nil {
			log.Printf("Failed to send OTP via WhatsApp: %v", err)
			// Continue anyway, return the OTP response
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ValidateOTP handles POST /api/otp/validate
func (h *OTPHandler) ValidateOTP(w http.ResponseWriter, r *http.Request) {
	// Validate API key
	if !h.validateAPIKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.OTPValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Phone == "" || req.Code == "" {
		http.Error(w, "Phone and code are required", http.StatusBadRequest)
		return
	}

	// Validate OTP
	resp, err := h.otpService.ValidateOTP(r.Context(), req.Phone, req.Code)
	if err != nil {
		log.Printf("Failed to validate OTP: %v", err)
		http.Error(w, "Failed to validate OTP", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetOTPStatus handles GET /api/otp/status (for debugging)
func (h *OTPHandler) GetOTPStatus(w http.ResponseWriter, r *http.Request) {
	// Validate API key
	if !h.validateAPIKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get active OTPs count (if the service supports it)
	if service, ok := h.otpService.(*services.OTPService); ok {
		status := map[string]interface{}{
			"status":         "active",
			"active_otps":    service.GetActiveOTPs(),
			"expiry_minutes": h.cfg.GetOTPExpiryMinutes(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
		return
	}

	// Fallback response
	status := map[string]interface{}{
		"status":         "active",
		"expiry_minutes": h.cfg.GetOTPExpiryMinutes(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// validateAPIKey validates the API key from header or query parameter
func (h *OTPHandler) validateAPIKey(r *http.Request) bool {
	expectedKey := h.cfg.GetAPIKey()
	if expectedKey == "" {
		return true // No API key configured
	}

	// Check header first
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key == expectedKey
	}

	// Check query parameter
	if key := r.URL.Query().Get("api_key"); key != "" {
		return key == expectedKey
	}

	return false
}
