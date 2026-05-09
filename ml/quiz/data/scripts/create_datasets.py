import json
import os
import re
import time

import requests

API_KEY = ""
URL = "https://api.deepseek.com/v1/chat/completions"

CACHE_FILE = "cache.json"
PROGRESS_FILE = "progress.json"

# Промпты синхронизированы с sphinx/app/pipeline.py
EXTRACTION_SYSTEM = """\
Извлеки 5-15 ключевых фактов из текста.

Формат — строгий JSON-массив:
[
  {"question": "...", "answer": "...", "type": "person | date | term | number | location | event | definition | process"}
]

ЖЕСТКИЕ ПРАВИЛА:
1. Только факты из текста.
2. Ответ (answer) должен быть УЛЬТРАКОРОТКИМ: максимум 1-3 слова. Это должно быть конкретное имя, год, место, число или термин.
3. НИКАКИХ предложений, описаний и причастных оборотов в answer.
   - ПЛОХО: "На территории, где находились хижины строителей"
   - ХОРОШО: "Возле Луксора"
   - ПЛОХО: "Золотая посмертная маска властелина Египта"
   - ХОРОШО: "Золотая маска"
4. Формулируй вопрос (question) так, чтобы на него можно было ответить одним-двумя словами.
5. Используй русский язык.
6. Выведи ТОЛЬКО валидный JSON, без вступительных слов и без разметки ```json."""

GENERATION_SYSTEM = """\
Создай образовательный квиз по фактам.

Формат — строгий JSON:
{
  "questions": [
    {"type": "single_choice", "question": "строка", "answer": "строка", "options": ["строка", "строка", "строка", "строка"]},
    {"type": "multiple_choice", "question": "строка", "answer": ["строка", "строка"], "options": ["строка", "строка", "строка", "строка", "строка"]},
    {"type": "short_answer", "question": "строка", "answer": "строка"}
  ]
}

ЖЕСТКИЕ ПРАВИЛА (ШТРАФ ЗА НАРУШЕНИЕ):
1. Сгенерируй РОВНО 6-8 вопросов. Не меньше 6, не больше 8.
2. ОБЯЗАТЕЛЬНОЕ РАСПРЕДЕЛЕНИЕ ТИПОВ:
   - single_choice: МИНИМУМ 3 вопроса (самый частый тип)
   - multiple_choice: МИНИМУМ 1 вопрос
   - short_answer: НЕ БОЛЕЕ 2 вопросов (редкий тип — только для имён, дат, терминов)
   Пример допустимого распределения: 4 single_choice + 2 multiple_choice + 2 short_answer.
   ЗАПРЕЩЕНО: более 2 short_answer подряд или более 2 short_answer всего.
3. Для short_answer: "answer" — 1-3 слова (имя, дата, термин). ЗАПРЕЩЕНЫ предложения. Если факт сложнее — делай single_choice.
4. Для single_choice: "answer" должен БУКВА В БУКВУ совпадать с одним из элементов "options".
5. Для multiple_choice:
   - "answer" — ровно 2 или 3 элемента.
   - Каждый элемент "answer" должен БУКВА В БУКВУ совпадать с элементом из "options".
6. Варианты ответов "options" — правдоподобные, без явных подсказок.
7. Используй русский язык.
8. Верни ТОЛЬКО валидный JSON, начинающийся с символа {, без маркдауна и комментариев."""

MAX_SHORT_ANSWER = 2  # не более 2 short_answer из 8 вопросов

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
    with open(PROGRESS_FILE, "w", encoding="utf-8") as f:
        json.dump({"processed_ids": list(processed_ids)}, f, ensure_ascii=False)


def load_progress():
    if os.path.exists(PROGRESS_FILE):
        with open(PROGRESS_FILE, encoding="utf-8") as f:
            data = json.load(f)
            return set(data.get("processed_ids", []))
    return set()


def _cache_key(messages, temperature):
    return f"{temperature}:{json.dumps(messages, ensure_ascii=False)}"


def cached_call(messages, temperature=0.3):
    key = _cache_key(messages, temperature)
    if key in CACHE:
        return CACHE[key]
    result = call_deepseek(messages, temperature)
    if result:
        CACHE[key] = result
    return result


# API


def call_deepseek(messages, temperature=0.3, retries=3):
    """messages — список {"role": ..., "content": ...}"""
    for _ in range(retries):
        try:
            r = requests.post(
                URL,
                headers={"Authorization": f"Bearer {API_KEY}", "Content-Type": "application/json"},
                json={
                    "model": "deepseek-chat",
                    "messages": messages,
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


def safe_json_load(s):
    try:
        return json.loads(s)
    except json.JSONDecodeError:
        return None


def clean_json_text(text):
    text = text.strip()
    if text.startswith("```"):
        text = text.split("```")[1]
        text = text.replace("json", "", 1).strip()
    return text


def extract_json_array(text):
    text = text.strip()
    match = re.search(r"\[.*]", text, re.DOTALL)
    if match:
        try:
            return json.loads(match.group())
        except json.JSONDecodeError:
            pass
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        pass
    return None


# PROMPTS


def messages_facts(text):
    return [
        {"role": "system", "content": EXTRACTION_SYSTEM},
        {"role": "user", "content": text},
    ]


def messages_quiz(facts):
    return [
        {"role": "system", "content": GENERATION_SYSTEM},
        {"role": "user", "content": f"Факты:\n{json.dumps(facts, ensure_ascii=False)}"},
    ]


# VALIDATION


def validate_quiz(quiz):
    """Возвращает список валидных вопросов, соответствующих sphinx-схеме."""
    if not isinstance(quiz, dict) or "questions" not in quiz:
        return []

    valid = []
    for q in quiz["questions"]:
        if "type" not in q or "question" not in q or "answer" not in q:
            continue

        t = q["type"]

        if t == "single_choice":
            if not isinstance(q["answer"], str):
                continue
            options = q.get("options")
            if not isinstance(options, list) or len(options) < 3:
                continue
            if q["answer"] not in options:
                continue

        elif t == "multiple_choice":
            if not isinstance(q["answer"], list) or len(q["answer"]) < 2:
                continue
            options = q.get("options")
            if not isinstance(options, list) or len(options) < 3:
                continue
            if not all(a in options for a in q["answer"]):
                continue
            q["answer"] = [str(a) for a in q["answer"]]

        elif t == "short_answer":
            if not isinstance(q["answer"], str):
                continue

        else:
            continue

        valid.append(q)

    # Ограничение: не более MAX_SHORT_ANSWER вопросов с письменным ответом
    sa_count = 0
    filtered = []
    for q in valid:
        if q["type"] == "short_answer":
            sa_count += 1
            if sa_count > MAX_SHORT_ANSWER:
                continue
        filtered.append(q)

    return filtered


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
    for attempt in range(max_retries):
        raw = cached_call(messages_quiz(facts), temperature=0.5)
        if not raw:
            print(f"  Attempt {attempt + 1}: No response from API")
            continue

        raw = clean_json_text(raw)
        quiz = safe_json_load(raw)

        if not quiz:
            print(f"  Attempt {attempt + 1}: Failed to parse JSON")
            continue

        questions = validate_quiz(quiz)

        if len(questions) >= 4:
            # Проверяем минимальное распределение типов
            types = {q["type"] for q in questions}
            sc_count = sum(1 for q in questions if q["type"] == "single_choice")
            if "single_choice" in types and sc_count >= 2 and len(types) >= 2:
                print(f"  Generated valid quiz with {len(questions)} questions")
                return questions

        print(f"  Attempt {attempt + 1}: Quiz validation failed ({len(questions)} valid questions)")

    print(f"  ✗ Failed to generate valid quiz after {max_retries} attempts")
    return []


# FILTER


def is_good_sample(facts, questions):
    if len(facts) < 3:
        return False
    if len(questions) < 4:
        return False
    types = {q["type"] for q in questions}
    return len(types) >= 2


# LOGGING


def log_error(msg):
    with open("errors.log", "a", encoding="utf-8") as f:
        f.write(msg + "\n")


# PIPELINE


def process_file(input_path, dataset_out, lora_out, start=0):
    processed_ids = load_progress()
    print(f"Already processed {len(processed_ids)} items")

    with open(input_path, encoding="utf-8") as f:
        lines = f.readlines()

    total = len(lines)

    with open(dataset_out, "a", encoding="utf-8") as dataset_file, open(lora_out, "a", encoding="utf-8") as lora_file:
        for i, line in enumerate(lines[start:], start=start):
            data = json.loads(line)
            item_id = data["id"]

            if item_id in processed_ids:
                print(f"\n[{i + 1}/{total}] Skipping {item_id} (already processed)")
                continue

            text = data["text"]
            print(f"\n[{i + 1}/{total}] Processing {item_id}")

            try:
                # === FACTS ===
                print("  Extracting facts...")
                facts_raw = cached_call(messages_facts(text), temperature=0.3)
                if not facts_raw:
                    log_error(f"{item_id} no facts response")
                    continue

                facts_raw = clean_json_text(facts_raw)
                facts = extract_json_array(facts_raw) or safe_json_load(facts_raw)

                if not facts or len(facts) < 3:
                    log_error(f"{item_id} bad facts (got {len(facts) if facts else 0})")
                    continue

                print(f"  ✓ Extracted {len(facts)} facts")

                # === QUIZ (options генерируются inline) ===
                print("  Generating quiz...")
                questions = generate_quiz_with_retry(facts, max_retries=3)

                if not questions:
                    log_error(f"{item_id} bad quiz after retries")
                    continue

                for q in questions:
                    normalize_question(q)

                if not is_good_sample(facts, questions):
                    log_error(f"{item_id} bad sample (facts: {len(facts)}, questions: {len(questions)})")
                    continue

                sample = {"text": text, "facts": facts, "questions": questions}

                # === SAVE DATASET ===
                dataset_file.write(json.dumps(sample, ensure_ascii=False) + "\n")
                dataset_file.flush()

                # === LORA FORMAT (generation task) ===
                lora_sample = {
                    "messages": [
                        {"role": "system", "content": GENERATION_SYSTEM},
                        {"role": "user", "content": f"Факты:\n{json.dumps(facts, ensure_ascii=False)}"},
                        {"role": "assistant", "content": json.dumps({"questions": questions}, ensure_ascii=False)},
                    ]
                }
                lora_file.write(json.dumps(lora_sample, ensure_ascii=False) + "\n")
                lora_file.flush()

                processed_ids.add(item_id)
                save_progress(processed_ids)
                print(f"  ✓ Saved (total processed: {len(processed_ids)})")

                if len(processed_ids) % 10 == 0:
                    save_cache()

                time.sleep(1)

            except Exception as e:
                log_error(f"{item_id} exception {e}")
                print(f"  ✗ Exception: {e}")
                import traceback

                traceback.print_exc()

    save_cache()
    print(f"\nProcessing complete! Processed {len(processed_ids)} items")


# ENTRYPOINT

if __name__ == "__main__":
    process_file(
        input_path="../foxford_data/texts.jsonl",
        dataset_out="../foxford_data/dataset.jsonl",
        lora_out="../foxford_data/lora_dataset.jsonl",
        start=0,
    )
