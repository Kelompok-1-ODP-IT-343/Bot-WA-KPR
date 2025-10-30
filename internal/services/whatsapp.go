package services

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waTypes "go.mau.fi/whatsmeow/types"
	waEvents "go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite" // SQLite driver for whatsmeow store
)

type WhatsAppService struct {
	client *whatsmeow.Client
}

func NewWhatsAppService(storePath string) (*WhatsAppService, error) {
	container, err := sqlstore.New(context.Background(), "sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout=5000&_pragma=foreign_keys=on", storePath), waLog.Stdout("SQLStore", "INFO", true))
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlstore: %w", err)
	}

	device := container.NewDevice()
	client := whatsmeow.NewClient(device, waLog.Stdout("Client", "INFO", true))

	service := &WhatsAppService{client: client}

	// Connect
	if client.Store.ID == nil {
		// First login: print QR to pair
		qr, _ := client.GetQRChannel(context.Background())
		if err = client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		for evt := range qr {
			if evt.Event == "code" {
				log.Println("Scan QR di WhatsApp untuk pairing:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				log.Printf("QR event: %s", evt.Event)
			}
		}
	} else {
		if err = client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	return service, nil
}

func (w *WhatsAppService) SendMessage(ctx context.Context, phone, message string) error {
	to := waTypes.NewJID(phone, waTypes.DefaultUserServer)
	_, err := w.client.SendMessage(ctx, to, &waProto.Message{Conversation: &message})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (w *WhatsAppService) IsConnected() bool {
	return w.client.IsConnected()
}

func (w *WhatsAppService) AddEventHandler(handler func(interface{})) {
	w.client.AddEventHandler(handler)
}

func (w *WhatsAppService) Disconnect() {
	w.client.Disconnect()
}

func ExtractText(e *waEvents.Message) string {
	if e.Message.GetConversation() != "" {
		return e.Message.GetConversation()
	}
	if e.Message.ExtendedTextMessage != nil {
		return e.Message.ExtendedTextMessage.GetText()
	}
	return ""
}
