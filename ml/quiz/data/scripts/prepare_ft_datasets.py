"""
Конвертация dataset.jsonl → два датасета для fine-tuning:
  1. extraction (text → facts)
  2. generation (facts → quiz questions)

Запуск:
    cd ml/quiz
    python data/scripts/prepare_ft_datasets.py
"""

import json
import random
from pathlib import Path

DATASET_PATH = Path(__file__).resolve().parent.parent / "foxford_data" / "dataset.jsonl"
OUTPUT_DIR = Path(__file__).resolve().parent.parent / "datasets"
VAL_RATIO = 0.1
SEED = 42
MIN_FACTS = 3
MIN_QUESTIONS = 3

EXTRACTION_SYSTEM = (
    "Ты — AI-ассистент для извлечения фактов из учебных текстов. "
    "Извлеки ключевые факты в формате JSON-массива. Каждый факт содержит поля: "
    "question (вопрос по факту), answer (краткий ответ), type (один из: "
    "person, date, term, number, location, event, definition, process)."
)

GENERATION_SYSTEM = (
    "Ты — AI-ассистент для создания образовательных квизов. "
    "На основе списка фактов создай квиз в формате JSON с полем questions. "
    "Типы вопросов: single_choice (с options), multiple_choice (answer — массив, с options), "
    "short_answer (без options, краткий ответ)."
)


def load_dataset(path: Path) -> list[dict]:
    samples = []
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                d = json.loads(line)
                samples.append(d)
            except json.JSONDecodeError:
                continue
    return samples


def is_valid(sample: dict) -> bool:
    facts = sample.get("facts", [])
    questions = sample.get("questions", [])
    if len(facts) < MIN_FACTS or len(questions) < MIN_QUESTIONS:
        return False
    if not sample.get("text", "").strip():
        return False
    return True


def make_extraction_sample(sample: dict) -> dict:
    return {
        "messages": [
            {"role": "system", "content": EXTRACTION_SYSTEM},
            {"role": "user", "content": sample["text"]},
            {"role": "assistant", "content": json.dumps(sample["facts"], ensure_ascii=False)},
        ]
    }


def make_generation_sample(sample: dict) -> dict:
    facts_str = json.dumps(sample["facts"], ensure_ascii=False)
    questions_str = json.dumps({"questions": sample["questions"]}, ensure_ascii=False)
    return {
        "messages": [
            {"role": "system", "content": GENERATION_SYSTEM},
            {"role": "user", "content": f"Факты:\n{facts_str}"},
            {"role": "assistant", "content": questions_str},
        ]
    }


def save_jsonl(data: list[dict], path: Path):
    with open(path, "w", encoding="utf-8") as f:
        for item in data:
            f.write(json.dumps(item, ensure_ascii=False) + "\n")


def main():
    if not DATASET_PATH.exists():
        print(f"Файл не найден: {DATASET_PATH}")
        print("Сначала запустите create_datasets.py")
        return

    raw = load_dataset(DATASET_PATH)
    print(f"Загружено сэмплов: {len(raw)}")

    valid = [s for s in raw if is_valid(s)]
    print(f"Валидных сэмплов: {len(valid)} (отброшено {len(raw) - len(valid)})")

    random.seed(SEED)
    random.shuffle(valid)

    split_idx = int(len(valid) * (1 - VAL_RATIO))
    train_samples = valid[:split_idx]
    val_samples = valid[split_idx:]
    print(f"Train: {len(train_samples)}, Val: {len(val_samples)}")

    extraction_train = [make_extraction_sample(s) for s in train_samples]
    extraction_val = [make_extraction_sample(s) for s in val_samples]
    generation_train = [make_generation_sample(s) for s in train_samples]
    generation_val = [make_generation_sample(s) for s in val_samples]

    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    save_jsonl(extraction_train, OUTPUT_DIR / "extraction_train.jsonl")
    save_jsonl(extraction_val, OUTPUT_DIR / "extraction_val.jsonl")
    save_jsonl(generation_train, OUTPUT_DIR / "generation_train.jsonl")
    save_jsonl(generation_val, OUTPUT_DIR / "generation_val.jsonl")

    print(f"\nСохранено в {OUTPUT_DIR}/:")
    print(f"  extraction_train.jsonl  ({len(extraction_train)} сэмплов)")
    print(f"  extraction_val.jsonl    ({len(extraction_val)} сэмплов)")
    print(f"  generation_train.jsonl  ({len(generation_train)} сэмплов)")
    print(f"  generation_val.jsonl    ({len(generation_val)} сэмплов)")


if __name__ == "__main__":
    main()
