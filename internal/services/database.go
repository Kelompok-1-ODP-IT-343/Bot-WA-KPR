package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Kelompok-1-ODP-IT-343/Bot-WA-KPR/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DatabaseService struct {
	db *sql.DB
}

func NewDatabaseService(databaseURL string) (domain.DatabaseService, error) {
	if databaseURL == "" {
		return &DatabaseService{}, nil // Allow nil DB for graceful degradation
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DatabaseService{db: db}, nil
}

func (d *DatabaseService) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return d.db.QueryContext(ctx, query, args...)
}

func (d *DatabaseService) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return d.db.ExecContext(ctx, query, args...)
}

func (d *DatabaseService) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}