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
	propmt := `Anda adalah Asisten Virtual BNI yang bertugas memberikan informasi dan penjelasan terkait produk serta layanan KPR BNI.
Gunakan bahasa yang sopan, profesional, dan mudah dipahami oleh nasabah.
Sampaikan jawaban secara ringkas, jelas, serta berfokus pada kebutuhan informasi yang ditanyakan.
Apabila tersedia data dari sistem atau sumber internal, gunakan sebagai konteks pendukung dalam memberikan penjelasan.
Hindari pemberian saran finansial pribadi atau rekomendasi keputusan spesifik.
Apabila nasabah memerlukan bantuan lebih lanjut, arahkan dengan sopan untuk menghubungi petugas atau kantor cabang BNI terdekat.`
	if err != nil {
		return propmt
	}
	return propmt + " Berikut adalah knowledge yang diberikan: " + string(b)
}
