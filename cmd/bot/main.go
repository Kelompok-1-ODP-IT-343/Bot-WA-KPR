package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/config"
	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/handlers"
	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/services"
)

func main() {
	// Initialize configuration
	cfg := config.NewConfig()

	// Initialize database service
	dbService, err := services.NewDatabaseService(cfg.GetDatabaseURL())
	if err != nil {
		log.Fatalf("Failed to initialize database service: %v", err)
	}
	defer dbService.Close()

	if cfg.GetDatabaseURL() != "" {
		log.Println("Connected to PostgreSQL")
	} else {
		log.Println("DATABASE_URL not set, DB queries will be disabled")
	}

	// Initialize WhatsApp service
	whatsappService, err := services.NewWhatsAppService(cfg.GetWhatsAppStorePath())
	if err != nil {
		log.Fatalf("Failed to initialize WhatsApp service: %v", err)
	}

	log.Println("WhatsApp bot running")

	// Initialize AI Query service
	aiQueryService := services.NewAIQueryService(dbService, cfg.GetGeminiAPIKey())

	// Initialize handlers
	messageHandler := handlers.NewMessageHandler(whatsappService, cfg)
	botHandler := handlers.NewBotHandler(aiQueryService, whatsappService)

	// Setup WhatsApp event handler for listening to user chats
	whatsappService.AddEventHandler(botHandler.HandleMessage)

	// Setup REST API for sending messages
	if cfg.GetAPIKey() == "" {
		log.Println("WARNING: API_KEY is empty, REST endpoint will reject requests")
	}

	http.HandleFunc("/api/send-message", messageHandler.SendMessage)

	go func() {
		log.Printf("REST API listening on %s", cfg.GetHTTPAddr())
		if err := http.ListenAndServe(cfg.GetHTTPAddr(), nil); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	whatsappService.Disconnect()
	log.Println("Shutdown")
}
