import json
import os
from pathlib import Path


def clean_text(text):
    text = text.replace("\u00a0", " ")
    while "\n\n" in text:
        text = text.replace("\n\n", "\n")

    return text.strip()


def collect_texts(root_dir, output_file):
    root_path = Path(root_dir)
    processed_count = 0

    with open(output_file, "w", encoding="utf-8") as out:
        for file_path in root_path.rglob("*.txt"):
            if file_path.is_file():
                with open(file_path, encoding="utf-8") as f:
                    text = f.read().strip()

                # Очищаем текст
                cleaned_text = clean_text(text)

                # Создаём уникальный ID
                relative_path = file_path.relative_to(root_path)
                file_id = str(relative_path).replace(os.sep, "_")

                record = {
                    "id": file_id,
                    "text": cleaned_text,
                    "length": len(cleaned_text),
                }

                out.write(json.dumps(record, ensure_ascii=False) + "\n")
                processed_count += 1

    return processed_count


if __name__ == "__main__":
    processed = collect_texts("../foxford_data", "../foxford_data/texts.jsonl")
    print(f"Итог: собрано {processed} текстовых файлов")
