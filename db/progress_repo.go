package db

import (
	"database/sql"
	"time"
)

type UserProgress struct {
	ID                 int        `json:"id"`
	UserID             int        `json:"user_id"`
	KanjiID            int        `json:"kanji_id"`
	Status             string     `json:"status"` // new, learning, reviewing, mastered
	SRSLevel           int        `json:"srs_level"`
	NextReviewDate     *time.Time `json:"next_review_date,omitempty"`
	MistakesCount      int        `json:"mistakes_count"`
	LastWrittenQuality float64    `json:"last_written_quality"`
	LastPracticedAt    *time.Time `json:"last_practiced_at,omitempty"`
}

// GetOrCreateProgress возвращает прогресс или создаёт новый
func GetOrCreateProgress(userID, kanjiID int) (*UserProgress, error) {
	query := "SELECT id, user_id, kanji_id, status, srs_level, next_review_date, mistakes_count, last_written_quality, last_practiced_at FROM user_progress WHERE user_id = ? AND kanji_id = ?"
	var p UserProgress
	err := DB.QueryRow(query, userID, kanjiID).Scan(
		&p.ID, &p.UserID, &p.KanjiID, &p.Status, &p.SRSLevel,
		&p.NextReviewDate, &p.MistakesCount, &p.LastWrittenQuality, &p.LastPracticedAt,
	)
	if err == nil {
		return &p, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Создаём новый
	insert := "INSERT INTO user_progress (user_id, kanji_id, status, srs_level) VALUES (?, ?, 'new', 0)"
	result, err := DB.Exec(insert, userID, kanjiID)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &UserProgress{
		ID:      int(id),
		UserID:  userID,
		KanjiID: kanjiID,
		Status:  "new",
	}, nil
}

// UpdateProgress обновляет прогресс после практики
func UpdateProgress(userID, kanjiID int, quality float64) error {
	now := time.Now()
	query := `
		UPDATE user_progress 
		SET last_written_quality = ?, last_practiced_at = ?, status = 'learning',
		    srs_level = CASE WHEN ? >= 0.8 THEN MIN(srs_level + 1, 5) ELSE MAX(srs_level - 1, 0) END,
		    mistakes_count = CASE WHEN ? < 0.7 THEN mistakes_count + 1 ELSE mistakes_count END,
		    next_review_date = ?
		WHERE user_id = ? AND kanji_id = ?`

	// Простой SRS: интервал зависит от уровня
	intervals := []int{1, 3, 7, 14, 30, 60} // дни
	srsLevel := 0
	p, _ := GetOrCreateProgress(userID, kanjiID)
	if p != nil {
		srsLevel = p.SRSLevel
	}
	if quality >= 0.8 && srsLevel < 5 {
		srsLevel++
	} else if quality < 0.7 && srsLevel > 0 {
		srsLevel--
	}
	nextReview := now.AddDate(0, 0, intervals[srsLevel])

	_, err := DB.Exec(query, quality, now, quality, quality, nextReview, userID, kanjiID)
	return err
}

// GetDueReviews возвращает иероглифы для повторения
func GetDueReviews(userID int) ([]UserProgress, error) {
	query := "SELECT id, user_id, kanji_id, status, srs_level, next_review_date, mistakes_count, last_written_quality, last_practiced_at FROM user_progress WHERE user_id = ? AND next_review_date <= datetime('now') ORDER BY next_review_date ASC"
	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []UserProgress
	for rows.Next() {
		var p UserProgress
		rows.Scan(&p.ID, &p.UserID, &p.KanjiID, &p.Status, &p.SRSLevel, &p.NextReviewDate, &p.MistakesCount, &p.LastWrittenQuality, &p.LastPracticedAt)
		result = append(result, p)
	}
	return result, nil
}

// SavePracticeHistory сохраняет запись о практике
func SavePracticeHistory(userID, kanjiID, strokeIndex int, score, dtw, angle, length float64) error {
	query := "INSERT INTO practice_history (user_id, kanji_id, stroke_index, score, dtw_distance, angle_diff, length_diff) VALUES (?, ?, ?, ?, ?, ?, ?)"
	_, err := DB.Exec(query, userID, kanjiID, strokeIndex, score, dtw, angle, length)
	return err
}
