package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	// Создаём директорию если нет
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("создать директорию БД: %w", err)
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("открыть БД: %w", err)
	}

	// Настройки
	DB.SetMaxOpenConns(1) // SQLite в режиме одного писателя
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := DB.Exec(p); err != nil {
			return fmt.Errorf("pragma %s: %w", p, err)
		}
	}

	return nil
}

func RunSchema(schemaPath string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("читать схему: %w", err)
	}

	if _, err := DB.Exec(string(data)); err != nil {
		return fmt.Errorf("выполнить схему: %w", err)
	}

	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
