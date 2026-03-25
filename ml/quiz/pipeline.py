import fitz  # PyMuPDF
import json
import os
import io
import re
import ast
import argparse
from PIL import Image
from openai import OpenAI


def _load_env_files():
    """Подхватывает .env из ml/ и корня репозитория (без зависимости python-dotenv)."""
    base = os.path.dirname(os.path.abspath(__file__))
    for root in (base, os.path.join(base, ".."), os.path.join(base, "..", "..")):
        path = os.path.join(root, ".env")
        if not os.path.isfile(path):
            continue
        try:
            with open(path, encoding="utf-8") as ef:
                for line in ef:
                    line = line.strip()
                    if not line or line.startswith("#") or "=" not in line:
                        continue
                    key, _, val = line.partition("=")
                    key, val = key.strip(), val.strip().strip('"').strip("'")
                    if key and key not in os.environ:
                        os.environ[key] = val
        except OSError:
            pass


_load_env_files()
API_KEY = os.environ.get("OPENAI_API_KEY", "")
MODEL = "gpt-4o-mini"

OUTPUT_FILE = "dataset.jsonl"

client = OpenAI(api_key=API_KEY)

def _parse_page_token(token) -> tuple[int, int]:
    """
    Одна ячейка расписания: число 19 или строка "19", "6-7", " 80-82 " → (start, end) включительно.
    """
    if isinstance(token, int):
        return (token, token)
    s = str(token).strip().replace("—", "-")
    if "-" in s:
        a, b = s.split("-", 1)
        return (int(a.strip()), int(b.strip()))
    n = int(s.strip())
    return (n, n)


def parse_chapter_segments(txt_filename: str) -> list[tuple[int, int | None]]:
    """
    Читает chapters-файл и возвращает список (start_page, end_page), 1-based, концы включительно.
    Для последнего блока в формате «только стартовые страницы» end_page = None → до конца PDF.

    Форматы:
    - [4, 8, 12, ...] — старт каждого параграфа; конец = страница перед следующим стартом.
    - ["6-7", "8-9", "19", ...] — явные диапазоны по каждому параграфу.
    """
    if not os.path.exists(txt_filename):
        print(f"Нет файла chapters: {txt_filename}")
        return []
    with open(txt_filename, "r", encoding="utf-8") as f:
        raw = f.read().strip()
    if not raw:
        return []
    try:
        data = ast.literal_eval(raw)
    except Exception:
        print(f"Не удалось разобрать chapters: {txt_filename}")
        return []
    if not isinstance(data, list) or not data:
        return []

    if all(isinstance(x, int) for x in data):
        starts = sorted(data)
        out: list[tuple[int, int | None]] = []
        for i, start_p in enumerate(starts):
            if i + 1 < len(starts):
                end_p = starts[i + 1] - 1
            else:
                end_p = None
            out.append((start_p, end_p))
        return out

    if all(isinstance(x, str) for x in data):
        return [_parse_page_token(x) for x in data]

    # Смешанный список (строки и числа)
    if all(isinstance(x, (int, str)) for x in data):
        return [_parse_page_token(x) for x in data]

    print(f"Неизвестный формат chapters (ожидали int или str): {txt_filename}")
    return []


def init_surya():
    """Инициализирует предикторы Surya OCR (один раз при запуске)."""
    from surya.foundation import FoundationPredictor
    from surya.recognition import RecognitionPredictor
    from surya.detection import DetectionPredictor

    print("Загрузка моделей Surya OCR...")
    foundation_predictor = FoundationPredictor()
    recognition_predictor = RecognitionPredictor(foundation_predictor)
    detection_predictor = DetectionPredictor()
    print("Модели Surya загружены.")
    return recognition_predictor, detection_predictor


# По умолчанию не более 2048px по длинной стороне — иначе возможен torch.AcceleratorError у Surya
DEFAULT_MAX_IMAGE_SIDE = 2048


def pdf_page_to_pil(page, dpi=300, max_side=DEFAULT_MAX_IMAGE_SIDE):
    """Конвертирует страницу fitz в PIL Image, ограничивая размер для Surya."""
    zoom = dpi / 72
    mat = fitz.Matrix(zoom, zoom)
    pix = page.get_pixmap(matrix=mat)
    img_data = pix.tobytes("png")
    image = Image.open(io.BytesIO(img_data)).convert("RGB")
    w, h = image.size
    if max(w, h) > max_side:
        ratio = max_side / max(w, h)
        new_w, new_h = int(w * ratio), int(h * ratio)
        image = image.resize((new_w, new_h), Image.Resampling.LANCZOS)
    return image


def clean_pdf_layer_text(text: str) -> str:
    """Убирает теги вроде <b>, <br> из текстового слоя PDF."""
    if not text:
        return ""
    t = re.sub(r"(?i)<br\s*/?>", "\n", text)
    t = re.sub(r"<[^>]+>", " ", t)
    t = re.sub(r"[ \t]+", " ", t)
    t = re.sub(r"\n{3,}", "\n\n", t)
    return t.strip()


def _surya_or_tesseract_page(
    page,
    page_num_1based: int,
    recognition_predictor,
    detection_predictor,
    ocr_dpi: int,
    max_image_side: int,
) -> str:
    image = pdf_page_to_pil(page, dpi=ocr_dpi, max_side=max_image_side)
    try:
        predictions = recognition_predictor([image], det_predictor=detection_predictor)
        if predictions and hasattr(predictions[0], "text_lines"):
            lines = [line.text for line in predictions[0].text_lines]
            return "\n".join(lines).strip()
    except Exception as e:
        print(f"      [WARN] Surya на стр. {page_num_1based}: {e!r}, fallback на Tesseract")
        tp = page.get_textpage_ocr(language="rus+eng", flags=fitz.TEXT_PRESERVE_WHITESPACE)
        return page.get_text("text", textpage=tp).strip()
    return ""


def _is_cyrillic(ch):
    """Символ — кириллица (русский и др.)."""
    return "\u0400" <= ch <= "\u04FF"


def _looks_like_valid_russian(text):
    """Проверяет, что в тексте достаточно слов из нормальной кириллицы (не кракозябры)."""
    if not text or len(text.strip()) < 50:
        return True
    words = text.split()
    if len(words) < 10:
        return True
    valid = 0
    for w in words:
        w = w.strip(".,;:!?()\"'-")
        if len(w) >= 2 and all(_is_cyrillic(c) for c in w):
            valid += 1
    ratio = valid / len(words)
    return ratio >= 0.25


def _cyrillic_word_ratio(text: str) -> float:
    words = text.split()
    if not words:
        return 0.0
    valid = 0
    for w in words:
        w = w.strip(".,;:!?()\"'-")
        if len(w) >= 2 and all(_is_cyrillic(c) for c in w):
            valid += 1
    return valid / len(words)


def _extraction_quality_score(text: str) -> float:
    """Выше — лучше для сравнения слоя и OCR (доля кириллических слов + объём)."""
    t = text.strip()
    if len(t) < 15:
        return 0.0
    return _cyrillic_word_ratio(t) * 3.0 + min(len(t), 80000) / 20000.0


def _try_fix_encoding(raw: str) -> str | None:
    """
    Пытается передекодировать текст слоя PDF.
    Частый случай: в PDF текст в Windows-1251, а шрифт отдаёт те же байты как Latin-1
    (коды 0–255 → U+0000–U+00FF). Тогда raw.encode("latin-1").decode("cp1251") даёт русский.
    Если результат валидный русский — возвращаем его, иначе None.
    """
    if not raw or not raw.strip():
        return None
    try:
        # Только если все символы в диапазоне 0–255 (байты, показанные как Latin-1)
        if any(ord(c) > 255 for c in raw):
            return None
        decoded = raw.encode("latin-1").decode("cp1251")
        if _looks_like_valid_russian(decoded):
            return decoded
    except (UnicodeDecodeError, UnicodeEncodeError):
        pass
    try:
        # Обратный вариант: могли сохранить как cp1252
        if any(ord(c) > 255 for c in raw):
            return None
        decoded = raw.encode("latin-1").decode("cp1252")
        if _looks_like_valid_russian(decoded):
            return decoded
    except (UnicodeDecodeError, UnicodeEncodeError):
        pass
    return None


def _is_cjk_char(ch):
    code = ord(ch)
    return (
        0x3040 <= code <= 0x309F
        or 0x30A0 <= code <= 0x30FF
        or 0x4E00 <= code <= 0x9FFF
        or 0xAC00 <= code <= 0xD7AF
    )


def clean_ocr_text(text):
    """
    Убирает из текста OCR-мусор: строки с CJK-символами и колонтитулы tilda.
    """
    if not text or not text.strip():
        return text
    out_lines = []
    for line in text.splitlines():
        line = line.strip()
        if not line:
            continue
        if "tilda" in line.lower() and ("material" in line.lower() or "скачан" in line.lower()):
            continue
        letters = [c for c in line if c.isalpha()]
        if letters:
            cjk_ratio = sum(1 for c in letters if _is_cjk_char(c)) / len(letters)
            if cjk_ratio > 0.5:
                continue
        out_lines.append(line)
    return "\n".join(out_lines)


def _layer_text_from_page(page) -> str:
    """Текстовый слой: исправление cp1251/cp1252, очистка HTML."""
    raw = page.get_text("text").strip()
    if not raw:
        return ""
    cleaned = clean_pdf_layer_text(raw)
    if cleaned and not _looks_like_valid_russian(cleaned):
        fixed = _try_fix_encoding(raw)
        if fixed is not None:
            cleaned = clean_pdf_layer_text(fixed)
        else:
            cleaned = ""
    return cleaned


def extract_text_from_pages(
    doc,
    start_page,
    end_page,
    recognition_predictor,
    detection_predictor,
    *,
    extract_mode: str = "auto",
    ocr_dpi: int = 300,
    max_image_side: int = DEFAULT_MAX_IMAGE_SIDE,
):
    """
    Извлекает текст из диапазона страниц PDF.
    start_page и end_page — номера страниц (1-based, включительно).

    extract_mode:
      auto    — слой PDF, если похож на нормальный русский, иначе Surya/Tesseract (как раньше).
      ocr_only — всегда Surya (или Tesseract при сбое); максимально единообразно для сканов.
      best    — и слой, и OCR; выбирается вариант с лучшей оценкой (кириллица + длина). Медленнее.
    """
    text_parts = []
    for p_num in range(start_page - 1, end_page):
        if p_num >= len(doc):
            continue
        page = doc[p_num]
        page_1 = p_num + 1

        if extract_mode == "ocr_only":
            page_text = _surya_or_tesseract_page(
                page, page_1, recognition_predictor, detection_predictor, ocr_dpi, max_image_side
            )
        elif extract_mode == "best":
            layer = _layer_text_from_page(page)
            ocr = _surya_or_tesseract_page(
                page, page_1, recognition_predictor, detection_predictor, ocr_dpi, max_image_side
            )
            sl = _extraction_quality_score(layer)
            so = _extraction_quality_score(ocr)
            if sl <= 0 and so <= 0:
                page_text = ""
            elif so >= sl:
                page_text = ocr
            else:
                page_text = layer
        else:
            page_text = _layer_text_from_page(page)
            if not page_text:
                page_text = _surya_or_tesseract_page(
                    page, page_1, recognition_predictor, detection_predictor, ocr_dpi, max_image_side
                )

        if page_text:
            page_text = clean_ocr_text(page_text)
            if page_text.strip():
                text_parts.append(page_text)

    return "\n\n".join(text_parts)


def generate_quiz_from_text(chapter_text):
    system_prompt = """Ты профессиональный методист. Тебе дан текст параграфа из учебника.
Твоя задача — составить по нему квиз в формате JSON.

Структура JSON (список):
[
  {"type": "single_choice", "question": "...", "options": ["..."], "correct_answ": "..."},
  {"type": "multiple_choice", "question": "...", "options": ["..."], "correct_answ": ["..."]}
]
Сделай 3-5 вопросов. Игнорируй колонтитулы и номера страниц.
Верни ТОЛЬКО валидный JSON."""
    try:
        response = client.chat.completions.create(
            model=MODEL,
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": f"Вот текст параграфа:\n\n{chapter_text}"}
            ],
            response_format={"type": "json_object"},
            max_tokens=2000
        )
        return json.loads(response.choices[0].message.content)
    except Exception as e:
        print(f"Ошибка API: {e}")
        return None


def iter_books(books_dir: str, only_names: set[str] | None = None):
    """Ищет все *.pdf в папке и возвращает (book_name, pdf_path, chapters_path)."""
    for fname in sorted(os.listdir(books_dir)):
        if not fname.lower().endswith(".pdf"):
            continue
        pdf_path = os.path.join(books_dir, fname)
        book_name = os.path.splitext(fname)[0]
        if only_names is not None and book_name not in only_names:
            continue
        chapters_path = os.path.join(books_dir, f"{book_name}_chapters.txt")
        yield book_name, pdf_path, chapters_path


# ================= MAIN =================

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--books_dir", required=True, help="Папка с учебниками: <book>.pdf и <book>_chapters.txt")
    parser.add_argument("--output", default=OUTPUT_FILE, help="Куда писать dataset.jsonl")
    parser.add_argument(
        "--books",
        default="",
        help="Через запятую имена книг без .pdf (например: biology_10,social_8). Пусто = все PDF в папке.",
    )
    parser.add_argument(
        "--extract_mode",
        choices=("auto", "ocr_only", "best"),
        default="auto",
        help="Качество извлечения: auto (слой или OCR), ocr_only (всегда Surya), best (слой+OCR, лучший по оценке).",
    )
    parser.add_argument(
        "--ocr_dpi",
        type=int,
        default=300,
        help="DPI рендера страницы для OCR. Выше (например 400) — детальнее, но тяжелее; затем всё равно ужимается до --max_image_side.",
    )
    parser.add_argument(
        "--max_image_side",
        type=int,
        default=DEFAULT_MAX_IMAGE_SIDE,
        help="Макс. сторона картинки для Surya (px). 2048 по умолчанию; 2560 может дать чуть лучше OCR, но риск падений Surya.",
    )
    args = parser.parse_args()
    only = None
    if args.books.strip():
        only = {x.strip() for x in args.books.split(",") if x.strip()}

    if not API_KEY:
        print("Не задан OPENAI_API_KEY. Установите переменную окружения.")
        return

    if not os.path.isdir(args.books_dir):
        print(f"Папка не найдена: {args.books_dir}")
        return

    print(
        f"Извлечение: mode={args.extract_mode}, ocr_dpi={args.ocr_dpi}, max_image_side={args.max_image_side}"
    )

    # Surya OCR один раз на все книги
    recognition_predictor, detection_predictor = init_surya()

    with open(args.output, "a", encoding="utf-8") as f_out:
        found_any = False

        for book_name, pdf_path, chapters_path in iter_books(args.books_dir, only):
            found_any = True

            if not os.path.exists(chapters_path):
                print(f"[SKIP] {book_name}: нет файла глав {chapters_path}")
                continue

            segments = parse_chapter_segments(chapters_path)
            if not segments:
                print(f"[SKIP] {book_name}: chapters пустой/не прочитан")
                continue

            print(f"\n=== Книга: {book_name} ===")
            doc = fitz.open(pdf_path)
            try:
                n_doc = len(doc)
                print(f"PDF: {n_doc} стр. Параграфов: {len(segments)}")

                for i, (start_p, end_p_opt) in enumerate(segments):
                    end_p = end_p_opt if end_p_opt is not None else n_doc
                    end_p = min(end_p, n_doc)
                    start_p = min(max(1, start_p), n_doc)
                    if end_p < start_p:
                        print(f"   [SKIP] Параграф {i + 1}: неверный диапазон {start_p}-{end_p}")
                        continue

                    print(f"\n{book_name} — Параграф {i + 1} (Стр {start_p}-{end_p})...")

                    chapter_text = extract_text_from_pages(
                        doc,
                        start_p,
                        end_p,
                        recognition_predictor,
                        detection_predictor,
                        extract_mode=args.extract_mode,
                        ocr_dpi=args.ocr_dpi,
                        max_image_side=args.max_image_side,
                    )

                    if not chapter_text.strip():
                        print("   [SKIP] Не удалось извлечь текст.")
                        continue

                    print(f"   -> Извлечено {len(chapter_text)} символов текста. Отправка в GPT...")

                    quiz = generate_quiz_from_text(chapter_text)

                    if quiz:
                        questions = []
                        if isinstance(quiz, list):
                            questions = quiz
                        elif isinstance(quiz, dict):
                            for v in quiz.values():
                                if isinstance(v, list):
                                    questions = v
                                    break

                        if questions:
                            record = {
                                "book": book_name,
                                "pages": f"{start_p}-{end_p}",
                                "input_text": chapter_text,
                                "output_quiz": questions
                            }
                            f_out.write(json.dumps(record, ensure_ascii=False) + "\n")
                            print(f"      [OK] Получено {len(questions)} вопросов.")
                        else:
                            print("      [WARN] Пустой ответ.")
                    else:
                        print("      [FAIL] Ошибка генерации.")
            finally:
                doc.close()

        if not found_any:
            print(f"В папке {args.books_dir} не найдено ни одного PDF.")


if __name__ == "__main__":
    main()
