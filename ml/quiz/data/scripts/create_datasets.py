import json
import os
import random
import re
import time

import requests

API_KEY = ""
URL = "https://api.deepseek.com/v1/chat/completions"

CACHE_FILE = "cache.json"
PROGRESS_FILE = "progress.json"  # Файл для сохранения прогресса

# CACHE

if os.path.exists(CACHE_FILE):
    with open(CACHE_FILE, encoding="utf-8") as f:
        CACHE = json.load(f)
else:
    CACHE = {}


def save_cache():
    with open(CACHE_FILE, "w", encoding="utf-8") as f:
        json.dump(CACHE, f, ensure_ascii=False)


def save_progress(processed_ids):
    """Сохраняет список обработанных ID"""
    with open(PROGRESS_FILE, "w", encoding="utf-8") as f:
        json.dump({"processed_ids": processed_ids}, f, ensure_ascii=False)


def load_progress():
    """Загружает список обработанных ID"""
    if os.path.exists(PROGRESS_FILE):
        with open(PROGRESS_FILE, encoding="utf-8") as f:
            data = json.load(f)
            return set(data.get("processed_ids", []))
    return set()


def cached_call(prompt, temperature=0.3):
    key = f"{temperature}:{prompt}"

    if key in CACHE:
        return CACHE[key]

    result = call_deepseek(prompt, temperature)

    if result:
        CACHE[key] = result

    return result


# API


def call_deepseek(prompt, temperature=0.3, retries=3):
    for _ in range(retries):
        try:
            r = requests.post(
                URL,
                headers={"Authorization": f"Bearer {API_KEY}", "Content-Type": "application/json"},
                json={
                    "model": "deepseek-chat",
                    "messages": [{"role": "user", "content": prompt}],
                    "temperature": temperature,
                },
                timeout=60,
            )
            data = r.json()
            return data["choices"][0]["message"]["content"]
        except Exception as e:
            print("Retry:", e)
            time.sleep(2)
    return None


def best_of_n(prompt, n=3, temperature=0.5):
    results = []

    for _ in range(n):
        res = cached_call(prompt, temperature)
        if res:
            results.append(res)

    if not results:
        return None

    # простая эвристика — самый длинный ответ
    return max(results, key=len)


def safe_json_load(s):
    try:
        return json.loads(s)
    except json.JSONDecodeError:
        return None


def clean_json_text(text):
    text = text.strip()

    # убрать ```json ... ```
    if text.startswith("```"):
        text = text.split("```")[1]
        text = text.replace("json", "", 1).strip()

    return text


def extract_json_array(text):
    """Пытается извлечь JSON массив из текста"""
    text = text.strip()

    # Сначала пробуем найти массив в тексте
    match = re.search(r"\[.*]", text, re.DOTALL)
    if match:
        try:
            return json.loads(match.group())
        except json.JSONDecodeError:
            pass

    # Если не нашли, пробуем распарсить весь текст
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        pass

    # Пробуем найти массивы, которые могут быть разделены переносами строк
    lines = text.split("\n")
    for line in lines:
        line = line.strip()
        if line.startswith("[") and line.endswith("]"):
            try:
                return json.loads(line)
            except json.JSONDecodeError:
                continue

    return None


# PROMPTS


def prompt_facts(text):
    return f"""
{text}

Извлеки 5-15 фактов.

Формат JSON:
[
  {{
    "question": "...",
    "answer": "...",
    "type": "person | date | term | number | location | event | definition | process"
  }}
]

Правила:
- только факты из текста
- короткие ответы
- без объяснений
"""


def prompt_quiz(facts):
    return f"""
Факты:
{json.dumps(facts, ensure_ascii=False)}

Создай квиз.

ТРЕБОВАНИЯ:
- 5-10 вопросов
- используй типы (необязательно все):
  - single_choice
  - multiple_choice
  - short_answer

СТРОГИЙ ФОРМАТ JSON (без комментариев, без markdown):

{{
  "questions": [
    {{
      "type": "single_choice",
      "question": "строка",
      "answer": "строка"
    }},
    {{
      "type": "multiple_choice",
      "question": "строка",
      "answer": ["строка", "строка"]
    }},
    {{
      "type": "short_answer",
      "question": "строка",
      "answer": "строка"
    }}
  ]
}}

ПРАВИЛА ДЛЯ MULTIPLE_CHOICE:
- answer должен быть МАССИВОМ из 2-3 правильных ответов
- Каждый правильный ответ - ЭТО ЦЕЛАЯ ФРАЗА, а не её часть
- НЕ разбивай один правильный ответ на несколько элементов массива
- Пример ХОРОШЕГО ответа: ["Многократное совпадение во времени условного и безусловного раздражителя", "условный раздражитель должен предшествовать безусловному"]
- Пример ПЛОХОГО ответа: ["Многократное совпадение", "во времени условного и безусловного раздражителя"] (это одна мысль, разбитая на части)
- НЕ добавляй точку в конце каждого ответа

ЖЁСТКИЕ ПРАВИЛА:
- НЕ используй поля: text, correct_answer, id, options
- ТОЛЬКО: type, question, answer (для multiple_choice answer - массив строк)
- JSON должен начинаться с {{ и заканчиваться }}
- без ```json
- answer должен быть точным фактом из предоставленных данных
- не добавляй лишний текст
- Для short_answer ответ должен быть кратким (1-5 слов)

Верни ТОЛЬКО JSON.
"""


def prompt_distractors(answer, text):
    return f"""
Контекст:
{text}

Правильный ответ: "{answer}"

Сгенерируй 3 неправильных, но правдоподобных ответа на основе контекста.
Эти ответы должны быть:
- Правдоподобными (могут встретиться в теме)
- Но неправильными с точки зрения фактов
- Краткими (1-5 слов)
- Разными по смыслу

ВАЖНО: Верни ТОЛЬКО JSON массив из 3 строк.
НЕ добавляй никакой текст до или после массива.
НЕ используй markdown форматирование.

Пример правильного ответа:
["неправильный ответ 1", "неправильный ответ 2", "неправильный ответ 3"]

Твой ответ (только JSON):
"""


# VALIDATION


def validate_quiz(quiz):
    """Возвращает список валидных вопросов или пустой список"""
    if not isinstance(quiz, dict) or "questions" not in quiz:
        return []

    valid = []

    for q in quiz["questions"]:
        if "type" not in q or "question" not in q or "answer" not in q:
            continue

        t = q["type"]

        # Проверка для multiple_choice - answer должен быть списком
        if t == "multiple_choice":
            if not isinstance(q["answer"], list) or len(q["answer"]) < 1:
                continue
            # Преобразуем в список строк
            q["answer"] = [str(a) for a in q["answer"]]

        # Проверка для остальных типов - answer должен быть строкой
        elif t in ["single_choice", "short_answer"]:
            if not isinstance(q["answer"], str):
                continue
            q["answer"] = str(q["answer"])
        else:
            continue

        valid.append(q)

    return valid


def normalize_question(q):
    t = q["type"].lower()

    if "single" in t:
        q["type"] = "single_choice"
    elif "multiple" in t:
        q["type"] = "multiple_choice"
    else:
        q["type"] = "short_answer"

    return q


def generate_quiz_with_retry(facts, max_retries=3):
    """Генерирует квиз с повторными попытками при пустом результате"""
    for attempt in range(max_retries):
        quiz_raw = best_of_n(prompt_quiz(facts), n=1)
        if not quiz_raw:
            print(f"  Attempt {attempt + 1}: No response from API")
            continue

        quiz_raw = clean_json_text(quiz_raw)
        quiz = safe_json_load(quiz_raw)

        if not quiz:
            print(f"  Attempt {attempt + 1}: Failed to parse JSON")
            continue

        questions = validate_quiz(quiz)

        if len(questions) >= 3:
            print(f"Generated valid quiz with {len(questions)} questions")
            return questions
        else:
            print(f"  Attempt {attempt + 1}: Quiz validation failed ({len(questions)} valid questions)")

    print(f"  ✗ Failed to generate valid quiz after {max_retries} attempts")
    return []


# DISTRACTORS


def generate_distractors(answer, text):
    """Генерирует дистракторы для ответа"""
    # Пробуем сгенерировать дистракторы несколько раз
    for attempt in range(3):
        res = best_of_n(prompt_distractors(answer, text), n=1, temperature=0.8)
        if not res:
            print(f"      Attempt {attempt + 1}: No response")
            continue

        # Очищаем текст от markdown и лишних символов
        res = clean_json_text(res)

        # Удаляем возможные обратные кавычки и пробелы
        res = res.strip()
        if res.startswith('"') and res.endswith('"'):
            # Если ответ в кавычках, пробуем распарсить как JSON строку
            try:
                res = json.loads(res)
            except json.JSONDecodeError:
                pass

        # Пробуем извлечь JSON массив
        distractors = extract_json_array(res)

        if isinstance(distractors, list) and len(distractors) > 0:
            # Очищаем дистракторы
            distractors = [str(d).strip() for d in distractors if d and str(d).strip()]
            # Убираем дубликаты и правильный ответ
            distractors = list(set(distractors) - {answer})

            if len(distractors) >= 3:
                return distractors[:3]
            elif len(distractors) > 0:
                print(f"      Generated {len(distractors)} distractors, need 3")
                continue
            else:
                print(f"      Attempt {attempt + 1}: Empty or duplicate distractors")
                continue
        else:
            print(f"      Attempt {attempt + 1}: Failed to parse JSON array")
            # Пробуем другой подход - если модель вернула строки через запятую
            if "," in res and not res.startswith("["):
                parts = [p.strip().strip('"') for p in res.split(",")]
                if len(parts) >= 3:
                    print(f"      Found comma-separated values: {parts[:3]}")
                    return parts[:3]
            continue

    print("    ✗ Failed to generate distractors")
    return []


def build_options(answer, distractors):
    """Собирает опции из правильного ответа и дистракторов"""
    # Для single_choice
    if isinstance(answer, str):
        options = [answer] + distractors
        # Если дистракторов меньше 3, добавляем заглушки
        while len(options) < 4:
            options.append(f"Вариант {len(options)}")
        random.shuffle(options)
        return options

    # Для multiple_choice (answer - список)
    elif isinstance(answer, list):
        all_options = answer + distractors
        # Убираем дубликаты
        all_options = list(set(all_options))
        # Если мало опций, добавляем заглушки
        while len(all_options) < 4:
            all_options.append(f"Вариант {len(all_options)}")
        random.shuffle(all_options)
        return all_options

    return []


# FILTER


def is_good_sample(facts, questions):
    if len(facts) < 3:
        return False

    if len(questions) < 3:
        return False

    types = set(q["type"] for q in questions)
    return len(types) >= 2


# LOGGING


def log_error(msg):
    with open("errors.log", "a", encoding="utf-8") as f:
        f.write(msg + "\n")


# PIPELINE


def process_file(input_path, dataset_out, lora_out, start=0):
    # Загружаем прогресс
    processed_ids = load_progress()
    print(f"Already processed {len(processed_ids)} items")

    with open(input_path, encoding="utf-8") as f:
        lines = f.readlines()

    total = len(lines)

    # Открываем файлы в режиме добавления
    with open(dataset_out, "a", encoding="utf-8") as dataset_file, open(lora_out, "a", encoding="utf-8") as lora_file:
        for i, line in enumerate(lines[start:], start=start):
            data = json.loads(line)
            item_id = data["id"]

            # Пропускаем уже обработанные
            if item_id in processed_ids:
                print(f"\n[{i + 1}/{total}] Skipping {item_id} (already processed)")
                continue

            text = data["text"]

            print(f"\n[{i + 1}/{total}] Processing {item_id}")

            try:
                # === FACTS ===
                print("  Extracting facts...")
                facts_raw = best_of_n(prompt_facts(text), n=1)
                if not facts_raw:
                    log_error(f"{item_id} no facts response")
                    continue

                facts_raw = clean_json_text(facts_raw)
                facts = safe_json_load(facts_raw)

                if not facts or len(facts) < 3:
                    log_error(f"{item_id} bad facts (got {len(facts) if facts else 0})")
                    continue

                print(f"  ✓ Extracted {len(facts)} facts")

                # === QUIZ WITH RETRY ===
                print("  Generating quiz...")
                questions = generate_quiz_with_retry(facts, max_retries=3)

                if not questions:
                    log_error(f"{item_id} bad quiz after retries")
                    continue

                # === ADD DISTRACTORS ONLY FOR CHOICE TYPES ===
                print("  Adding distractors...")
                for idx, q in enumerate(questions):
                    q = normalize_question(q)

                    # Генерируем дистракторы ТОЛЬКО для choice типов
                    if q["type"] in ["single_choice", "multiple_choice"]:
                        # Для multiple_choice answer - это список
                        if isinstance(q["answer"], list):
                            # Генерируем дистракторы для каждого правильного ответа?
                            # Пока генерируем на основе первого
                            answer_for_distractors = q["answer"][0] if q["answer"] else ""
                            if answer_for_distractors:
                                distractors = generate_distractors(answer_for_distractors, text)
                                q["options"] = build_options(q["answer"], distractors)
                            else:
                                q["options"] = q["answer"]
                        else:
                            distractors = generate_distractors(q["answer"], text)
                            q["options"] = build_options(q["answer"], distractors)

                        print(f"    Question {idx + 1} ({q['type']}): options count = {len(q['options'])}")
                    else:
                        # Для short_answer не нужны опции
                        print(f"    Question {idx + 1} ({q['type']}): no options needed")

                if not is_good_sample(facts, questions):
                    log_error(f"{item_id} bad sample (facts: {len(facts)}, questions: {len(questions)})")
                    continue

                sample = {"text": text, "facts": facts, "questions": questions}

                # === SAVE DATASET ===
                dataset_file.write(json.dumps(sample, ensure_ascii=False) + "\n")
                dataset_file.flush()  # Сразу сохраняем на диск

                # === LORA FORMAT ===
                lora_sample = {
                    "messages": [
                        {"role": "system", "content": "Ты создаёшь образовательные квизы"},
                        {"role": "user", "content": f"Факты:\n{json.dumps(facts, ensure_ascii=False)}"},
                        {"role": "assistant", "content": json.dumps({"questions": questions}, ensure_ascii=False)},
                    ]
                }

                lora_file.write(json.dumps(lora_sample, ensure_ascii=False) + "\n")
                lora_file.flush()  # Сразу сохраняем на диск

                # Сохраняем прогресс
                processed_ids.add(item_id)
                save_progress(list(processed_ids))

                print(f"  ✓ Successfully saved sample (total processed: {len(processed_ids)})")

                # сохраняем кэш иногда
                if len(processed_ids) % 10 == 0:
                    save_cache()

                time.sleep(1)

            except Exception as e:
                log_error(f"{item_id} exception {e}")
                print(f"  ✗ Exception: {e}")
                import traceback

                traceback.print_exc()
                # Не прерываем программу, продолжаем со следующим

    save_cache()
    print(f"\nProcessing complete! Processed {len(processed_ids)} items")


# ENTRYPOINT

if __name__ == "__main__":
    process_file(
        input_path="../foxford_data/texts.jsonl",
        dataset_out="../foxford_data/dataset.jsonl",
        lora_out="../foxford_data/lora_dataset.jsonl",
        start=0,  # можно менять для продолжения
    )
