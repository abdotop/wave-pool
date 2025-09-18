package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// TestNew_EmptyPath verifies New returns an error when dbPath is empty.
func TestNew_EmptyPath(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatalf("expected error for empty dbPath")
	}
}

// TestNew_CreatesDBAndPings ensures DB file is created and ping succeeds.
func TestNew_CreatesDBAndPings(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test_store.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })

	if err := s.DB.Ping(); err != nil {
		t.Fatalf("ping error: %v", err)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db file at %s: %v", dbPath, err)
	}
}

// TestGooseMigrations_Up ensures goose migrations can be applied and table exists with key columns.
func TestGooseMigrations_Up(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "migrate.db")

	dsn := "file:" + dbPath + "?_busy_timeout=5000&_foreign_keys=on"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Apply migrations from the repo root: db/migrations
	migrationsDir := filepath.Join("..", "..", "db", "migrations")
	absDir, err := filepath.Abs(migrationsDir)
	if err != nil {
		t.Fatalf("abs migrations dir: %v", err)
	}
	// Explicitly set dialect for programmatic goose usage
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	if err := goose.Up(db, absDir); err != nil {
		t.Fatalf("goose.Up: %v", err)
	}

	// Verify table exists
	var name string
	row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='checkout_sessions'")
	if err := row.Scan(&name); err != nil {
		t.Fatalf("checkout_sessions table not found: %v", err)
	}

	// Verify important columns exist
	cols := map[string]bool{}
	rows, err := db.Query("PRAGMA table_info(checkout_sessions)")
	if err != nil {
		t.Fatalf("pragma table_info: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var colName, colType string
		var notnull, pk int
		var dfltValue any
		if err := rows.Scan(&cid, &colName, &colType, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan pragma row: %v", err)
		}
		cols[colName] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}

	mustHave := []string{
		"id", "amount", "checkout_status", "currency", "error_url", "success_url",
		"payment_status", "wave_launch_url", "when_created", "when_expires",
		"restrict_payer_mobile", "enforce_payer_mobile", "last_payment_error_code", "last_payment_error_message",
	}
	for _, c := range mustHave {
		if !cols[c] {
			t.Fatalf("expected column %q to exist", c)
		}
	}
}

// TestAutoMigrate_CreatesTableAndIsIdempotent verifies AutoMigrate applies migrations
// and can be called multiple times safely without errors.
func TestAutoMigrate_CreatesTableAndIsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "auto.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })

	migrationsDir := filepath.Join("..", "..", "db", "migrations")

	if err := s.AutoMigrate(migrationsDir); err != nil {
		t.Fatalf("AutoMigrate first run: %v", err)
	}
	// Second run should be a no-op
	if err := s.AutoMigrate(migrationsDir); err != nil {
		t.Fatalf("AutoMigrate second run: %v", err)
	}

	// Verify the table exists after AutoMigrate
	var name string
	row := s.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='checkout_sessions'")
	if err := row.Scan(&name); err != nil {
		t.Fatalf("checkout_sessions table not found after AutoMigrate: %v", err)
	}
}

// TestAutoMigrate_EmptyDirError ensures an empty migrationsDir returns an error.
func TestAutoMigrate_EmptyDirError(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "auto-empty.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })

	if err := s.AutoMigrate(""); err == nil {
		t.Fatalf("expected error for empty migrationsDir")
	}
}
