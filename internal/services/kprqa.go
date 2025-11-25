package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
)

type KPRQAService struct {
	aiQuery    domain.AIQueryService
	promptPath string
}

func NewKPRQAService(aiQuery domain.AIQueryService, geminiKey, promptPath string) domain.KPRQAService {
	return &KPRQAService{
		aiQuery:    aiQuery,
		promptPath: promptPath,
	}
}

// Ask menggabungkan prompt dari file dengan input user.
// Jika text mengandung intent query ke tabel whitelist, akan dicoba eksekusi SELECT aman
// dan hasilnya dijadikan konteks untuk jawaban KPR.
func (s *KPRQAService) Ask(ctx context.Context, text string) (string, error) {
	log.Printf("[QA] Ask start len=%d", len(text))
	basePrompt := s.readPromptFile()
	if s.aiQuery == nil {
		return "", fmt.Errorf("AIQueryService not configured")
	}
	out, err := s.aiQuery.AnswerWithDB(ctx, text, basePrompt)
	if err != nil {
		log.Printf("[QA] Ask error: %v", err)
		return "", err
	}
	log.Printf("[QA] Ask ok len=%d", len(out))
	return out, nil
}

// AskForUser: gunakan AnswerWithDBForUser agar akses data digating berdasarkan nomor pengguna
func (s *KPRQAService) AskForUser(ctx context.Context, phone, text string) (string, error) {
	log.Printf("[QA] AskForUser start phone=%s len=%d", phone, len(text))
	basePrompt := s.readPromptFile()
	if s.aiQuery == nil {
		return "", fmt.Errorf("AIQueryService not configured")
	}
	out, err := s.aiQuery.AnswerWithDBForUser(ctx, phone, text, basePrompt)
	if err != nil {
		log.Printf("[QA] AskForUser error: %v", err)
		return "", err
	}
	log.Printf("[QA] AskForUser ok len=%d", len(out))
	return out, nil
}

func (s *KPRQAService) readPromptFile() string {
	b, err := os.ReadFile(s.promptPath)
	propmt := `Aku Tanti, asisten virtual BNI untuk KPR BNI Griya.
Jawab dengan bahasa Indonesia natural, ramah, dan to the point.
Hindari format kaku; berikan jawaban langsung.
Gunakan data sistem sebagai konteks secara aman jika tersedia; jangan tampilkan data sensitif.
Jika pengguna menyebut nomor aplikasi KPR atau sudah teridentifikasi lewat nomor WhatsApp, jawab langsung pakai data yang ada tanpa meminta informasi tambahan.
Untuk pertanyaan seperti jumlah pinjaman, angsuran bulanan, tenor, status, dan bunga: berikan angka/nilai spesifik dari [FAKTA] secara ringkas.
Jangan menolak akses data yang aman; fokus pada jawaban langsung berbasis [FAKTA].
Jangan tulis kalimat proses/menunggu seperti "sedang saya cek", "mohon tunggu", atau bentuk serupa; langsung berikan hasilnya.
Jawab hanya berdasarkan [FAKTA] dan [KONTEKS DATA] yang disediakan sistem. Jika [FAKTA] kosong untuk pertanyaan berbasis data, jawab singkat bahwa data tidak tersedia; jangan mengarang atau menyimpulkan.`
	if err != nil {
		return propmt
	}
	// Tambahkan DDL agar AI memahami skema dan kolom aman (relative ke promptPath)
	ddlPath := filepath.Join(filepath.Dir(s.promptPath), "ddl.sql")
	ddl := ""
	if dbb, derr := os.ReadFile(ddlPath); derr == nil {
		lines := strings.Split(string(dbb), "\n")
		if len(lines) > 245 {
			lines = lines[:245]
		}
		ddl = strings.Join(lines, "\n")
	}
	base := propmt + " Berikut adalah knowledge yang diberikan: " + string(b)
	if ddl != "" {
		base += "\n\nSkema basis data (DDL ringkas):\n" + ddl
		base += "\n\nContoh kebutuhan umum: status aplikasi, plafon pinjaman, angsuran bulanan, tenor, bunga, promo aktif. Gunakan kolom yang relevan dari kpr_applications, kpr_rates, approval_workflow."
	}
	return base
}
