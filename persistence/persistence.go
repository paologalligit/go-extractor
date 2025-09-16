package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paologalligit/go-extractor/entities"
)

// Persistence defines the interface for logging seat counts
// Implementations: FilePersistence, PostgresPersistence
type Persistence interface {
	WriteSessionSeats(ctx context.Context, entry entities.SeatLogEntry) error
}

// FilePersistence implements Persistence by appending to a file
type FilePersistence struct {
	FilePath string
	mu       sync.Mutex
}

func NewFilePersistence(filePath string) *FilePersistence {
	return &FilePersistence{FilePath: filePath}
}

func (f *FilePersistence) WriteSessionSeats(ctx context.Context, entry entities.SeatLogEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	file, err := os.OpenFile(f.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	if err := enc.Encode(entry); err != nil {
		return fmt.Errorf("error writing log entry: %w", err)
	}
	return nil
}

// PostgresPersistence implements Persistence by writing to the session table
type PostgresPersistence struct {
	Pool *pgxpool.Pool
}

func NewPostgresPersistence(pool *pgxpool.Pool) *PostgresPersistence {
	return &PostgresPersistence{Pool: pool}
}

func (p *PostgresPersistence) WriteSessionSeats(ctx context.Context, entry entities.SeatLogEntry) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO session (cinema_name, film_name, session_id, seats, logged_at, start_hour)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		entry.CinemaName,
		entry.FilmName,
		entry.SessionId,
		entry.Seats,
		entry.LoggedAt,
		entry.StartHour,
	)
	if err != nil {
		return fmt.Errorf("error inserting seat log entry: %w", err)
	}
	return nil
}
