package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/services"
	waEvents "go.mau.fi/whatsmeow/types/events"
)

type BotHandler struct {
    qa       domain.KPRQAService
    whatsapp domain.WhatsAppService
}

func NewBotHandler(qa domain.KPRQAService, whatsapp domain.WhatsAppService) *BotHandler {
    return &BotHandler{
        qa:       qa,
        whatsapp: whatsapp,
    }
}

func (h *BotHandler) HandleMessage(evt interface{}) {
	switch e := evt.(type) {
	case *waEvents.Message:
		if e.Message.GetConversation() == "" && e.Message.ExtendedTextMessage == nil {
			return
		}

		// Abaikan pesan dari diri sendiri atau dari grup
		if e.Info.IsFromMe || e.Info.IsGroup {
			return
		}

		from := e.Info.MessageSource.Sender
		text := strings.TrimSpace(services.ExtractText(e))
		if text == "" {
			return
		}

		log.Printf("msg from %s: %s", from.String(), text)

		// Route message - hanya untuk AI query
		ctx := context.Background()
		// Kirim balasan ke pengirim (user) dengan format nomor saja
		h.handleQueryRequest(ctx, from.User, text)
	}
}

func (h *BotHandler) handleQueryRequest(ctx context.Context, phone, text string) {
    result, err := h.qa.Ask(ctx, text)
    if err != nil {
        h.sendReply(ctx, phone, fmt.Sprintf("AI error: %v", err))
        return
    }

    h.sendReply(ctx, phone, result)
}

func (h *BotHandler) sendReply(ctx context.Context, phone, message string) {
	if err := h.whatsapp.SendMessage(ctx, phone, message); err != nil {
		log.Printf("Failed to send reply: %v", err)
	}
}
