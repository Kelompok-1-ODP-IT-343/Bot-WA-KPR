package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
	ai "github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var allowedTables = map[string]struct{}{
	// Sesuai dengan ddl.sql
	"users":             {},
	"roles":             {},
	"branch_staff":      {},
	"user_profiles":     {},
	"kpr_rates":         {},
	"kpr_applications":  {},
	"approval_workflow": {}, // nama tabel di DDL adalah singular
	// direferensikan oleh kpr_applications (FK)
	"properties": {},
}

// init mencoba menyelaraskan allowedTables dengan tabel pada ddl.sql bila tersedia
func init() {
	refreshAllowedTablesFromDDL("ddl.sql")
}

// refreshAllowedTablesFromDDL membaca file DDL dan memperbarui daftar allowedTables secara dinamis
func refreshAllowedTablesFromDDL(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		// abaikan jika tidak ada; gunakan daftar default
		fmt.Println("Warning: Tidak dapat membaca DDL untuk memperbarui tabel yang diizinkan:", err)
		return
	}
	names := parseDDLForTables(string(data))
	if len(names) == 0 {
		return
	}
	// Reset dan isi ulang
	nt := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.TrimSpace(strings.ToLower(n))
		if n == "" {
			continue
		}
		nt[n] = struct{}{}
	}
	// Pastikan tidak kosong; baru ganti
	if len(nt) > 0 {
		allowedTables = nt
	}
}

// parseDDLForTables mengekstrak nama tabel dari statement CREATE TABLE
func parseDDLForTables(ddl string) []string {
	out := []string{}
	lower := strings.ToLower(ddl)
	idx := 0
	for {
		j := strings.Index(lower[idx:], "create table")
		if j == -1 {
			break
		}
		// posisi absolut
		start := idx + j + len("create table")
		// ambil substring setelah kata kunci untuk mencari nama tabel
		rest := ddl[start:]
		restTrim := strings.TrimSpace(rest)
		// nama tabel sampai spasi atau tanda '('
		end := len(restTrim)
		if p := strings.IndexAny(restTrim, " (\n\r\t"); p != -1 {
			end = p
		}
		name := restTrim[:end]
		name = strings.TrimSpace(name)
		// hilangkan schema qualifier public.
		if strings.HasPrefix(strings.ToLower(name), "public.") {
			name = name[len("public."):]
		}
		// hilangkan tanda kutip ganda jika ada
		name = strings.Trim(name, "\"")
		if name != "" {
			out = append(out, name)
		}
		// maju indeks
		idx = start
	}
	// unik dan stabil
	seen := map[string]struct{}{}
	uniq := []string{}
	for _, n := range out {
		k := strings.ToLower(strings.TrimSpace(n))
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		uniq = append(uniq, k)
	}
	sort.Strings(uniq)
	return uniq
}

// allowedTablesList mengembalikan daftar tabel yang diizinkan dalam bentuk string koma-terpisah
func allowedTablesList() string {
	names := make([]string, 0, len(allowedTables))
	for k := range allowedTables {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

type AIQueryService struct {
	db        domain.DatabaseService
	geminiKey string
}

func NewAIQueryService(db domain.DatabaseService, geminiKey string) domain.AIQueryService {
	return &AIQueryService{
		db:        db,
		geminiKey: geminiKey,
	}
}

func (a *AIQueryService) PlanQuery(ctx context.Context, text string) (*domain.SQLPlan, error) {
	if a.geminiKey == "" {
		// Fallback: naive parser
		return a.naivePlan(text)
	}

	client, err := ai.NewClient(ctx, option.WithAPIKey(a.geminiKey))
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.0-flash")
	// Prompt disesuaikan dengan skema di ddl.sql (nama tabel persis)
	prompt := "Anda adalah perencana SQL AMAN untuk PostgreSQL. Kembalikan hanya JSON dengan field: " +
		"operation (hanya 'SELECT'), " +
		"table (salah satu dari: " + allowedTablesList() + "), " +
		"columns (opsional array nama kolom), " +
		"filters (opsional array dari objek {column, op='=', value}), " +
		"limit (opsional int; default 20). " +
		"Aturan keras: (1) HANYA tabel whitelist di atas; gunakan nama persis sesuai DDL. " +
		"(2) JANGAN gunakan JOIN, subquery, agregasi, ORDER BY, atau GROUP BY. " +
		"(3) Filters hanya boleh memakai operator '='. " +
		"(4) Jika columns/filters tidak disebutkan, kembalikan field tersebut kosong. " +
		"Teks: " + text

	resp, err := model.GenerateContent(ctx, ai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini: %w", err)
	}

	var s string
	for _, c := range resp.Candidates {
		for _, p := range c.Content.Parts {
			if t, ok := p.(ai.Text); ok {
				s += string(t)
			}
		}
	}

	plan, err := a.parsePlanJSON(s)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func (a *AIQueryService) ExecuteQuery(ctx context.Context, plan *domain.SQLPlan) (string, error) {
	// Validate plan
	if _, ok := allowedTables[strings.ToLower(plan.Table)]; !ok || strings.ToUpper(plan.Operation) != "SELECT" {
		return "", fmt.Errorf("operation not allowed: only SELECT from %s", allowedTablesList())
	}

	query, args := a.buildSafeSelect(plan)
	rows, err := a.db.Query(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	return a.rowsToText(rows, 20)
}

// AnswerWithDB implements full flow:
// 1) Model menghasilkan rencana SELECT aman (PlanQuery)
// 2) Eksekusi ke database (ExecuteQuery)
// 3) Gabungkan hasil sebagai konteks, lalu minta jawaban AI berbasis basePrompt + pertanyaan user
func (a *AIQueryService) AnswerWithDB(ctx context.Context, text string, basePrompt string) (string, error) {
	// Generate plan (uses model if geminiKey set, else naive)
	plan, err := a.PlanQuery(ctx, text)
	if err != nil {
		return "", fmt.Errorf("plan error: %w", err)
	}

	// Execute query
	dbContext, err := a.ExecuteQuery(ctx, plan)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}

	// If no AI key, fallback to DB context
	if strings.TrimSpace(a.geminiKey) == "" {
		if strings.TrimSpace(dbContext) != "" {
			return dbContext, nil
		}
		return "AI tidak aktif dan tidak ada konteks data.", nil
	}

	client, err := ai.NewClient(ctx, option.WithAPIKey(a.geminiKey))
	if err != nil {
		return "", fmt.Errorf("gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.0-flash")

	var sb strings.Builder
	if basePrompt != "" {
		sb.WriteString(basePrompt)
		sb.WriteString("\n\n")
	}
	if strings.TrimSpace(dbContext) != "" {
		sb.WriteString("[KONTEKS DATA]:\n")
		sb.WriteString(dbContext)
		sb.WriteString("\n\n")
	}
	sb.WriteString("[PERTANYAAN USER]: ")
	sb.WriteString(text)

	resp, err := model.GenerateContent(ctx, ai.Text(sb.String()))
	if err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}

	var out string
	for _, c := range resp.Candidates {
		for _, p := range c.Content.Parts {
			if t, ok := p.(ai.Text); ok {
				out += string(t)
			}
		}
	}

	if strings.TrimSpace(out) == "" {
		if strings.TrimSpace(dbContext) != "" {
			return dbContext, nil
		}
		return "Tidak ada jawaban.", nil
	}
	return out, nil
}

// AnswerWithDBForUser: ambil konteks user berdasarkan phone, batasi rencana query ke user terkait bila mungkin,
// gabungkan konteks user + hasil DB, lalu minta AI merumuskan jawaban akhir.
func (a *AIQueryService) AnswerWithDBForUser(ctx context.Context, userPhone string, text string, basePrompt string) (string, error) {
	userID, userCtx, err := a.getUserContext(ctx, userPhone)
	if err != nil {
		// Jangan hard fail, lanjut tanpa userCtx tapi beri info minimal
		userCtx = fmt.Sprintf("User tidak ditemukan untuk phone=%s", userPhone)
	}

	// Generate plan
	plan, err := a.PlanQuery(ctx, text)
	if err != nil {
		return "", fmt.Errorf("plan error: %w", err)
	}

	// Personalisasi: injeksikan filter user bila tabel mendukung
	// known tables with user_id: user_profiles, kpr_applications, branch_staff
	// approval_workflow: pakai assigned_to
	// users: pakai phone
	tbl := strings.ToLower(plan.Table)
	addFilter := func(column string, value string) {
		if plan.Filters == nil {
			plan.Filters = []domain.Filter{}
		}
		// Hindari duplikasi kolom
		for _, f := range plan.Filters {
			if strings.EqualFold(f.Column, column) {
				return
			}
		}
		plan.Filters = append(plan.Filters, domain.Filter{Column: column, Op: "=", Value: value})
	}

	if userID > 0 {
		switch tbl {
		case "user_profiles", "kpr_applications", "branch_staff":
			addFilter("user_id", fmt.Sprintf("%d", userID))
		case "approval_workflow":
			addFilter("assigned_to", fmt.Sprintf("%d", userID))
		}
	}
	if tbl == "users" && strings.TrimSpace(userPhone) != "" {
		addFilter("phone", userPhone)
	}

	// Execute query
	dbContext, err := a.ExecuteQuery(ctx, plan)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}

	// Jika tidak ada AI key, gabungkan userCtx + dbContext
	if strings.TrimSpace(a.geminiKey) == "" {
		var sb strings.Builder
		if strings.TrimSpace(userCtx) != "" {
			sb.WriteString(userCtx)
			sb.WriteString("\n")
		}
		if strings.TrimSpace(dbContext) != "" {
			sb.WriteString(dbContext)
		}
		out := strings.TrimSpace(sb.String())
		if out == "" {
			return "AI tidak aktif dan tidak ada konteks data.", nil
		}
		return out, nil
	}

	// Gabungkan ke prompt akhir
	client, err := ai.NewClient(ctx, option.WithAPIKey(a.geminiKey))
	if err != nil {
		return "", fmt.Errorf("gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.0-flash")

	var sb strings.Builder
	if basePrompt != "" {
		sb.WriteString(basePrompt)
		sb.WriteString("\n\n")
	}
	if strings.TrimSpace(userCtx) != "" {
		sb.WriteString("[KONTEKS USER]:\n")
		sb.WriteString(userCtx)
		sb.WriteString("\n\n")
	}
	if strings.TrimSpace(dbContext) != "" {
		sb.WriteString("[KONTEKS DATA]:\n")
		sb.WriteString(dbContext)
		sb.WriteString("\n\n")
	}
	sb.WriteString("[PERTANYAAN USER]: ")
	sb.WriteString(text)

	resp, err := model.GenerateContent(ctx, ai.Text(sb.String()))
	if err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}

	var out string
	for _, c := range resp.Candidates {
		for _, p := range c.Content.Parts {
			if t, ok := p.(ai.Text); ok {
				out += string(t)
			}
		}
	}

	if strings.TrimSpace(out) == "" {
		// fallback: gabungkan userCtx + dbContext
		var fb strings.Builder
		if strings.TrimSpace(userCtx) != "" {
			fb.WriteString(userCtx)
			fb.WriteString("\n")
		}
		if strings.TrimSpace(dbContext) != "" {
			fb.WriteString(dbContext)
		}
		res := strings.TrimSpace(fb.String())
		if res == "" {
			return "Tidak ada jawaban.", nil
		}
		return res, nil
	}
	return out, nil
}

// getUserContext mengambil user dari tabel users berdasarkan phone dan (opsional) profil dari user_profiles
// mengembalikan userID dan ringkasan konteks
func (a *AIQueryService) getUserContext(ctx context.Context, phone string) (int, string, error) {
	if strings.TrimSpace(phone) == "" {
		return 0, "", fmt.Errorf("phone kosong")
	}
	// Query users by phone
	q := "SELECT id, username, email, status, created_at FROM users WHERE phone = $1 LIMIT 1"
	rows, err := a.db.Query(ctx, q, phone)
	if err != nil {
		return 0, "", fmt.Errorf("db users: %w", err)
	}
	defer rows.Close()

	var (
		id        int
		username  sql.NullString
		email     sql.NullString
		status    sql.NullString
		createdAt sql.NullString
	)
	if rows.Next() {
		if err := rows.Scan(&id, &username, &email, &status, &createdAt); err != nil {
			return 0, "", err
		}
	} else {
		return 0, "", fmt.Errorf("user tidak ditemukan")
	}

	// Optional: profile
	pq := "SELECT full_name, monthly_income, occupation FROM user_profiles WHERE user_id = $1 LIMIT 1"
	profRows, err := a.db.Query(ctx, pq, id)
	if err != nil {
		// tidak fatal
		profRows = nil
	}
	var (
		fullName      sql.NullString
		monthlyIncome sql.NullString
		occupation    sql.NullString
	)
	if profRows != nil {
		defer profRows.Close()
		if profRows.Next() {
			_ = profRows.Scan(&fullName, &monthlyIncome, &occupation)
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "user_id=%d username=%s email=%s status=%s created_at=%s", id, nullStr(username), nullStr(email), nullStr(status), nullStr(createdAt))
	if fullName.Valid || monthlyIncome.Valid || occupation.Valid {
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "profile: full_name=%s monthly_income=%s occupation=%s", nullStr(fullName), nullStr(monthlyIncome), nullStr(occupation))
	}
	return id, sb.String(), nil
}

func nullStr(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

func (a *AIQueryService) naivePlan(text string) (*domain.SQLPlan, error) {
	lower := strings.ToLower(text)
	var tbl string
	// pilih tabel pertama yang disebutkan dan masuk whitelist
	names := make([]string, 0, len(allowedTables))
	for k := range allowedTables {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, n := range names {
		if strings.Contains(lower, n) {
			tbl = n
			break
		}
	}
	if tbl == "" {
		return nil, fmt.Errorf("table not allowed")
	}

	return &domain.SQLPlan{
		Operation: "SELECT",
		Table:     tbl,
		Filters:   []domain.Filter{},
		Limit:     20,
	}, nil
}

func (a *AIQueryService) parsePlanJSON(s string) (*domain.SQLPlan, error) {
	// Very simple JSON extractor to avoid dependency
	plan := &domain.SQLPlan{Operation: "SELECT"}
	s = strings.ReplaceAll(s, "\n", " ")

	if i := strings.Index(strings.ToLower(s), "\"table\""); i != -1 {
		j := strings.Index(s[i:], ":")
		k := strings.Index(s[i+j+1:], "\"")
		if j != -1 && k != -1 {
			z := s[i+j+1+k+1:]
			kk := strings.Index(z, "\"")
			if kk != -1 {
				plan.Table = z[:kk]
			}
		}
	}

	if plan.Table == "" {
		return nil, fmt.Errorf("invalid plan: missing table")
	}

	plan.Limit = 20
	return plan, nil
}

func (a *AIQueryService) buildSafeSelect(p *domain.SQLPlan) (string, []interface{}) {
	cols := "*"
	if len(p.Columns) > 0 {
		cols = strings.Join(p.Columns, ",")
	}

	q := fmt.Sprintf("SELECT %s FROM %s", cols, p.Table)
	args := []interface{}{}

	if len(p.Filters) > 0 {
		w := []string{}
		for i, f := range p.Filters {
			if f.Op != "=" { // only allow equality
				continue
			}
			w = append(w, fmt.Sprintf("%s = $%d", f.Column, i+1))
			args = append(args, f.Value)
		}
		if len(w) > 0 {
			q += " WHERE " + strings.Join(w, " AND ")
		}
	}

	if p.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", p.Limit)
	}

	return q, args
}

func (a *AIQueryService) rowsToText(rows interface{}, max int) (string, error) {
	// Type assertion for sql.Rows
	sqlRows, ok := rows.(interface {
		Columns() ([]string, error)
		Next() bool
		Scan(dest ...interface{}) error
	})
	if !ok {
		return "", fmt.Errorf("invalid rows type")
	}

	cols, err := sqlRows.Columns()
	if err != nil {
		return "", err
	}

	var out strings.Builder
	count := 0

	for sqlRows.Next() {
		vals := make([]interface{}, len(cols))
		scans := make([]interface{}, len(cols))
		for i := range vals {
			scans[i] = &vals[i]
		}

		if err := sqlRows.Scan(scans...); err != nil {
			return "", err
		}

		for i, c := range cols {
			fmt.Fprintf(&out, "%s=%v ", c, vals[i])
		}
		out.WriteString("\n")
		count++

		if count >= max {
			break
		}
	}

	if count == 0 {
		return "Tidak ada hasil.", nil
	}

	return out.String(), nil
}
