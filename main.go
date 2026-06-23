package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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

	http.HandleFunc("/api/dictionary/", handleDictionary)

	http.HandleFunc("/api/quiz", handleQuiz)

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

var kanjiDict map[string]KanjiDictEntry

type KanjiDictEntry struct {
	Onyomi      []string   `json:"onyomi"`
	Kunyomi     []string   `json:"kunyomi"`
	Meaning     string     `json:"meaning"`
	JLPT        string     `json:"jlpt"`
	StrokeCount int        `json:"stroke_count"`
	Words       []WordInfo `json:"words"`
}

type WordInfo struct {
	Word    string `json:"word"`
	Reading string `json:"reading"`
	Meaning string `json:"meaning"`
}

func loadCache() {
	if cacheLoaded {
		return
	}

	// Загружаем уровни JLPT
	jlptData, err := os.ReadFile(filepath.Join("model", "jlpt_levels.json"))
	if err == nil {
		json.Unmarshal(jlptData, &jlptLevels)
	}

	dictData, err := os.ReadFile(filepath.Join("model", "kanji_dict.json"))
	if err == nil {
		json.Unmarshal(dictData, &kanjiDict)
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

func handleDictionary(w http.ResponseWriter, r *http.Request) {
	kanji := strings.TrimPrefix(r.URL.Path, "/api/dictionary/")
	if kanji == "" {
		http.Error(w, "Не указан иероглиф", http.StatusBadRequest)
		return
	}

	loadCache()

	entry, ok := kanjiDict[kanji]
	if !ok {
		http.Error(w, "Иероглиф не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// Структура для вопроса
type QuizQuestion struct {
	Kanji        string   `json:"kanji"`
	Prompt       string   `json:"prompt,omitempty"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correctIndex"`
}

func handleQuiz(w http.ResponseWriter, r *http.Request) {
	loadCache()

	quizType := r.URL.Query().Get("type")
	level := r.URL.Query().Get("level")

	// Собираем доступные иероглифы
	var available []KanjiInfo
	for _, k := range kanjiCache {
		if level == "" || k.Level == level {
			if _, ok := kanjiDict[k.Character]; ok {
				available = append(available, k)
			}
		}
	}

	if len(available) < 4 {
		http.Error(w, "Недостаточно иероглифов для этого уровня", http.StatusNotFound)
		return
	}

	// Выбираем правильный ответ
	correctIdx := rand.Intn(len(available))
	correct := available[correctIdx]
	correctData := kanjiDict[correct.Character]

	// Собираем 3 неправильных варианта
	wrongIndices := make([]int, 0, 3)
	used := map[int]bool{correctIdx: true}
	for len(wrongIndices) < 3 {
		idx := rand.Intn(len(available))
		if !used[idx] {
			used[idx] = true
			wrongIndices = append(wrongIndices, idx)
		}
	}

	switch quizType {
	case "reading":
		// Собираем все чтения правильного иероглифа
		allReadings := append([]string{}, correctData.Onyomi...)
		allReadings = append(allReadings, correctData.Kunyomi...)
		if len(allReadings) == 0 {
			http.Error(w, "Нет чтений для выбранного иероглифа", http.StatusNotFound)
			return
		}
		correctReading := allReadings[rand.Intn(len(allReadings))]

		// Собираем неправильные чтения из других иероглифов
		wrongReadings := make([]string, 0, 3)
		for _, idx := range wrongIndices {
			data := kanjiDict[available[idx].Character]
			pool := append([]string{}, data.Onyomi...)
			pool = append(pool, data.Kunyomi...)
			if len(pool) > 0 {
				wrongReadings = append(wrongReadings, pool[rand.Intn(len(pool))])
			}
		}

		// Формируем варианты и перемешиваем
		options := append([]string{correctReading}, wrongReadings...)
		correctOptionIndex := 0
		rand.Shuffle(len(options), func(i, j int) {
			options[i], options[j] = options[j], options[i]
		})
		for i, opt := range options {
			if opt == correctReading {
				correctOptionIndex = i
			}
		}

		question := QuizQuestion{
			Kanji:        correct.Character,
			Options:      options,
			CorrectIndex: correctOptionIndex,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(question)

	case "meaning":
		correctMeaning := correctData.Meaning
		if correctMeaning == "" {
			http.Error(w, "Нет значения для выбранного иероглифа", http.StatusNotFound)
			return
		}

		// Собираем неправильные значения
		wrongMeanings := make([]string, 0, 3)
		for _, idx := range wrongIndices {
			data := kanjiDict[available[idx].Character]
			if data.Meaning != "" && data.Meaning != correctMeaning {
				wrongMeanings = append(wrongMeanings, data.Meaning)
			}
		}
		// Если не хватило уникальных значений, добавляем заглушки
		for len(wrongMeanings) < 3 {
			wrongMeanings = append(wrongMeanings, "—")
		}

		options := append([]string{correctMeaning}, wrongMeanings[:3]...)
		correctOptionIndex := 0
		rand.Shuffle(len(options), func(i, j int) {
			options[i], options[j] = options[j], options[i]
		})
		for i, opt := range options {
			if opt == correctMeaning {
				correctOptionIndex = i
			}
		}

		question := QuizQuestion{
			Kanji:        correct.Character,
			Options:      options,
			CorrectIndex: correctOptionIndex,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(question)

	case "character":
		// Prompt — что показываем вместо иероглифа
		var prompt string
		if len(correctData.Onyomi) > 0 {
			prompt = correctData.Onyomi[0]
		} else if len(correctData.Kunyomi) > 0 {
			prompt = correctData.Kunyomi[0]
		} else {
			prompt = correctData.Meaning
		}

		// Формируем 4 варианта иероглифов
		options := make([]string, 4)
		correctOptionIndex := rand.Intn(4)
		options[correctOptionIndex] = correct.Character

		for i, j := 0, 0; i < 4 && j < len(wrongIndices); i++ {
			if i != correctOptionIndex {
				options[i] = available[wrongIndices[j]].Character
				j++
			}
		}

		question := QuizQuestion{
			Kanji:        correct.Character,
			Prompt:       prompt,
			Options:      options,
			CorrectIndex: correctOptionIndex,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(question)

	default:
		http.Error(w, "Неизвестный тип задания. Используйте: reading, meaning, character", http.StatusBadRequest)
	}
}
