# Bot WhatsApp KPR

Bot WhatsApp sederhana untuk sistem KPR dengan 2 service utama:

1. **Listen Chat**: Mendengarkan chat dari user untuk AI query ke database
2. **Send Message API**: Endpoint REST API untuk mengirim pesan WhatsApp

## Arsitektur

```
cmd/bot/                 # Application entry point
internal/
├── domain/             # Domain models dan interfaces
│   ├── models.go       # Core entities (SQLPlan, SendMessage)
│   └── interfaces.go   # Service interfaces
├── config/             # Configuration management
│   └── config.go       # Environment variables handler
├── services/           # Business logic layer
│   ├── database.go     # Database operations
│   ├── whatsapp.go     # WhatsApp client wrapper
│   └── aiquery.go      # AI-powered database queries
└── handlers/           # HTTP & message handlers
    ├── message.go      # REST API handlers
    └── bot.go          # WhatsApp message handlers
```

## Prerequisites

- Go 1.21+
- PostgreSQL (opsional, untuk AI query)
- WhatsApp Business Account
- Google Gemini API Key (opsional, untuk AI query)

## Configuration

Copy `.env.example` ke `.env`:

```bash
# Database (opsional)
DATABASE_URL=postgres://user:pass@localhost:5432/kpr_db

# WhatsApp
WHATSAPP_STORE_PATH=whatsmeow.db

# AI Service (opsional)
GEMINI_API_KEY=your_gemini_api_key

# REST API
API_KEY=your_secure_api_key
HTTP_ADDR=:8080

# Privasi AI (opsional)
# Jika diset ke "true", Gemini boleh menerima konteks data dari DB.
# Default: false (Gemini TIDAK menerima data mentah; data hanya dirender oleh sistem)
GEMINI_CAN_SEE_DATA=false
```

## Menjalankan Aplikasi

```bash
# Install dependencies
go mod tidy

# Build aplikasi
go build ./cmd/bot

# Jalankan
./bot
```

### Docker

```bash
# Start PostgreSQL (opsional)
docker-compose up -d

# Build dan run
docker build -t bot-wa-kpr .
docker run --env-file .env bot-wa-kpr
```

## Penggunaan

### 1. Listen Chat (AI Query)

Kirim pesan apa saja ke bot WhatsApp, bot akan:
- Menggunakan Gemini AI untuk memahami pertanyaan
- Membuat SQL query yang aman (hanya SELECT)
- Menjalankan query ke database
- Mengirim hasil kembali ke chat

Contoh:
- "Tampilkan semua users"
- "Cari aplikasi KPR yang pending"
- "Berapa banyak approval workflow yang aktif?"

### 2. Send Message API

```bash
POST /api/send-message
Headers: X-API-Key: your_api_key
Content-Type: application/json

{
  "phone": "6281234567890",
  "message": "Halo, ini pesan dari sistem KPR"
}
```

Response:
```json
{
  "status": "sent",
  "phone": "6281234567890"
}
```

## Security

- Hanya operasi SELECT yang diizinkan untuk AI query
- Whitelist tabel: `users`, `kpr_applications`, `approval_workflows`
- Prepared statements untuk mencegah SQL injection
- API key authentication untuk REST endpoints
- Privasi AI: saat `GEMINI_CAN_SEE_DATA=false`, data hasil DB TIDAK dikirim ke AI. Jawaban AI dibuat tanpa melihat data mentah, dan ringkasan data (dengan masking) dirender oleh sistem secara terpisah.

## Testing

```bash
# Run unit tests
go test ./internal/...

# Run dengan coverage
go test -cover ./internal/...
```