package db

import (
	"database/sql"
	"encoding/json"
)

type KanjiCharacter struct {
	ID          int      `json:"id"`
	Character   string   `json:"character"`
	Onyomi      []string `json:"onyomi"`
	Kunyomi     []string `json:"kunyomi"`
	Meaning     string   `json:"meaning"`
	JlptLevel   string   `json:"jlpt_level"`
	StrokeCount int      `json:"stroke_count"`
	SvgFilename string   `json:"svg_filename"`
}

// ListByJLPT возвращает иероглифы по уровню JLPT
func ListKanjiByJLPT(level string) ([]KanjiCharacter, error) {
	query := "SELECT id, character, onyomi, kunyomi, meaning, jlpt_level, stroke_count, svg_filename FROM kanji_characters"
	args := []interface{}{}
	if level != "" {
		query += " WHERE jlpt_level = ?"
		args = append(args, level)
	}
	query += " ORDER BY character"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []KanjiCharacter
	for rows.Next() {
		var k KanjiCharacter
		var onyomiJSON, kunyomiJSON string
		if err := rows.Scan(&k.ID, &k.Character, &onyomiJSON, &kunyomiJSON, &k.Meaning, &k.JlptLevel, &k.StrokeCount, &k.SvgFilename); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(onyomiJSON), &k.Onyomi)
		json.Unmarshal([]byte(kunyomiJSON), &k.Kunyomi)
		result = append(result, k)
	}
	return result, nil
}

// GetByCharacter возвращает иероглиф по символу
func GetKanjiByCharacter(char string) (*KanjiCharacter, error) {
	query := "SELECT id, character, onyomi, kunyomi, meaning, jlpt_level, stroke_count, svg_filename FROM kanji_characters WHERE character = ?"
	var k KanjiCharacter
	var onyomiJSON, kunyomiJSON string
	err := DB.QueryRow(query, char).Scan(&k.ID, &k.Character, &onyomiJSON, &kunyomiJSON, &k.Meaning, &k.JlptLevel, &k.StrokeCount, &k.SvgFilename)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(onyomiJSON), &k.Onyomi)
	json.Unmarshal([]byte(kunyomiJSON), &k.Kunyomi)
	return &k, nil
}

// GetAllCharacters возвращает все иероглифы
func GetAllKanji() ([]KanjiCharacter, error) {
	return ListKanjiByJLPT("")
}

// UpdateSVGFilenames заполняет svg_filename для всех иероглифов
func UpdateSVGFilenames(charToFile map[string]string) error {
	stmt, err := DB.Prepare("UPDATE kanji_characters SET svg_filename = ? WHERE character = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for char, filename := range charToFile {
		_, err := stmt.Exec(filename, char)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadSVGCache загружает соответствие character -> svg_filename из БД
func LoadSVGCache() (map[string]string, map[string]string, error) {
	rows, err := DB.Query("SELECT character, svg_filename FROM kanji_characters WHERE svg_filename != ''")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	charToFile := make(map[string]string)
	fileToChar := make(map[string]string)

	for rows.Next() {
		var char, filename string
		if err := rows.Scan(&char, &filename); err != nil {
			continue
		}
		charToFile[char] = filename
		fileToChar[filename] = char
	}

	return charToFile, fileToChar, nil
}
