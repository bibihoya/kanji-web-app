package stroke

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type svgRoot struct {
	XMLName xml.Name   `xml:"svg"`
	Groups  []svgGroup `xml:"g"`
	Paths   []svgPath  `xml:"path"`
}

type svgGroup struct {
	ID      string     `xml:"id,attr"`
	Element string     `xml:"kvg:element,attr"`
	Paths   []svgPath  `xml:"path"`
	Groups  []svgGroup `xml:"g"`
}

type svgPath struct {
	Data string `xml:"d,attr"`
	Type string `xml:"kvg:type,attr"`
}

func LoadKanjiVG(filePath string, kanjiChar string) (*KanjiTemplate, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл %s: %w", filePath, err)
	}

	var root svgRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("не удалось распарсить SVG: %w", filePath, err)
	}

	var strokes []Stroke
	strokeOrder := 0

	// Мощная рекурсивная функция
	var dive func(groups []svgGroup, depth int)
	dive = func(groups []svgGroup, depth int) {
		for _, g := range groups {
			// Сначала обрабатываем path в этой группе
			for _, p := range g.Paths {
				pts, err := parseSVGPathData(p.Data)
				if err == nil && len(pts) >= 2 {
					strokeOrder++
					strokes = append(strokes, Stroke{
						Points: pts,
						Order:  strokeOrder,
					})
					fmt.Printf("DEBUG: черта %d, тип=%s, точек=%d\n", strokeOrder, p.Type, len(pts))
				}
			}
			// Затем погружаемся во вложенные группы
			if len(g.Groups) > 0 {
				dive(g.Groups, depth+1)
			}
		}
	}

	// Запускаем погружение с верхнего уровня
	dive(root.Groups, 0)

	// Обрабатываем path, которые могут быть прямо в <svg>
	for _, p := range root.Paths {
		pts, err := parseSVGPathData(p.Data)
		if err == nil && len(pts) >= 2 {
			strokeOrder++
			strokes = append(strokes, Stroke{
				Points: pts,
				Order:  strokeOrder,
			})
		}
	}

	if len(strokes) == 0 {
		// Печатаем структуру для диагностики
		fmt.Printf("DEBUG: групп верхнего уровня: %d\n", len(root.Groups))
		printGroupStructure(root.Groups, 0)
		return nil, fmt.Errorf("не найдено ни одной черты в файле %s", filePath)
	}

	fmt.Printf("DEBUG: всего извлечено %d черт\n", len(strokes))
	return &KanjiTemplate{
		Char:    kanjiChar,
		Strokes: strokes,
	}, nil
}

func printGroupStructure(groups []svgGroup, depth int) {
	for _, g := range groups {
		indent := strings.Repeat("  ", depth)
		fmt.Printf("%s- group id=%s element=%s paths=%d groups=%d\n",
			indent, g.ID, g.Element, len(g.Paths), len(g.Groups))
		if len(g.Groups) > 0 {
			printGroupStructure(g.Groups, depth+1)
		}
	}
}

// parseSVGPathData разбирает SVG-путь и возвращает слайс точек.
// Поддерживает команды: M, L, C, Q, H, V, Z (и их относительные версии).
func parseSVGPathData(data string) ([]Point, error) {
	var points []Point
	var curX, curY, startX, startY float64

	// Разбиваем строку на команды и их аргументы
	tokens := tokenizeSVGPath(data)

	var cmd byte = 'M' // Текущая команда
	var prevCmd byte = 'M'

	i := 0
	for i < len(tokens) {
		token := tokens[i]

		// Проверяем, является ли токен командой
		if isSVGCommand(token) {
			cmd = token[0]
			if cmd >= 'a' && cmd <= 'z' {
				cmd -= 32 // Приводим к верхнему регистру
			}
			i++
			continue
		}

		// Обрабатываем аргументы
		switch cmd {
		case 'M', 'L':
			if i+1 < len(tokens) {
				x, y, advance, err := parseNextCoord(tokens, i, curX, curY, cmd)
				if err != nil {
					i++
					continue
				}
				curX, curY = x, y
				points = append(points, Point{X: x, Y: y})
				if cmd == 'M' {
					startX, startY = x, y
					cmd = 'L' // После M неявно следует L
				}
				prevCmd = 'L'
				i += advance
				continue
			}
			i++

		case 'H':
			if i < len(tokens) {
				val, err := parseFloatToken(tokens[i])
				if err == nil {
					if cmd == 'h' {
						curX += val
					} else {
						curX = val
					}
					points = append(points, Point{X: curX, Y: curY})
					prevCmd = 'H'
					i++
					continue
				}
			}
			i++

		case 'V':
			if i < len(tokens) {
				val, err := parseFloatToken(tokens[i])
				if err == nil {
					if cmd == 'v' {
						curY += val
					} else {
						curY = val
					}
					points = append(points, Point{X: curX, Y: curY})
					prevCmd = 'V'
					i++
					continue
				}
			}
			i++

		case 'C':
			if i+5 < len(tokens) {
				x1, e1 := parseFloatToken(tokens[i])
				y1, e2 := parseFloatToken(tokens[i+1])
				x2, e3 := parseFloatToken(tokens[i+2])
				y2, e4 := parseFloatToken(tokens[i+3])
				x, e5 := parseFloatToken(tokens[i+4])
				y, e6 := parseFloatToken(tokens[i+5])

				if e1 == nil && e2 == nil && e3 == nil && e4 == nil && e5 == nil && e6 == nil {
					// Для относительных координат преобразуем
					if cmd == 'c' {
						x1 += curX
						y1 += curY
						x2 += curX
						y2 += curY
						x += curX
						y += curY
					}
					// Интерполируем кривую Безье (20 шагов)
					const steps = 20
					for s := 1; s <= steps; s++ {
						t := float64(s) / float64(steps)
						t1 := 1 - t
						px := t1*t1*t1*curX + 3*t1*t1*t*x1 + 3*t1*t*t*x2 + t*t*t*x
						py := t1*t1*t1*curY + 3*t1*t1*t*y1 + 3*t1*t*t*y2 + t*t*t*y
						points = append(points, Point{X: px, Y: py})
					}
					curX, curY = x, y
					prevCmd = 'C'
					i += 6
					continue
				}
			}
			i++

		case 'Q':
			if i+3 < len(tokens) {
				x1, e1 := parseFloatToken(tokens[i])
				y1, e2 := parseFloatToken(tokens[i+1])
				x, e3 := parseFloatToken(tokens[i+2])
				y, e4 := parseFloatToken(tokens[i+3])

				if e1 == nil && e2 == nil && e3 == nil && e4 == nil {
					if cmd == 'q' {
						x1 += curX
						y1 += curY
						x += curX
						y += curY
					}
					const steps = 20
					for s := 1; s <= steps; s++ {
						t := float64(s) / float64(steps)
						t1 := 1 - t
						px := t1*t1*curX + 2*t1*t*x1 + t*t*x
						py := t1*t1*curY + 2*t1*t*y1 + t*t*y
						points = append(points, Point{X: px, Y: py})
					}
					curX, curY = x, y
					prevCmd = 'Q'
					i += 4
					continue
				}
			}
			i++

		case 'Z', 'z':
			curX, curY = startX, startY
			points = append(points, Point{X: curX, Y: curY})
			cmd = prevCmd
			i++

		default:
			i++
		}
	}

	if len(points) < 2 {
		return nil, fmt.Errorf("недостаточно точек: %d", len(points))
	}

	return points, nil
}

// tokenizeSVGPath разбивает строку пути на токены (команды и числа)
func tokenizeSVGPath(data string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range data {
		switch {
		case r == ' ' || r == ',' || r == '\t' || r == '\n' || r == '\r':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(r))
		case r == '-':
			// Минус может быть началом числа или разделителем
			if current.Len() > 0 && current.String() != "-" {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			current.WriteRune(r)
		case r == '.' || (r >= '0' && r <= '9') || r == 'e' || r == 'E':
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// isSVGCommand возвращает true, если токен — команда SVG
func isSVGCommand(s string) bool {
	if len(s) != 1 {
		return false
	}
	c := s[0]
	return strings.ContainsRune("MmLlCcQqZzHhVv", rune(c))
}

// parseNextCoord извлекает следующую пару координат (x, y)
func parseNextCoord(tokens []string, i int, curX, curY float64, cmd byte) (float64, float64, int, error) {
	x, err1 := parseFloatToken(tokens[i])
	y, err2 := parseFloatToken(tokens[i+1])
	if err1 != nil || err2 != nil {
		return 0, 0, 0, fmt.Errorf("некорректные координаты")
	}
	if cmd == 'm' || cmd == 'l' {
		x += curX
		y += curY
	}
	return x, y, 2, nil
}

func parseFloatToken(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("пустой токен")
	}
	return strconv.ParseFloat(s, 64)
}
