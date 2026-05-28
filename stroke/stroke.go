package stroke

import (
	"math"
)

// Point — точка касания/траектории
type Point struct {
	X, Y float64
}

// DistanceTo вычисляет евклидово расстояние до другой точки
func (p Point) DistanceTo(other Point) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// Stroke — одна черта иероглифа
type Stroke struct {
	Points []Point // Траектория черты
	Order  int     // Порядковый номер (начиная с 1)
}

// Normalize приводит траекторию к единому масштабу и центрирует
func (s *Stroke) Normalize() {
	if len(s.Points) == 0 {
		return
	}

	// Находим ограничивающий прямоугольник
	minX, minY := s.Points[0].X, s.Points[0].Y
	maxX, maxY := s.Points[0].X, s.Points[0].Y
	for _, p := range s.Points {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	width := maxX - minX
	height := maxY - minY
	scale := math.Max(width, height)
	if scale == 0 {
		scale = 1
	}

	// Центрируем и масштабируем
	cx := (minX + maxX) / 2
	cy := (minY + maxY) / 2
	for i := range s.Points {
		s.Points[i].X = (s.Points[i].X - cx) / scale
		s.Points[i].Y = (s.Points[i].Y - cy) / scale
	}
}

// DirectionAngle возвращает угол направления от первой до последней точки (в радианах)
func (s *Stroke) DirectionAngle() float64 {
	if len(s.Points) < 2 {
		return 0
	}
	first := s.Points[0]
	last := s.Points[len(s.Points)-1]
	return math.Atan2(last.Y-first.Y, last.X-first.X)
}

// Length возвращает общую длину траектории
func (s *Stroke) Length() float64 {
	total := 0.0
	for i := 1; i < len(s.Points); i++ {
		total += s.Points[i].DistanceTo(s.Points[i-1])
	}
	return total
}

// KanjiTemplate — эталон иероглифа
type KanjiTemplate struct {
	Char    string   // Сам иероглиф
	Strokes []Stroke // Эталонные черты в правильном порядке
}

// StrokeResult — результат проверки одной черты
type StrokeResult struct {
	Order        int     // Номер черты
	DTWScore     float64 // Сырая дистанция DTW (чем меньше, тем лучше)
	Similarity   float64 // Похожесть 0..1 (1 = идеально)
	AngleDiff    float64 // Разница углов в градусах
	LengthDiff   float64 // Относительная разница длин (0 = одинаковые)
	OverallScore float64 // Общая оценка 0..1
}

// AnalysisResult — полный результат проверки
type AnalysisResult struct {
	Char          string         // Какой иероглиф проверяли
	StrokeResults []StrokeResult // Результаты по каждой черте
	OrderCorrect  bool           // Правильный ли порядок
	OverallScore  float64        // Общая оценка 0..1
	Feedback      string         // Текстовый отзыв
}

// centerPoints смещает все точки так, чтобы центр масс оказался в (0,0)
func (s *Stroke) centerPoints() {
	if len(s.Points) == 0 {
		return
	}

	var cx, cy float64
	for _, p := range s.Points {
		cx += p.X
		cy += p.Y
	}
	cx /= float64(len(s.Points))
	cy /= float64(len(s.Points))

	for i := range s.Points {
		s.Points[i].X -= cx
		s.Points[i].Y -= cy
	}
}

// normalizeSize масштабирует траекторию так, чтобы максимальный размер стал 1.0
func (s *Stroke) normalizeSize() {
	if len(s.Points) == 0 {
		return
	}

	var maxDist float64
	for _, p := range s.Points {
		dist := math.Sqrt(p.X*p.X + p.Y*p.Y)
		if dist > maxDist {
			maxDist = dist
		}
	}

	if maxDist > 0 {
		for i := range s.Points {
			s.Points[i].X /= maxDist
			s.Points[i].Y /= maxDist
		}
	}
}

// Rotate поворачивает все точки на заданный угол (в радианах)
func (s *Stroke) Rotate(angle float64) {
    cos := math.Cos(angle)
    sin := math.Sin(angle)
    for i := range s.Points {
        x := s.Points[i].X
        y := s.Points[i].Y
        s.Points[i].X = x*cos - y*sin
        s.Points[i].Y = x*sin + y*cos
    }
}