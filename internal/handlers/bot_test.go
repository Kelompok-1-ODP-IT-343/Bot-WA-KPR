package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
)

type mockWhatsApp struct {
	lastPhone string
	lastMsg   string
}

func (m *mockWhatsApp) SendMessage(ctx context.Context, phone, message string) error {
	m.lastPhone = phone
	m.lastMsg = message
	return nil
}

func (m *mockWhatsApp) SendMessageWithAutoRevoke(ctx context.Context, phone, message string, after time.Duration) error {
	return nil
}

func (m *mockWhatsApp) IsConnected() bool { return true }

func TestSendReply_NoPrefixInjection(t *testing.T) {
	mw := &mockWhatsApp{}
	var qa domain.KPRQAService = nil
	h := &BotHandler{qa: qa, whatsapp: mw}

	ctx := context.Background()
	h.sendReply(ctx, "628123", "Halo! Aku Tanti, asisten virtual BNI.")
	if mw.lastMsg != "Halo! Aku Tanti, asisten virtual BNI." {
		t.Fatalf("unexpected message: %q", mw.lastMsg)
	}

	h.sendReply(ctx, "628123", "Pesan biasa tanpa intro")
	if mw.lastMsg != "Pesan biasa tanpa intro" {
		t.Fatalf("unexpected message: %q", mw.lastMsg)
	}
}
