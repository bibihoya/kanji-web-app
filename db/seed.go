package db

import (
	"encoding/json"
	"fmt"
	"os"
)

func SeedFromJSON(dictPath, jlptPath string) error {
	// Загружаем словарь
	dictData, err := os.ReadFile(dictPath)
	if err != nil {
		return fmt.Errorf("читать словарь: %w", err)
	}
	var kanjiDict map[string]struct {
		Onyomi      []string `json:"onyomi"`
		Kunyomi     []string `json:"kunyomi"`
		Meaning     string   `json:"meaning"`
		Jlpt        string   `json:"jlpt"`
		StrokeCount int      `json:"stroke_count"`
	}
	if err := json.Unmarshal(dictData, &kanjiDict); err != nil {
		return fmt.Errorf("парсить словарь: %w", err)
	}

	// Создаём пользователя по умолчанию
	DB.Exec("INSERT OR IGNORE INTO users (id, username) VALUES (1, 'default')")

	// Вставляем иероглифы
	stmt, err := DB.Prepare("INSERT OR IGNORE INTO kanji_characters (character, onyomi, kunyomi, meaning, jlpt_level, stroke_count) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for char, data := range kanjiDict {
		onyomiJSON, _ := json.Marshal(data.Onyomi)
		kunyomiJSON, _ := json.Marshal(data.Kunyomi)
		stmt.Exec(char, string(onyomiJSON), string(kunyomiJSON), data.Meaning, data.Jlpt, data.StrokeCount)
	}

	fmt.Printf("Загружено %d иероглифов\n", len(kanjiDict))
	return nil
}
