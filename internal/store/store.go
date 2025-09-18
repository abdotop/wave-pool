package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// Store encapsulates database access.
type Store struct {
	DB *sql.DB
}

// New opens a SQLite database at dbPath and runs migrations.
func New(dbPath string) (*Store, error) {
	if dbPath == "" {
		return nil, errors.New("dbPath cannot be empty")
	}

	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_foreign_keys=on", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	return &Store{DB: db}, nil
}

// AutoMigrate applies pending goose migrations from the provided directory.
// It is safe to call repeatedly; goose will no-op if already up to date.
func (s *Store) AutoMigrate(migrationsDir string) error {
	if migrationsDir == "" {
		return errors.New("migrationsDir cannot be empty")
	}
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	absDir, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("resolve migrations dir: %w", err)
	}
	if err := goose.Up(s.DB, absDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
