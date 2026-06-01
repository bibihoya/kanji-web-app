package main

import (
	"fmt"

	"kanji-web-app/stroke"
)

func main() {
	// 1. Создаём простой эталон: вертикальная черта сверху вниз
	templateStroke := stroke.Stroke{
		Order: 1,
		Points: []stroke.Point{
			{X: 100, Y: 50},
			{X: 100, Y: 200},
		},
	}
	template := &stroke.KanjiTemplate{
		Char:    "тест",
		Strokes: []stroke.Stroke{templateStroke},
	}

	// 2. Имитируем пользовательский ввод: тоже вертикальная черта сверху вниз, но немного левее и длиннее
	drawnStroke := stroke.Stroke{
		Points: []stroke.Point{
			{X: 80, Y: 30},
			{X: 80, Y: 220},
		},
	}

	fmt.Println("=== Тест 1: Идеальное совпадение направления ===")
	result := stroke.CompareStrokes(template.Strokes[0], drawnStroke, 1)
	fmt.Printf("Результат: angleDiff=%.1f°, similarity=%.3f, итого=%.3f\n",
		result.AngleDiff, result.Similarity, result.OverallScore)

	// 3. Имитируем ввод в противоположном направлении: снизу вверх
	drawnStrokeReversed := stroke.Stroke{
		Points: []stroke.Point{
			{X: 120, Y: 200},
			{X: 120, Y: 50},
		},
	}

	fmt.Println("\n=== Тест 2: Противоположное направление (снизу вверх) ===")
	result2 := stroke.CompareStrokes(template.Strokes[0], drawnStrokeReversed, 1)
	fmt.Printf("Результат: angleDiff=%.1f°, similarity=%.3f, итого=%.3f\n",
		result2.AngleDiff, result2.Similarity, result2.OverallScore)
}
