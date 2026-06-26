-- Пользователи
CREATE TABLE IF NOT EXISTS users (
                                     id INTEGER PRIMARY KEY AUTOINCREMENT,
                                     username TEXT UNIQUE NOT NULL DEFAULT 'default',
                                     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Иероглифы
CREATE TABLE IF NOT EXISTS kanji_characters (
                                                id INTEGER PRIMARY KEY AUTOINCREMENT,
                                                character TEXT UNIQUE NOT NULL,
                                                onyomi TEXT DEFAULT '[]',
                                                kunyomi TEXT DEFAULT '[]',
                                                meaning TEXT NOT NULL DEFAULT '',
                                                jlpt_level TEXT DEFAULT '',
                                                stroke_count INTEGER DEFAULT 0,
                                                grade INTEGER DEFAULT 0,
                                                frequency INTEGER DEFAULT 0,
                                                svg_filename TEXT DEFAULT ''
);

-- Слова-примеры
CREATE TABLE IF NOT EXISTS vocabulary (
                                          id INTEGER PRIMARY KEY AUTOINCREMENT,
                                          kanji_id INTEGER NOT NULL REFERENCES kanji_characters(id),
    word TEXT NOT NULL,
    reading TEXT NOT NULL,
    meaning TEXT NOT NULL DEFAULT '',
    jlpt_level TEXT DEFAULT '',
    is_common BOOLEAN DEFAULT 1
    );

-- Уроки
CREATE TABLE IF NOT EXISTS lessons (
                                       id INTEGER PRIMARY KEY AUTOINCREMENT,
                                       lesson_number INTEGER NOT NULL,
                                       title TEXT NOT NULL,
                                       description TEXT DEFAULT '',
                                       jlpt_level TEXT NOT NULL,
                                       kanji_ids TEXT NOT NULL DEFAULT '[]'
);

-- Прогресс пользователя
CREATE TABLE IF NOT EXISTS user_progress (
                                             id INTEGER PRIMARY KEY AUTOINCREMENT,
                                             user_id INTEGER NOT NULL REFERENCES users(id),
    kanji_id INTEGER NOT NULL REFERENCES kanji_characters(id),
    status TEXT DEFAULT 'new',
    srs_level INTEGER DEFAULT 0,
    next_review_date TIMESTAMP,
    mistakes_count INTEGER DEFAULT 0,
    last_written_quality REAL DEFAULT 0,
    last_practiced_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, kanji_id)
    );

-- История практики
CREATE TABLE IF NOT EXISTS practice_history (
                                                id INTEGER PRIMARY KEY AUTOINCREMENT,
                                                user_id INTEGER NOT NULL REFERENCES users(id),
    kanji_id INTEGER NOT NULL REFERENCES kanji_characters(id),
    stroke_index INTEGER NOT NULL DEFAULT 0,
    score REAL NOT NULL DEFAULT 0,
    dtw_distance REAL DEFAULT 0,
    angle_diff REAL DEFAULT 0,
    length_diff REAL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

-- Индексы
CREATE INDEX IF NOT EXISTS idx_kanji_jlpt ON kanji_characters(jlpt_level);
CREATE INDEX IF NOT EXISTS idx_kanji_char ON kanji_characters(character);
CREATE INDEX IF NOT EXISTS idx_progress_user ON user_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_progress_kanji ON user_progress(kanji_id);
CREATE INDEX IF NOT EXISTS idx_progress_review ON user_progress(next_review_date);
CREATE INDEX IF NOT EXISTS idx_vocab_kanji ON vocabulary(kanji_id);
CREATE INDEX IF NOT EXISTS idx_history_user ON practice_history(user_id);