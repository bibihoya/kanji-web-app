import json

# Загружаем скачанный файл
with open('model/kanji_dict.json', 'r', encoding='utf-8') as f:
    source_dict = json.load(f)

# Конвертируем в наш формат
converted = {}

for kanji, data in source_dict.items():
    # Определяем уровень JLPT
    jlpt_new = data.get('jlpt_new')
    jlpt_level = ''
    if jlpt_new is not None:
        jlpt_level = f'N{jlpt_new}'

    converted[kanji] = {
        'onyomi': data.get('readings_on', []),
        'kunyomi': data.get('readings_kun', []),
        'meaning': ', '.join(data.get('meanings', [])[:5]),
        'jlpt': jlpt_level,
        'stroke_count': data.get('strokes', 0),
        'words': []
    }

# Сохраняем результат
with open('model/kanji_dict.json', 'w', encoding='utf-8') as f:
    json.dump(converted, f, ensure_ascii=False, indent=2)

print(f'Конвертировано {len(converted)} иероглифов')