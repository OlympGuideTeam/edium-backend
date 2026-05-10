import json
import os
import re
from pathlib import Path

# Пути относительно этого файла — работают из любой рабочей директории
SCRIPTS_DIR = Path(__file__).resolve().parent
DATA_DIR = SCRIPTS_DIR / "foxford_data"

MIN_LENGTH = 1000
MAX_LENGTH = 4000


def clean_text(text: str) -> str:
    # NBSP и другие «пробельные» Unicode-символы → обычный пробел
    text = text.replace(" ", " ")
    text = text.replace("​", "")  # zero-width space
    text = text.replace("⁠", "")  # word joiner
    # Несколько пробелов подряд → один
    text = re.sub(r" {2,}", " ", text)
    # Более двух переносов строк → два
    text = re.sub(r"\n{3,}", "\n\n", text)
    return text.strip()


def split_into_chunks(text: str, min_len: int, max_len: int) -> list[str]:
    """
    Разбивает текст на чанки длиной [min_len, max_len] по абзацам.
    Если один абзац длиннее max_len — режет по предложениям.
    """
    paragraphs = [p.strip() for p in re.split(r"\n\n+", text) if p.strip()]

    chunks: list[str] = []
    current_parts: list[str] = []
    current_len = 0

    for para in paragraphs:
        # Абзац сам по себе длиннее потолка — дробим по предложениям
        if len(para) > max_len:
            sentences = re.split(r"(?<=[.!?])\s+", para)
            for sent in sentences:
                if current_len + len(sent) + 1 > max_len and current_len >= min_len:
                    chunks.append("\n\n".join(current_parts))
                    current_parts, current_len = [], 0
                current_parts.append(sent)
                current_len += len(sent) + 1
        else:
            if current_len + len(para) + 2 > max_len and current_len >= min_len:
                chunks.append("\n\n".join(current_parts))
                current_parts, current_len = [], 0
            current_parts.append(para)
            current_len += len(para) + 2

    if current_parts and current_len >= min_len:
        chunks.append("\n\n".join(current_parts))

    return chunks


def collect_texts(root_dir: Path, output_file: Path):
    root_path = Path(root_dir)
    output_file = Path(output_file)
    output_file.parent.mkdir(parents=True, exist_ok=True)

    processed_count = 0
    skipped_count = 0

    with open(output_file, "w", encoding="utf-8") as out:
        for file_path in sorted(root_path.rglob("*.txt")):
            if not file_path.is_file():
                continue

            with open(file_path, encoding="utf-8") as f:
                text = f.read()

            cleaned = clean_text(text)

            # Короткий — пропускаем
            if len(cleaned) < MIN_LENGTH:
                skipped_count += 1
                continue

            # В диапазоне — сохраняем как есть
            if len(cleaned) <= MAX_LENGTH:
                chunks = [cleaned]
            else:
                # Длинный — разбиваем на чанки
                chunks = split_into_chunks(cleaned, MIN_LENGTH, MAX_LENGTH)
                if not chunks:
                    skipped_count += 1
                    continue

            relative_path = file_path.relative_to(root_path)
            base_id = str(relative_path).replace(os.sep, "_")

            for idx, chunk in enumerate(chunks):
                file_id = base_id if len(chunks) == 1 else f"{base_id}_part{idx:02d}"
                record = {
                    "id": file_id,
                    "text": chunk,
                    "length": len(chunk),
                }
                out.write(json.dumps(record, ensure_ascii=False) + "\n")
                processed_count += 1

    return processed_count, skipped_count


if __name__ == "__main__":
    output = DATA_DIR / "texts.jsonl"
    output.unlink(missing_ok=True)

    processed, skipped = collect_texts(DATA_DIR, output)
    print(f"Итог: собрано {processed} чанков, пропущено {skipped} файлов (< {MIN_LENGTH} симв.)")
