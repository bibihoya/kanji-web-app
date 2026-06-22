package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"kanji-web-app/stroke"
)

var kanjiDir = filepath.Join("model", "kanji")

// Структура запроса от фронтенда
type CheckRequest struct {
	Kanji   string        `json:"kanji"`   // Например, "0f9ab" (ID из KanjiVG)
	Strokes [][]PointJSON `json:"strokes"` // Черты, нарисованные пользователем
}

type PointJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Структура ответа
type CheckResponse struct {
	Kanji         string             `json:"kanji"`
	OverallScore  float64            `json:"overallScore"`
	OrderCorrect  bool               `json:"orderCorrect"`
	Feedback      string             `json:"feedback"`
	StrokeResults []StrokeResultJSON `json:"strokeResults"`
}

type StrokeResultJSON struct {
	Order        int     `json:"order"`
	Similarity   float64 `json:"similarity"`
	AngleDiff    float64 `json:"angleDiff"`
	LengthDiff   float64 `json:"lengthDiff"`
	OverallScore float64 `json:"overallScore"`
}

func main() {
	// Статические файлы (index.html, CSS, JS)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// API для проверки черт
	http.HandleFunc("/api/check", handleCheck)

	// API для получения списка доступных иероглифов
	http.HandleFunc("/api/kanji", handleKanjiList)
	// API для получения SVG эталона
	http.HandleFunc("/api/template/", handleTemplate)
	// API для проверки одной черты
	http.HandleFunc("/api/check-stroke", handleCheckStroke)

	log.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Только POST", http.StatusMethodNotAllowed)
		return
	}

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Неверный JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Загружаем эталон
	svgPath := filepath.Join(kanjiDir, req.Kanji+".svg")
	template, err := stroke.LoadKanjiVG(svgPath, req.Kanji)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка загрузки эталона: %v", err), http.StatusInternalServerError)
		return
	}

	// Конвертируем пользовательский ввод в []stroke.Stroke
	userStrokes := make([]stroke.Stroke, len(req.Strokes))
	for i, pts := range req.Strokes {
		strokePts := make([]stroke.Point, len(pts))
		for j, p := range pts {
			strokePts[j] = stroke.Point{X: p.X, Y: p.Y}
		}
		userStrokes[i] = stroke.Stroke{
			Points: strokePts,
			Order:  i + 1,
		}
	}

	// Анализируем
	result := stroke.AnalyzeKanji(template, userStrokes)

	// Формируем ответ
	resp := CheckResponse{
		Kanji:         result.Char,
		OverallScore:  result.OverallScore,
		OrderCorrect:  result.OrderCorrect,
		Feedback:      result.Feedback,
		StrokeResults: make([]StrokeResultJSON, len(result.StrokeResults)),
	}
	for i, sr := range result.StrokeResults {
		resp.StrokeResults[i] = StrokeResultJSON{
			Order:        sr.Order,
			Similarity:   sr.Similarity,
			AngleDiff:    sr.AngleDiff,
			LengthDiff:   sr.LengthDiff,
			OverallScore: sr.OverallScore,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleTemplate(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID иероглифа из URL: /api/template/0f9ab
	id := strings.TrimPrefix(r.URL.Path, "/api/template/")
	if id == "" {
		http.Error(w, "Не указан ID", http.StatusBadRequest)
		return
	}
	svgPath := filepath.Join(kanjiDir, id+".svg")
	http.ServeFile(w, r, svgPath)
}

type CheckStrokeRequest struct {
	Kanji       string      `json:"kanji"`
	StrokeIndex int         `json:"strokeIndex"`
	Points      []PointJSON `json:"points"`
}

type CheckStrokeResponse struct {
	OverallScore float64 `json:"overallScore"`
	Similarity   float64 `json:"similarity"`
	AngleDiff    float64 `json:"angleDiff"`
	LengthDiff   float64 `json:"lengthDiff"`
}

func handleCheckStroke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Только POST", http.StatusMethodNotAllowed)
		return
	}

	var req CheckStrokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Неверный JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Загружаем эталон
	svgPath := filepath.Join(kanjiDir, req.Kanji+".svg")
	template, err := stroke.LoadKanjiVG(svgPath, req.Kanji)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка загрузки эталона: %v", err), http.StatusInternalServerError)
		return
	}

	if req.StrokeIndex < 0 || req.StrokeIndex >= len(template.Strokes) {
		http.Error(w, "Неверный индекс черты", http.StatusBadRequest)
		return
	}

	// Конвертируем точки пользователя
	userPts := make([]stroke.Point, len(req.Points))
	for i, p := range req.Points {
		userPts[i] = stroke.Point{X: p.X, Y: p.Y}
	}
	userStroke := stroke.Stroke{Points: userPts, Order: req.StrokeIndex + 1}

	// Сравниваем
	result := stroke.CompareStrokes(template.Strokes[req.StrokeIndex], userStroke, req.StrokeIndex+1)

	resp := CheckStrokeResponse{
		OverallScore: result.OverallScore,
		Similarity:   result.Similarity,
		AngleDiff:    result.AngleDiff,
		LengthDiff:   result.LengthDiff,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

var (
	kanjiCache  []KanjiInfo
	jlptLevels  map[string][]string
	cacheLoaded bool
)

func loadCache() {
	if cacheLoaded {
		return
	}

	// Загружаем уровни JLPT
	jlptData, err := os.ReadFile(filepath.Join("model", "jlpt_levels.json"))
	if err == nil {
		json.Unmarshal(jlptData, &jlptLevels)
	}

	// Загружаем список иероглифов из SVG
	files, err := os.ReadDir(kanjiDir)
	if err != nil {
		return
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".svg") {
			id := strings.TrimSuffix(f.Name(), ".svg")
			svgPath := filepath.Join(kanjiDir, f.Name())

			char, err := stroke.ExtractKanjiChar(svgPath)
			if err != nil {
				char = id
			}

			// Определяем уровень
			level := ""
			for lvl, chars := range jlptLevels {
				for _, c := range chars {
					if c == char {
						level = lvl
						break
					}
				}
				if level != "" {
					break
				}
			}

			kanjiCache = append(kanjiCache, KanjiInfo{
				ID:        id,
				Character: char,
				Level:     level,
			})
		}
	}

	cacheLoaded = true
}

type KanjiInfo struct {
	ID        string `json:"id"`
	Character string `json:"character"`
	Level     string `json:"level,omitempty"`
}

func handleKanjiList(w http.ResponseWriter, r *http.Request) {
	loadCache()

	level := r.URL.Query().Get("level")

	var filtered []KanjiInfo
	for _, k := range kanjiCache {
		if level == "" || k.Level == level {
			filtered = append(filtered, k)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}
