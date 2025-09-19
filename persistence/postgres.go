package persistence

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// NewPostgresPool creates a new pgx connection pool
func NewPostgresPool(ctx context.Context) (*pgxpool.Pool, error) {
	_ = godotenv.Load() // Load .env if present, ignore error
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}
	return pool, nil
}

// InitPostgresSchema reads db/schema.sql and executes its statements
func InitPostgresSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schemaFile := "db/schema.sql"
	sqlBytes, err := os.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}
	sql := string(sqlBytes)
	// Split on semicolon to support multiple statements
	stmts := strings.SplitSeq(sql, ";")
	for stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %q: %w", stmt, err)
		}
	}
	return nil
}
