import json
import os
from pathlib import Path


def clean_text(text):
    text = text.replace("\u00a0", " ")
    while "\n\n" in text:
        text = text.replace("\n\n", "\n")

    return text.strip()


MIN_LENGTH = 1000
MAX_LENGTH = 4000


def collect_texts(root_dir, output_file):
    root_path = Path(root_dir)
    processed_count = 0
    skipped_count = 0

    with open(output_file, "w", encoding="utf-8") as out:
        for file_path in root_path.rglob("*.txt"):
            if file_path.is_file():
                with open(file_path, encoding="utf-8") as f:
                    text = f.read().strip()

                cleaned_text = clean_text(text)
                length = len(cleaned_text)

                if length < MIN_LENGTH or length > MAX_LENGTH:
                    skipped_count += 1
                    continue

                relative_path = file_path.relative_to(root_path)
                file_id = str(relative_path).replace(os.sep, "_")

                record = {
                    "id": file_id,
                    "text": cleaned_text,
                    "length": length,
                }

                out.write(json.dumps(record, ensure_ascii=False) + "\n")
                processed_count += 1

    return processed_count, skipped_count


if __name__ == "__main__":
    processed, skipped = collect_texts("../foxford_data", "../foxford_data/texts.jsonl")
    print(
        f"Итог: собрано {processed} текстовых файлов, пропущено {skipped} (вне диапазона {MIN_LENGTH}–{MAX_LENGTH} симв.)"
    )
