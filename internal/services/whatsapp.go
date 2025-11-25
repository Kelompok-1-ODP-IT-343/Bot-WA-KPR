package services

import (
	"context"
	"fmt"
	"log"
	"math/rand"
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
	_ "modernc.org/sqlite"
)

type WhatsAppService struct {
	client *whatsmeow.Client
}

// stripDevicePart menghapus suffix perangkat pada user JID (contoh: 628xx:12 -> 628xx)
func stripDevicePart(user string) string {
	if idx := strings.Index(user, ":"); idx != -1 {
		return user[:idx]
	}
	return user
}

// normalizePhone membersihkan input agar hanya berisi digit nomor WA
// - menghapus bagian setelah '@' jika ada (JID penuh)
// - menghapus suffix perangkat (":device")
// - menghapus karakter non-digit (spasi, tanda plus, dash)
func normalizePhone(s string) string {
	s = strings.TrimSpace(s)
	// buang domain JID jika ada
	if at := strings.Index(s, "@"); at != -1 {
		s = s[:at]
	}
	// buang suffix device
	s = stripDevicePart(s)
	// buang plus
	s = strings.ReplaceAll(s, "+", "")
	// keep hanya digit
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func NewWhatsAppService(storePath string) (*WhatsAppService, error) {
	log.Printf("Initializing WhatsApp service with store path: %s", storePath)
	// rand.Seed is deprecated as of Go 1.20; global generator auto-seeds

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
			log.Println("[WA] Event Connected")
		case *waEvents.Disconnected:
			log.Printf("[WA] Event Disconnected: %+v", v)
		case *waEvents.LoggedOut:
			log.Println("[WA] Event LoggedOut")
		case *waEvents.ConnectFailure:
			log.Printf("[WA] Event ConnectFailure: reason=%d loggedOut=%v", v.Reason, v.Reason.IsLoggedOut())
		case *waEvents.KeepAliveTimeout:
			log.Printf("[WA] Event KeepAliveTimeout: errorCount=%d lastSuccess=%s", v.ErrorCount, v.LastSuccess)
		case *waEvents.KeepAliveRestored:
			log.Printf("[WA] Event KeepAliveRestored")
		case *waEvents.Presence:
			log.Printf("[WA] Event Presence: from=%s unavailable=%v lastSeen=%s", v.From, v.Unavailable, v.LastSeen)
		case *waEvents.ChatPresence:
			log.Printf("[WA] Event ChatPresence: chat=%s state=%s media=%s", v.MessageSource.SourceString(), v.State, v.Media)
		case *waEvents.Receipt:
			log.Printf("[WA] Event Receipt: %+v", v)
		case *waEvents.Message:
			log.Printf("[WA] Event Message: %+v", v)
		default:
			log.Printf("[WA] Event %T", v)
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

	phoneNorm := normalizePhone(phone)
	if phoneNorm == "" {
		return fmt.Errorf("invalid phone input")
	}
	if phoneNorm != phone {
		log.Printf("[WA] Normalized phone '%s' -> '%s'", phone, phoneNorm)
	}
	to := waTypes.NewJID(phoneNorm, waTypes.DefaultUserServer)
	msg := &waProto.Message{Conversation: &message}

	delay := time.Second + time.Duration(rand.Intn(1000))*time.Millisecond
	if err := w.client.SendPresence(ctx, waTypes.PresenceAvailable); err != nil {
		log.Printf("[WA] SendPresence error: %v", err)
	}
	if err := w.client.SendChatPresence(ctx, to, waTypes.ChatPresenceComposing, waTypes.ChatPresenceMediaText); err != nil {
		log.Printf("[WA] SendChatPresence error: %v", err)
	}
	log.Printf("[WA] Humanized delay before send: %s len=%d", delay, len(message))
	time.Sleep(delay)

	// Retry mechanism for encryption issues
	var resp whatsmeow.SendResponse
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		log.Printf("[WA] Sending attempt %d/%d to %s", i+1, maxRetries, phone)
		resp, err = w.client.SendMessage(ctx, to, msg)
		if err == nil {
			log.Printf("[WA] Attempt %d success id=%s", i+1, resp.ID)
			break
		}

		// Check if it's an encryption error
		if strings.Contains(fmt.Sprintf("%v", err), "can't encrypt message") ||
			strings.Contains(fmt.Sprintf("%v", err), "no signal session established") {
			log.Printf("[WA] Encryption error (attempt %d/%d): %v", i+1, maxRetries, err)

			if i < maxRetries-1 {
				// Wait before retry
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				log.Printf("[WA] Retrying message send to %s after backoff...", phone)
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

	return nil
}

// (Auto revoke dihapus)

func (w *WhatsAppService) IsConnected() bool {
	return w.client.IsConnected()
}

// Reconnect mencoba menyambungkan kembali client jika terputus
func (w *WhatsAppService) Reconnect(ctx context.Context) error {
	if err := w.client.Connect(); err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}
	return nil
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
