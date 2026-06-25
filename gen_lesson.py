import json

# Список N5
n5 = "日一国人年大十二本中長出三時行見月後前生五間上東四今金九入学高円子外八六下来気小七山話女北午百書先名川千水半男西電校語土木聞食車何南万毎白天母火右読友左休父雨"

kanji_list = list(n5)

lessons = {}
lesson_num = 1

for i in range(0, len(kanji_list), 5):
    batch = kanji_list[i:i+5]

    steps = []

    # Практика для каждого иероглифа
    for k in batch:
        steps.append({
            "type": "practice",
            "kanji": k,
            "title": f"Научитесь писать {k}"
        })

    # Викторина на чтения
    steps.append({
        "type": "quiz_reading",
        "kanji": batch,
        "title": f"Угадайте чтения иероглифов: {', '.join(batch)}"
    })

    # Викторина на значения
    steps.append({
        "type": "quiz_meaning",
        "kanji": batch,
        "title": f"Угадайте значения: {', '.join(batch)}"
    })

    # Викторина на выбор иероглифа
    steps.append({
        "type": "quiz_character",
        "kanji": batch,
        "title": f"Выберите правильный иероглиф: {', '.join(batch)}"
    })

    # Финальный тест + предыдущие иероглифы (если есть)
    review_kanji = batch[:]
    if lesson_num > 1:
        prev = kanji_list[max(0, i-5):i]
        review_kanji = batch + prev[:2]  # добавляем 2 из предыдущего урока

    steps.append({
        "type": "quiz_mixed",
        "kanji": review_kanji,
        "title": f"Финальный тест: {', '.join(review_kanji)}"
    })

    lessons[str(lesson_num)] = {
        "title": f"N5 — Урок {lesson_num}",
        "description": f"Иероглифы: {', '.join(batch)}",
        "level": "N5",
        "kanji": batch,
        "steps": steps
    }

    lesson_num += 1

with open('lessons.json', 'w', encoding='utf-8') as f:
    json.dump(lessons, f, ensure_ascii=False, indent=2)

print(f"Сгенерировано {lesson_num-1} уроков")