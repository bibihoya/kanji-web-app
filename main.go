package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kanji-web-app/db"
	"kanji-web-app/stroke"
)

var kanjiDir = filepath.Join("model", "kanji")

// Кеш для быстрого поиска SVG
var (
	svgCharToFile map[string]string // character -> svg filename (без .svg)
	svgFileToChar map[string]string // svg filename -> character
)

// ==================== СТРУКТУРЫ ====================

type PointJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
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

type CheckRequest struct {
	Kanji   string        `json:"kanji"`
	Strokes [][]PointJSON `json:"strokes"`
}

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

type QuizQuestion struct {
	Kanji        string   `json:"kanji"`
	Prompt       string   `json:"prompt,omitempty"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correctIndex"`
}

type KanjiInfo struct {
	ID        string `json:"id"`
	Character string `json:"character"`
	Level     string `json:"level,omitempty"`
}

// ==================== MAIN ====================

func main() {
	// 1. Инициализируем БД
	dbPath := filepath.Join("data", "kanji.db")
	if err := db.Init(dbPath); err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	// 2. Загружаем схему
	schemaPath := filepath.Join("db", "schema.sql")
	if err := db.RunSchema(schemaPath); err != nil {
		log.Fatalf("Ошибка загрузки схемы: %v", err)
	}

	// 3. Загружаем кеш SVG (быстро, ~100мс)
	initSVGCache()

	// 4. Загружаем начальные данные (медленно только в первый раз)
	seedIfEmpty()

	// 5. Статические файлы
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// 6. API endpoints
	http.HandleFunc("/api/kanji", handleKanjiList)
	http.HandleFunc("/api/template/", handleTemplate)
	http.HandleFunc("/api/check-stroke", handleCheckStroke)
	http.HandleFunc("/api/check", handleCheck)
	http.HandleFunc("/api/dictionary/", handleDictionary)
	http.HandleFunc("/api/quiz", handleQuiz)
	http.HandleFunc("/api/lessons", handleLessons)
	http.HandleFunc("/api/progress/", handleProgress)

	log.Println("🚀 Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ==================== КЕШ SVG ====================

func initSVGCache() {
	// Пробуем загрузить из БД (быстро)
	charToFile, fileToChar, err := db.LoadSVGCache()
	if err == nil && len(charToFile) > 0 {
		svgCharToFile = charToFile
		svgFileToChar = fileToChar
		log.Printf("📦 Кеш SVG загружен из БД: %d записей", len(charToFile))
		return
	}

	// Первый запуск — парсим файлы (медленно, но только один раз)
	log.Println("🔍 Первый запуск: парсинг SVG-файлов...")
	svgCharToFile = make(map[string]string)
	svgFileToChar = make(map[string]string)

	files, err := os.ReadDir(kanjiDir)
	if err != nil {
		log.Printf("⚠ Не могу прочитать папку SVG: %v", err)
		return
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".svg") {
			continue
		}
		svgID := strings.TrimSuffix(f.Name(), ".svg")
		svgPath := filepath.Join(kanjiDir, f.Name())

		char, err := stroke.ExtractKanjiChar(svgPath)
		if err != nil {
			char = svgID
		}

		svgCharToFile[char] = svgID
		svgFileToChar[svgID] = char
	}

	log.Printf("📦 Распарсено %d SVG-файлов", len(svgCharToFile))

	// Сохраняем в БД для будущих запусков
	if err := db.UpdateSVGFilenames(svgCharToFile); err != nil {
		log.Printf("⚠ Не удалось сохранить в БД: %v", err)
	} else {
		log.Println("✅ Кеш сохранён в БД")
	}
}

// Быстрый поиск SVG-файла по иероглифу (из кеша)
func findSVGByCharacter(char string) string {
	if id, ok := svgCharToFile[char]; ok {
		return id
	}
	return ""
}

// Быстрый поиск character по SVG-файлу (из кеша)
func findCharBySVG(svgID string) string {
	if char, ok := svgFileToChar[svgID]; ok {
		return char
	}
	return ""
}

// ==================== СИДЫ ====================

func seedIfEmpty() {
	kanji, _ := db.GetAllKanji()
	if len(kanji) > 0 {
		log.Printf("✅ БД уже содержит %d иероглифов, сиды пропущены", len(kanji))
		return
	}

	log.Println("🌱 Загрузка начальных данных из JSON...")
	dictPath := filepath.Join("model", "kanji_dict.json")
	if err := db.SeedFromJSON(dictPath, ""); err != nil {
		log.Printf("⚠ Предупреждение: ошибка загрузки сидов: %v", err)
	} else {
		log.Println("✅ Начальные данные загружены")
	}
}

// ==================== ОБРАБОТЧИКИ API ====================

func handleKanjiList(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")

	kanjiList, err := db.ListKanjiByJLPT(level)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка БД: %v", err), http.StatusInternalServerError)
		return
	}

	var result []KanjiInfo
	for _, k := range kanjiList {
		svgID := findSVGByCharacter(k.Character)
		if svgID == "" {
			svgID = k.SvgFilename
		}
		if svgID == "" {
			svgID = k.Character
		}
		result = append(result, KanjiInfo{
			ID:        svgID,
			Character: k.Character,
			Level:     k.JlptLevel,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleTemplate(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/template/")
	if id == "" {
		http.Error(w, "Не указан ID", http.StatusBadRequest)
		return
	}
	svgPath := filepath.Join(kanjiDir, id+".svg")
	if _, err := os.Stat(svgPath); os.IsNotExist(err) {
		http.Error(w, "SVG не найден", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, svgPath)
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

	userPts := make([]stroke.Point, len(req.Points))
	for i, p := range req.Points {
		userPts[i] = stroke.Point{X: p.X, Y: p.Y}
	}
	userStroke := stroke.Stroke{Points: userPts, Order: req.StrokeIndex + 1}

	result := stroke.CompareStrokes(template.Strokes[req.StrokeIndex], userStroke, req.StrokeIndex+1)

	// Сохраняем прогресс асинхронно
	go saveStrokeProgress(req.Kanji, req.StrokeIndex, result)

	resp := CheckStrokeResponse{
		OverallScore: result.OverallScore,
		Similarity:   result.Similarity,
		AngleDiff:    result.AngleDiff,
		LengthDiff:   result.LengthDiff,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func saveStrokeProgress(svgID string, strokeIndex int, result stroke.StrokeResult) {
	char := findCharBySVG(svgID)
	if char == "" {
		return
	}

	kanji, err := db.GetKanjiByCharacter(char)
	if err != nil || kanji == nil {
		return
	}

	// Сохраняем историю
	if err := db.SavePracticeHistory(1, kanji.ID, strokeIndex, result.OverallScore, result.DTWScore, result.AngleDiff, result.LengthDiff); err != nil {
		log.Printf("Ошибка сохранения истории: %v", err)
	}

	// Обновляем прогресс
	if err := db.UpdateProgress(1, kanji.ID, result.OverallScore); err != nil {
		log.Printf("Ошибка обновления прогресса: %v", err)
	}
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

	svgPath := filepath.Join(kanjiDir, req.Kanji+".svg")
	template, err := stroke.LoadKanjiVG(svgPath, req.Kanji)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка загрузки эталона: %v", err), http.StatusInternalServerError)
		return
	}

	userStrokes := make([]stroke.Stroke, len(req.Strokes))
	for i, pts := range req.Strokes {
		strokePts := make([]stroke.Point, len(pts))
		for j, p := range pts {
			strokePts[j] = stroke.Point{X: p.X, Y: p.Y}
		}
		userStrokes[i] = stroke.Stroke{Points: strokePts, Order: i + 1}
	}

	result := stroke.AnalyzeKanji(template, userStrokes)

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

func handleDictionary(w http.ResponseWriter, r *http.Request) {
	char := strings.TrimPrefix(r.URL.Path, "/api/dictionary/")
	if char == "" {
		http.Error(w, "Не указан иероглиф", http.StatusBadRequest)
		return
	}

	// Декодируем URL-encoded иероглиф
	char, _ = urlDecode(char)

	kanji, err := db.GetKanjiByCharacter(char)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка БД: %v", err), http.StatusInternalServerError)
		return
	}
	if kanji == nil {
		http.Error(w, "Иероглиф не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kanji)
}

func urlDecode(s string) (string, error) {
	result, err := strconv.Unquote(`"` + s + `"`)
	if err != nil {
		// Пробуем простой вариант
		return s, nil
	}
	return result, nil
}

func handleQuiz(w http.ResponseWriter, r *http.Request) {
	quizType := r.URL.Query().Get("type")
	level := r.URL.Query().Get("level")

	kanjiList, err := db.ListKanjiByJLPT(level)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка БД: %v", err), http.StatusInternalServerError)
		return
	}

	if len(kanjiList) < 4 {
		http.Error(w, "Недостаточно иероглифов для этого уровня", http.StatusNotFound)
		return
	}

	// Выбираем правильный ответ
	correctIdx := rand.Intn(len(kanjiList))
	correct := kanjiList[correctIdx]

	// 3 неправильных
	wrongIndices := make([]int, 0, 3)
	used := map[int]bool{correctIdx: true}
	for len(wrongIndices) < 3 {
		idx := rand.Intn(len(kanjiList))
		if !used[idx] {
			used[idx] = true
			wrongIndices = append(wrongIndices, idx)
		}
	}

	var question QuizQuestion

	switch quizType {
	case "reading":
		allReadings := append([]string{}, correct.Onyomi...)
		allReadings = append(allReadings, correct.Kunyomi...)
		if len(allReadings) == 0 {
			http.Error(w, "Нет чтений", http.StatusNotFound)
			return
		}
		correctReading := allReadings[rand.Intn(len(allReadings))]

		wrongReadings := make([]string, 0, 3)
		for _, idx := range wrongIndices {
			pool := append([]string{}, kanjiList[idx].Onyomi...)
			pool = append(pool, kanjiList[idx].Kunyomi...)
			if len(pool) > 0 {
				wrongReadings = append(wrongReadings, pool[rand.Intn(len(pool))])
			}
		}

		options := append([]string{correctReading}, wrongReadings...)
		correctOptionIndex := 0
		rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
		for i, opt := range options {
			if opt == correctReading {
				correctOptionIndex = i
				break
			}
		}
		question = QuizQuestion{
			Kanji: correct.Character, Options: options, CorrectIndex: correctOptionIndex,
		}

	case "meaning":
		correctMeaning := correct.Meaning
		if correctMeaning == "" {
			http.Error(w, "Нет значения", http.StatusNotFound)
			return
		}
		wrongMeanings := make([]string, 0, 3)
		for _, idx := range wrongIndices {
			m := kanjiList[idx].Meaning
			if m != "" && m != correctMeaning {
				wrongMeanings = append(wrongMeanings, m)
			}
		}
		for len(wrongMeanings) < 3 {
			wrongMeanings = append(wrongMeanings, "—")
		}
		options := append([]string{correctMeaning}, wrongMeanings[:3]...)
		correctOptionIndex := 0
		rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
		for i, opt := range options {
			if opt == correctMeaning {
				correctOptionIndex = i
				break
			}
		}
		question = QuizQuestion{
			Kanji: correct.Character, Options: options, CorrectIndex: correctOptionIndex,
		}

	case "character":
		var prompt string
		if len(correct.Onyomi) > 0 {
			prompt = correct.Onyomi[0]
		} else if len(correct.Kunyomi) > 0 {
			prompt = correct.Kunyomi[0]
		} else {
			prompt = correct.Meaning
		}

		options := make([]string, 4)
		correctOptionIndex := rand.Intn(4)
		options[correctOptionIndex] = correct.Character
		for i, j := 0, 0; i < 4 && j < len(wrongIndices); i++ {
			if i != correctOptionIndex {
				options[i] = kanjiList[wrongIndices[j]].Character
				j++
			}
		}
		question = QuizQuestion{
			Kanji: correct.Character, Prompt: prompt, Options: options, CorrectIndex: correctOptionIndex,
		}

	default:
		http.Error(w, "Неизвестный тип задания. Используйте: reading, meaning, character", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(question)
}

func handleLessons(w http.ResponseWriter, r *http.Request) {
	lessonData, err := os.ReadFile(filepath.Join("model", "lessons.json"))
	if err != nil {
		http.Error(w, "Уроки не найдены", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(lessonData)
}

func handleProgress(w http.ResponseWriter, r *http.Request) {
	userID := 1 // Пока один пользователь

	kanjiIDStr := strings.TrimPrefix(r.URL.Path, "/api/progress/")
	kanjiID, _ := strconv.Atoi(kanjiIDStr)

	if kanjiID > 0 {
		// Конкретный иероглиф
		progress, err := db.GetOrCreateProgress(userID, kanjiID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(progress)
		return
	}

	// Все иероглифы для повторения
	due, err := db.GetDueReviews(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(due)
}
