package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

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
	log.Printf("Initializing WhatsApp service with store path: %s", storePath)

	container, err := sqlstore.New(context.Background(), "sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout=5000&_pragma=foreign_keys=on", storePath), waLog.Stdout("SQLStore", "INFO", true))
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlstore: %w", err)
	}

	// Get the first device from the store, or create a new one if none exists
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Printf("No existing device found, creating new device: %v", err)
		deviceStore = container.NewDevice()
	} else {
		log.Printf("Found existing device with ID: %s", deviceStore.ID)
	}

	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))
	service := &WhatsAppService{client: client}

	// Add event handler to monitor connection status
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *waEvents.Connected:
			log.Println("WhatsApp client connected successfully")
		case *waEvents.Disconnected:
			log.Printf("WhatsApp client disconnected: %v", v)
		case *waEvents.LoggedOut:
			log.Println("WhatsApp client logged out")
		}
	})

	// Check if we have a valid session
	if client.Store.ID == nil {
		log.Println("No session found, starting QR code pairing...")
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
		log.Printf("Existing session found for device ID: %s", client.Store.ID)
		if err = client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect with existing session: %w", err)
		}

		// Wait a bit for the connection to stabilize
		log.Println("Waiting for connection to stabilize...")
		time.Sleep(3 * time.Second)

		if client.IsConnected() {
			log.Println("Successfully connected using existing session!")
		} else {
			log.Println("Warning: Client may not be fully connected yet")
		}
	}

	return service, nil
}

func (w *WhatsAppService) SendMessage(ctx context.Context, phone, message string) error {
	// Check if client is connected
	if !w.client.IsConnected() {
		return fmt.Errorf("WhatsApp client is not connected")
	}

	to := waTypes.NewJID(phone, waTypes.DefaultUserServer)
	msg := &waProto.Message{Conversation: &message}

	// Retry mechanism for encryption issues
	var resp whatsmeow.SendResponse
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		resp, err = w.client.SendMessage(ctx, to, msg)
		if err == nil {
			break
		}

		// Check if it's an encryption error
		if strings.Contains(fmt.Sprintf("%v", err), "can't encrypt message") ||
			strings.Contains(fmt.Sprintf("%v", err), "no signal session established") {
			log.Printf("[WA] Encryption error (attempt %d/%d): %v", i+1, maxRetries, err)

			if i < maxRetries-1 {
				// Wait before retry
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				log.Printf("[WA] Retrying message send to %s...", phone)
				continue
			}
		}

		// For other errors, don't retry
		break
	}

	if err != nil {
		return fmt.Errorf("failed to send message after %d attempts: %w", maxRetries, err)
	}

	log.Printf("[WA] ✅ Sent message ID: %s to %s", resp.ID, phone)

	// Auto revoke/unsend after 1 minute
	go func(messageID string, jid waTypes.JID) {
		log.Printf("[WA] Scheduling auto-revoke for message %s after 1 minute", messageID)
		select {
		case <-time.After(1 * time.Minute):
			log.Printf("[WA] Auto-revoking message %s after 1 minute", messageID)
			// Use whatsmeow's built-in BuildRevoke method
			revoke := w.client.BuildRevoke(jid, w.client.Store.ID.ToNonAD(), messageID)

			_, err := w.client.SendMessage(context.Background(), jid, revoke)
			if err != nil {
				log.Printf("[WA] Failed to revoke message %s: %v", messageID, err)
			} else {
				log.Printf("[WA] ✅ Auto-revoked message %s after 1 minute", messageID)
			}
		case <-ctx.Done():
			log.Printf("[WA] Auto-revoke context canceled for message %s", messageID)
			return
		}
	}(resp.ID, to)

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
