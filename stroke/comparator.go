package stroke

import (
	"fmt"
	"math"
	"strings"
)

func DTW(a, b []Point) float64 {
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		return math.Inf(1)
	}

	dp := make([][]float64, n+1)
	for i := range dp {
		dp[i] = make([]float64, m+1)
		for j := range dp[i] {
			dp[i][j] = math.Inf(1)
		}
	}
	dp[0][0] = 0

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := a[i-1].DistanceTo(b[j-1])
			dp[i][j] = cost + min(
				dp[i-1][j],
				dp[i][j-1],
				dp[i-1][j-1],
			)
		}
	}

	return dp[n][m]
}

func CompareStrokes(template, drawn Stroke, order int) StrokeResult {
	t := Stroke{Points: make([]Point, len(template.Points))}
	copy(t.Points, template.Points)
	d := Stroke{Points: make([]Point, len(drawn.Points))}
	copy(d.Points, drawn.Points)

	// Отладочный вывод ДО обработки
	fmt.Printf("\n=== ЧЕРТА %d: ДО ОБРАБОТКИ ===\n", order)
	fmt.Printf("Эталон: %d точек, диапазон X=[%.1f-%.1f], Y=[%.1f-%.1f]\n",
		len(t.Points), minX(t.Points), maxX(t.Points), minY(t.Points), maxY(t.Points))
	fmt.Printf("Польз.: %d точек, диапазон X=[%.1f-%.1f], Y=[%.1f-%.1f]\n",
		len(d.Points), minX(d.Points), maxX(d.Points), minY(d.Points), maxY(d.Points))

	// Центрирование
	t.centerPoints()
	d.centerPoints()

	fmt.Printf("ПОСЛЕ ЦЕНТРИРОВАНИЯ:\n")
	fmt.Printf("Эталон: X=[%.3f-%.3f], Y=[%.3f-%.3f]\n",
		minX(t.Points), maxX(t.Points), minY(t.Points), maxY(t.Points))
	fmt.Printf("Польз.: X=[%.3f-%.3f], Y=[%.3f-%.3f]\n",
		minX(d.Points), maxX(d.Points), minY(d.Points), maxY(d.Points))

	tAngle := t.DirectionAngle()
	dAngle := d.DirectionAngle()
	angleDiff := math.Abs(tAngle-dAngle) * 180.0 / math.Pi
	if angleDiff > 180 {
		angleDiff = 360 - angleDiff
	}

	// 3. Поворачиваем пользовательскую черту к эталонному углу
	d.Rotate(tAngle - dAngle) // теперь направления совпадают

	// Нормализация размера
	t.normalizeSize()
	d.normalizeSize()

	fmt.Printf("ПОСЛЕ НОРМАЛИЗАЦИИ:\n")
	fmt.Printf("Эталон: X=[%.3f-%.3f], Y=[%.3f-%.3f]\n",
		minX(t.Points), maxX(t.Points), minY(t.Points), maxY(t.Points))
	fmt.Printf("Польз.: X=[%.3f-%.3f], Y=[%.3f-%.3f]\n",
		minX(d.Points), maxX(d.Points), minY(d.Points), maxY(d.Points))

	// DTW расстояние
	dtwScore := DTW(t.Points, d.Points)
	fmt.Printf("DTW расстояние: %.4f\n", dtwScore)

	// Преобразование в similarity
	similarity := 1.0 / (1.0 + dtwScore/15.0)
	fmt.Printf("Similarity (форма): %.4f\n", similarity)

	// Угол
	tAngle = t.DirectionAngle()
	dAngle = d.DirectionAngle()
	angleDiff = math.Abs(tAngle-dAngle) * 180.0 / math.Pi
	if angleDiff > 180 {
		angleDiff = 360 - angleDiff
	}
	fmt.Printf("Угол эталона: %.1f°, угол польз.: %.1f°, разница: %.1f°\n",
		tAngle*180.0/math.Pi, dAngle*180.0/math.Pi, angleDiff)

	// Длина
	tLen := t.Length()
	dLen := d.Length()
	lengthDiff := math.Abs(tLen-dLen) / math.Max(tLen, dLen)
	fmt.Printf("Длина эталона: %.4f, длина польз.: %.4f, разница: %.1f%%\n",
		tLen, dLen, lengthDiff*100)

	// Итоговые оценки
	angleScore := math.Exp(-angleDiff / 60.0)
	lengthScore := math.Exp(-lengthDiff * 2)
	overallScore := 0.45*similarity + 0.35*angleScore + 0.2*lengthScore
	fmt.Printf("Оценки: форма=%.3f, угол=%.3f, длина=%.3f, ИТОГО=%.3f\n",
		similarity, angleScore, lengthScore, overallScore)

	return StrokeResult{
		Order:        order,
		DTWScore:     dtwScore,
		Similarity:   similarity,
		AngleDiff:    angleDiff,
		LengthDiff:   lengthDiff,
		OverallScore: overallScore,
	}
}

// Вспомогательные функции для отладки
func minX(pts []Point) float64 {
	if len(pts) == 0 {
		return 0
	}
	m := pts[0].X
	for _, p := range pts {
		if p.X < m {
			m = p.X
		}
	}
	return m
}
func maxX(pts []Point) float64 {
	if len(pts) == 0 {
		return 0
	}
	m := pts[0].X
	for _, p := range pts {
		if p.X > m {
			m = p.X
		}
	}
	return m
}
func minY(pts []Point) float64 {
	if len(pts) == 0 {
		return 0
	}
	m := pts[0].Y
	for _, p := range pts {
		if p.Y < m {
			m = p.Y
		}
	}
	return m
}
func maxY(pts []Point) float64 {
	if len(pts) == 0 {
		return 0
	}
	m := pts[0].Y
	for _, p := range pts {
		if p.Y > m {
			m = p.Y
		}
	}
	return m
}

func AnalyzeKanji(template *KanjiTemplate, drawn []Stroke) AnalysisResult {
	result := AnalysisResult{
		Char:          template.Char,
		StrokeResults: make([]StrokeResult, 0, len(template.Strokes)),
		OrderCorrect:  true,
	}

	numToCompare := len(drawn)
	if numToCompare > len(template.Strokes) {
		numToCompare = len(template.Strokes)
	}

	totalScore := 0.0

	for i := 0; i < numToCompare; i++ {
		sr := CompareStrokes(template.Strokes[i], drawn[i], i+1)
		result.StrokeResults = append(result.StrokeResults, sr)
		totalScore += sr.OverallScore
	}

	if len(drawn) != len(template.Strokes) {
		result.OrderCorrect = false
	}

	if len(result.StrokeResults) > 0 {
		result.OverallScore = totalScore / float64(len(result.StrokeResults))
	}

	result.Feedback = generateFeedback(result, template, drawn)

	return result
}

func generateFeedback(result AnalysisResult, template *KanjiTemplate, drawn []Stroke) string {
	var feedbackParts []string

	// Общая оценка
	if result.OverallScore >= 0.85 {
		feedbackParts = append(feedbackParts, "🎉 Отлично! Иероглиф написан очень хорошо.")
	} else if result.OverallScore >= 0.7 {
		feedbackParts = append(feedbackParts, "👍 Хорошо! Есть небольшие замечания.")
	} else if result.OverallScore >= 0.5 {
		feedbackParts = append(feedbackParts, "📝 Неплохо, но нужно доработать некоторые черты.")
	} else {
		feedbackParts = append(feedbackParts, "🖌 Требуется практика. Обратите внимание на:")
	}

	// Детальные подсказки по каждой черте
	for _, sr := range result.StrokeResults {
		if sr.OverallScore < 0.5 {
			var issues []string
			if sr.Similarity < 0.4 {
				issues = append(issues, "форма")
			}
			if sr.AngleDiff > 30 {
				issues = append(issues, "направление")
			}
			if sr.LengthDiff > 0.4 {
				issues = append(issues, "длина")
			}
			if len(issues) > 0 {
				feedbackParts = append(feedbackParts,
					fmt.Sprintf("  • Черта %d: исправьте %s", sr.Order, strings.Join(issues, " и ")))
			}
		}
	}

	// Проверка количества черт
	if len(drawn) != len(template.Strokes) {
		feedbackParts = append(feedbackParts,
			fmt.Sprintf("  • Количество черт: у вас %d, должно быть %d", len(drawn), len(template.Strokes)))
	}

	return strings.Join(feedbackParts, "\n")
}

// Добавляем strings в импорт
