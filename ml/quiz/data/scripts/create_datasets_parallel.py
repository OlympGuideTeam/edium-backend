"""
Параллельная генерация эталонов через DeepSeek.

Разбивает texts.jsonl на N слайсов и запускает N потоков.
Каждый поток пишет в свой файл и хранит свой прогресс — при обрыве
можно перезапустить скрипт и он продолжит с места остановки.

После завершения всех воркеров файлы сливаются в dataset.jsonl и lora_dataset.jsonl.

Использование:
    # Запустить все воркеры (по умолчанию 4):
    python create_datasets_parallel.py

    # Явно задать число воркеров:
    python create_datasets_parallel.py --workers 6

    # Запустить только один воркер (для ручного управления):
    python create_datasets_parallel.py --worker-id 2 --workers 4

    # Только слить уже готовые файлы без генерации:
    python create_datasets_parallel.py --merge-only
"""

import argparse
import json
import re
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

import requests

API_KEY = ""
URL = "https://api.deepseek.com/v1/chat/completions"

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
3. Для short_answer: "answer" — 1-3 слова (имя, дата, термин). ЗАПРЕЩЕНЫ предложения.
4. Для single_choice: "answer" должен БУКВА В БУКВУ совпадать с одним из элементов "options".
5. Для multiple_choice:
   - "answer" — ровно 2 или 3 элемента.
   - Каждый элемент "answer" должен БУКВА В БУКВУ совпадать с элементом из "options".
6. Варианты ответов "options" — правдоподобные, без явных подсказок.
7. Используй русский язык.
8. Верни ТОЛЬКО валидный JSON, начинающийся с символа {, без маркдауна и комментариев."""

MAX_SHORT_ANSWER = 2


# ── API ──────────────────────────────────────────────────────────────────────


def call_deepseek(messages, temperature=0.3, retries=4):
    for attempt in range(retries):
        try:
            r = requests.post(
                URL,
                headers={"Authorization": f"Bearer {API_KEY}", "Content-Type": "application/json"},
                json={"model": "deepseek-chat", "messages": messages, "temperature": temperature},
                timeout=90,
            )
            data = r.json()
            return data["choices"][0]["message"]["content"]
        except Exception as e:
            wait = 2**attempt
            print(f"  [API] retry {attempt + 1}/{retries}: {e}, wait {wait}s")
            time.sleep(wait)
    return None


# ── JSON helpers ─────────────────────────────────────────────────────────────


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
    return safe_json_load(text)


# ── Validation ───────────────────────────────────────────────────────────────


def validate_quiz(quiz):
    if not isinstance(quiz, dict) or "questions" not in quiz:
        return []

    valid = []
    for q in quiz.get("questions", []):
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

    # Не более MAX_SHORT_ANSWER вопросов с письменным ответом
    sa_count = 0
    filtered = []
    for q in valid:
        if q["type"] == "short_answer":
            sa_count += 1
            if sa_count > MAX_SHORT_ANSWER:
                continue
        filtered.append(q)

    return filtered


def is_good_sample(facts, questions):
    if len(facts) < 3 or len(questions) < 4:
        return False
    types = {q["type"] for q in questions}
    sc_count = sum(1 for q in questions if q["type"] == "single_choice")
    return "single_choice" in types and sc_count >= 2 and len(types) >= 2


# ── Worker ───────────────────────────────────────────────────────────────────


class Worker:
    def __init__(self, worker_id: int, lines: list[str], out_dir: Path):
        self.worker_id = worker_id
        self.lines = lines
        self.out_dir = out_dir

        self.dataset_path = out_dir / f"dataset_worker_{worker_id}.jsonl"
        self.lora_path = out_dir / f"lora_worker_{worker_id}.jsonl"
        self.progress_path = out_dir / f"progress_worker_{worker_id}.json"
        self.errors_path = out_dir / f"errors_worker_{worker_id}.log"

    def _load_progress(self) -> set:
        if self.progress_path.exists():
            data = json.loads(self.progress_path.read_text(encoding="utf-8"))
            return set(data.get("processed_ids", []))
        return set()

    def _save_progress(self, processed_ids: set):
        self.progress_path.write_text(
            json.dumps({"processed_ids": list(processed_ids)}, ensure_ascii=False),
            encoding="utf-8",
        )

    def _log_error(self, msg: str):
        with open(self.errors_path, "a", encoding="utf-8") as f:
            f.write(msg + "\n")

    def _generate_facts(self, text: str):
        messages = [
            {"role": "system", "content": EXTRACTION_SYSTEM},
            {"role": "user", "content": text},
        ]
        raw = call_deepseek(messages, temperature=0.3)
        if not raw:
            return None
        raw = clean_json_text(raw)
        return extract_json_array(raw)

    def _generate_quiz(self, facts) -> list:
        messages = [
            {"role": "system", "content": GENERATION_SYSTEM},
            {"role": "user", "content": f"Факты:\n{json.dumps(facts, ensure_ascii=False)}"},
        ]
        for attempt in range(3):
            raw = call_deepseek(messages, temperature=0.5)
            if not raw:
                print(f"  [W{self.worker_id}] Attempt {attempt + 1}: no response")
                continue
            raw = clean_json_text(raw)
            quiz = safe_json_load(raw)
            if not quiz:
                print(f"  [W{self.worker_id}] Attempt {attempt + 1}: bad JSON")
                continue
            questions = validate_quiz(quiz)
            if is_good_sample(facts, questions):
                return questions
            print(f"  [W{self.worker_id}] Attempt {attempt + 1}: {len(questions)} valid questions")
        return []

    def run(self) -> int:
        processed_ids = self._load_progress()
        total = len(self.lines)
        saved = 0

        print(f"[W{self.worker_id}] Start: {total} items, already done: {len(processed_ids)}")

        with (
            open(self.dataset_path, "a", encoding="utf-8") as ds_f,
            open(self.lora_path, "a", encoding="utf-8") as lora_f,
        ):
            for i, line in enumerate(self.lines):
                data = json.loads(line)
                item_id = data["id"]

                if item_id in processed_ids:
                    continue

                text = data["text"]
                print(f"[W{self.worker_id}] [{i + 1}/{total}] {item_id}")

                try:
                    facts = self._generate_facts(text)
                    if not facts or len(facts) < 3:
                        self._log_error(f"{item_id} bad facts ({len(facts) if facts else 0})")
                        continue

                    questions = self._generate_quiz(facts)
                    if not questions:
                        self._log_error(f"{item_id} bad quiz")
                        continue

                    sample = {"text": text, "facts": facts, "questions": questions}
                    ds_f.write(json.dumps(sample, ensure_ascii=False) + "\n")
                    ds_f.flush()

                    lora_sample = {
                        "messages": [
                            {"role": "system", "content": GENERATION_SYSTEM},
                            {"role": "user", "content": f"Факты:\n{json.dumps(facts, ensure_ascii=False)}"},
                            {"role": "assistant", "content": json.dumps({"questions": questions}, ensure_ascii=False)},
                        ]
                    }
                    lora_f.write(json.dumps(lora_sample, ensure_ascii=False) + "\n")
                    lora_f.flush()

                    processed_ids.add(item_id)
                    self._save_progress(processed_ids)
                    saved += 1

                    print(f"[W{self.worker_id}] ✓ saved (total: {len(processed_ids)})")
                    time.sleep(0.5)

                except Exception as e:
                    self._log_error(f"{item_id} exception {e}")
                    import traceback

                    print(f"[W{self.worker_id}] ✗ {e}")
                    traceback.print_exc()

        print(f"[W{self.worker_id}] Done. Saved {saved} new items.")
        return saved


# ── Merge ─────────────────────────────────────────────────────────────────────


def merge_outputs(out_dir: Path, num_workers: int, dataset_out: Path, lora_out: Path):
    """Сливает файлы всех воркеров в итоговые файлы."""
    print(f"\nMerging {num_workers} workers → {dataset_out.name}, {lora_out.name}")
    ds_count = 0
    lora_count = 0

    with open(dataset_out, "w", encoding="utf-8") as ds_f, open(lora_out, "w", encoding="utf-8") as lora_f:
        for wid in range(num_workers):
            ds_path = out_dir / f"dataset_worker_{wid}.jsonl"
            lora_path = out_dir / f"lora_worker_{wid}.jsonl"

            if ds_path.exists():
                with open(ds_path, encoding="utf-8") as f:
                    for line in f:
                        if line.strip():
                            ds_f.write(line)
                            ds_count += 1

            if lora_path.exists():
                with open(lora_path, encoding="utf-8") as f:
                    for line in f:
                        if line.strip():
                            lora_f.write(line)
                            lora_count += 1

    print(f"Merged: {ds_count} dataset samples, {lora_count} lora samples")


# ── Entry point ───────────────────────────────────────────────────────────────


_SCRIPTS_DIR = Path(__file__).resolve().parent
_DATA_DIR = _SCRIPTS_DIR / "foxford_data"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", default=str(_DATA_DIR / "texts.jsonl"))
    parser.add_argument("--out-dir", default=str(_DATA_DIR / "workers"))
    parser.add_argument("--dataset-out", default=str(_DATA_DIR / "dataset.jsonl"))
    parser.add_argument("--lora-out", default=str(_DATA_DIR / "lora_dataset.jsonl"))
    parser.add_argument("--workers", type=int, default=6, help="Число параллельных воркеров")
    parser.add_argument("--worker-id", type=int, default=None, help="Запустить только этот воркер (0-based)")
    parser.add_argument("--merge-only", action="store_true", help="Только слить готовые файлы воркеров")
    args = parser.parse_args()

    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    dataset_out = Path(args.dataset_out)
    lora_out = Path(args.lora_out)

    if args.merge_only:
        merge_outputs(out_dir, args.workers, dataset_out, lora_out)
        return

    # Читаем все строки
    with open(args.input, encoding="utf-8") as f:
        lines = [line for line in f if line.strip()]

    total = len(lines)
    print(f"Total texts: {total}, workers: {args.workers}")

    # Разбиваем на слайсы
    slice_size = (total + args.workers - 1) // args.workers
    slices = [lines[i * slice_size : (i + 1) * slice_size] for i in range(args.workers)]
    slices = [s for s in slices if s]  # убираем пустые
    actual_workers = len(slices)

    if args.worker_id is not None:
        # Один воркер
        wid = args.worker_id
        if wid >= actual_workers:
            print(f"Worker {wid} has no data (total slices: {actual_workers})")
            return
        Worker(wid, slices[wid], out_dir).run()
    else:
        # Все воркеры в параллельных потоках
        workers = [Worker(i, slices[i], out_dir) for i in range(actual_workers)]
        with ThreadPoolExecutor(max_workers=actual_workers) as executor:
            futures = {executor.submit(w.run): w.worker_id for w in workers}
            for future in as_completed(futures):
                wid = futures[future]
                try:
                    saved = future.result()
                    print(f"[W{wid}] Finished, saved {saved} items")
                except Exception as e:
                    print(f"[W{wid}] Failed with exception: {e}")

        merge_outputs(out_dir, actual_workers, dataset_out, lora_out)
        print("\nAll done!")


if __name__ == "__main__":
    main()
