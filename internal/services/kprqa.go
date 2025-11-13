package services

import (
    "context"
    "fmt"
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
    basePrompt := s.readPromptFile()
    if s.aiQuery == nil {
        return "", fmt.Errorf("AIQueryService not configured")
    }
    return s.aiQuery.AnswerWithDB(ctx, text, basePrompt)
}

func (s *KPRQAService) readPromptFile() string {
    b, err := os.ReadFile(s.promptPath)
    if err != nil {
        return "Anda adalah asisten KPR. Jawab pertanyaan seputar KPR dengan bahasa yang jelas, singkat, dan aman. Jika tersedia data dari database, gunakan sebagai konteks tambahan. Hindari saran finansial spesifik; arahkan ke petugas bank bila diperlukan."
    }
    return string(b)
}