package services

import (
	"context"
	"fmt"
	"log"
	"os"

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
Hindari format kaku seperti daftar "Intisari/Data/Langkah/Disclaimer"; berikan jawaban langsung.
Gunakan data sistem sebagai konteks secara aman jika tersedia; jangan tampilkan data sensitif.
Jangan berikan saran finansial personal atau rekomendasi keputusan spesifik.
Jika butuh bantuan lanjutan, arahkan secara ringkas ke petugas atau cabang BNI terdekat.`
	if err != nil {
		return propmt
	}
	return propmt + " Berikut adalah knowledge yang diberikan: " + string(b)
}
