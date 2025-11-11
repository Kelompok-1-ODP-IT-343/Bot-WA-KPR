package services

import (
	"context"
	"fmt"
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
        "table (salah satu dari: users, roles, branch_staff, user_profiles, kpr_rates, kpr_applications, approval_workflow, properties), " +
        "columns (opsional array nama kolom), " +
        "filters (opsional array dari objek {column, op='=', value}), " +
        "limit (opsional int; default 20). " +
        "Aturan keras: (1) HANYA tabel whitelist di atas; gunakan nama persis sesuai DDL (termasuk singular 'approval_workflow'). " +
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
    if _, ok := allowedTables[plan.Table]; !ok || strings.ToUpper(plan.Operation) != "SELECT" {
        return "", fmt.Errorf("operation not allowed: only SELECT from users, roles, branch_staff, user_profiles, kpr_rates, kpr_applications, approval_workflow, properties")
    }

	query, args := a.buildSafeSelect(plan)
	rows, err := a.db.Query(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	return a.rowsToText(rows, 20)
}

func (a *AIQueryService) naivePlan(text string) (*domain.SQLPlan, error) {
	lower := strings.ToLower(text)
	var tbl string
	if strings.Contains(lower, "users") {
		tbl = "users"
	} else if strings.Contains(lower, "roles") {
		tbl = "roles"
	} else if strings.Contains(lower, "branch_staff") {
		tbl = "branch_staff"
	} else if strings.Contains(lower, "user_profiles") {
		tbl = "user_profiles"
	} else if strings.Contains(lower, "kpr_rates") {
		tbl = "kpr_rates"
	} else if strings.Contains(lower, "kpr_applications") {
		tbl = "kpr_applications"
	} else if strings.Contains(lower, "approval_workflow") { // singular sesuai DDL
		tbl = "approval_workflow"
	} else if strings.Contains(lower, "properties") {
		tbl = "properties"
	} else {
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
