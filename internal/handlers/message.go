package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
)

type MessageHandler struct {
	whatsappService domain.WhatsAppService
	config          domain.ConfigService
}

func NewMessageHandler(whatsappService domain.WhatsAppService, config domain.ConfigService) *MessageHandler {
	return &MessageHandler{
		whatsappService: whatsappService,
		config:          config,
	}
}

func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Validate API key
	key := r.Header.Get("X-API-Key")
	if key == "" {
		key = r.URL.Query().Get("api_key")
	}

	apiKey := h.config.GetAPIKey()
	if apiKey == "" || key != apiKey {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "unauthorized"})
		return
	}

	// Parse request
	var req domain.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid json"})
		return
	}

	req.Phone = strings.TrimSpace(req.Phone)
	req.Message = strings.TrimSpace(req.Message)
	if req.Phone == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "phone is required"})
		return
	}
	if req.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "message is required"})
		return
	}

	// Send message
	ctx := r.Context()
	if err := h.whatsappService.SendMessage(ctx, req.Phone, req.Message); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "failed to send message"})
		return
	}

	response := &domain.SendMessageResponse{
		Status: "sent",
		Phone:  req.Phone,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
