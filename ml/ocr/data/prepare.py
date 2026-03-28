"""
Подготовка данных для обучения OCR.

Экспортирует страницы из PDF в PNG и получает эталонный текст двумя способами:
  1. GPT-4o Vision — точный ground truth для recognition-модели.
  2. Surya bbox — авторазметка регионов для detection-модели (YOLO-формат).

Запуск:
    python -m ocr.data.prepare --books_dir ../books --output_dir ../../data/recognition --pages_per_book 30
    python -m ocr.data.prepare --books_dir ../books --output_dir ../../data/detection --mode detection --pages_per_book 30
"""

from __future__ import annotations

import argparse
import ast
import base64
import io
import json
import os
import random
import sys

import fitz  # PyMuPDF
from PIL import Image

# ---------------------------------------------------------------------------
# .env loader
# ---------------------------------------------------------------------------


def _load_env():
    base = os.path.dirname(os.path.abspath(__file__))
    for root in (base, os.path.join(base, ".."), os.path.join(base, "..", ".."), os.path.join(base, "..", "..", "..")):
        path = os.path.join(root, ".env")
        if not os.path.isfile(path):
            continue
        with open(path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                key, _, val = line.partition("=")
                key, val = key.strip(), val.strip().strip('"').strip("'")
                if key and key not in os.environ:
                    os.environ[key] = val


_load_env()


# ---------------------------------------------------------------------------
# PDF утилиты
# ---------------------------------------------------------------------------


def _page_to_pil(page: fitz.Page, dpi: int = 300, max_side: int = 2048) -> Image.Image:
    zoom = dpi / 72
    pix = page.get_pixmap(matrix=fitz.Matrix(zoom, zoom))
    img = Image.open(io.BytesIO(pix.tobytes("png"))).convert("RGB")
    w, h = img.size
    if max(w, h) > max_side:
        ratio = max_side / max(w, h)
        img = img.resize((int(w * ratio), int(h * ratio)), Image.Resampling.LANCZOS)
    return img


def _iter_books(books_dir: str):
    """Возвращает (book_name, pdf_path) для каждого PDF в папке."""
    for fname in sorted(os.listdir(books_dir)):
        if fname.lower().endswith(".pdf"):
            yield os.path.splitext(fname)[0], os.path.join(books_dir, fname)


def _parse_chapters(books_dir: str, book_name: str) -> list[int]:
    """Возвращает страницы из chapters-файла или пустой список."""
    path = os.path.join(books_dir, f"{book_name}_chapters.txt")
    if not os.path.isfile(path):
        return []
    with open(path, encoding="utf-8") as f:
        raw = f.read().strip()
    if not raw:
        return []
    try:
        data = ast.literal_eval(raw)
    except Exception:
        return []
    pages: list[int] = []
    for item in data:
        if isinstance(item, int):
            pages.append(item)
        elif isinstance(item, str):
            item = item.strip()
            if "-" in item:
                a, b = item.split("-", 1)
                try:
                    pages.extend(range(int(a), int(b) + 1))
                except ValueError:
                    pass
            else:
                try:
                    pages.append(int(item))
                except ValueError:
                    pass
    return sorted(set(pages))


# ---------------------------------------------------------------------------
# GPT-4o Vision — ground truth для recognition
# ---------------------------------------------------------------------------


def _pil_to_base64(img: Image.Image) -> str:
    buf = io.BytesIO()
    img.save(buf, format="PNG")
    return base64.b64encode(buf.getvalue()).decode()


def _get_ground_truth_gpt(img: Image.Image, client) -> str:
    """Отправляет изображение страницы в GPT-4o Vision, получает точный текст."""
    b64 = _pil_to_base64(img)
    response = client.chat.completions.create(
        model="gpt-4o",
        messages=[
            {
                "role": "system",
                "content": (
                    "Ты OCR-эксперт. Точно перепиши ВЕСЬ текст со страницы учебника. "
                    "Сохраняй абзацы и порядок строк. "
                    "Игнорируй колонтитулы (номер страницы, название главы). "
                    "Не добавляй никаких пояснений — только текст."
                ),
            },
            {
                "role": "user",
                "content": [
                    {"type": "image_url", "image_url": {"url": f"data:image/png;base64,{b64}"}},
                    {"type": "text", "text": "Перепиши текст страницы."},
                ],
            },
        ],
        max_tokens=4096,
    )
    return response.choices[0].message.content.strip()


# ---------------------------------------------------------------------------
# Surya авторазметка — bboxes для detection
# ---------------------------------------------------------------------------


def _get_surya_bboxes(img: Image.Image) -> list[dict]:
    """Возвращает список регионов {bbox: [x0,y0,x1,y1], class: 'text'} через Surya."""
    from surya.detection import DetectionPredictor
    from surya.foundation import FoundationPredictor
    from surya.recognition import RecognitionPredictor

    fp = FoundationPredictor()
    rp = RecognitionPredictor(fp)
    dp = DetectionPredictor()

    preds = rp([img], det_predictor=dp)
    regions = []
    if preds and hasattr(preds[0], "text_lines"):
        for line in preds[0].text_lines:
            b = line.bbox  # [x0, y0, x1, y1]
            regions.append({"bbox": b, "text": line.text, "class": "text"})
    return regions


def _bboxes_to_yolo(regions: list[dict], img_w: int, img_h: int) -> str:
    """Конвертирует bboxes в YOLO-формат (class cx cy w h, нормализованные)."""
    lines = []
    for r in regions:
        x0, y0, x1, y1 = r["bbox"]
        cx = (x0 + x1) / 2 / img_w
        cy = (y0 + y1) / 2 / img_h
        bw = (x1 - x0) / img_w
        bh = (y1 - y0) / img_h
        lines.append(f"0 {cx:.6f} {cy:.6f} {bw:.6f} {bh:.6f}")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# Основная логика
# ---------------------------------------------------------------------------


def prepare_recognition(books_dir: str, output_dir: str, pages_per_book: int, dpi: int, seed: int):
    """Экспорт страниц + ground truth текст (для обучения recognition-модели)."""
    from openai import OpenAI

    api_key = os.environ.get("OPENAI_API_KEY", "")
    if not api_key:
        sys.exit("Не задан OPENAI_API_KEY")

    client = OpenAI(api_key=api_key)
    os.makedirs(output_dir, exist_ok=True)
    metadata_path = os.path.join(output_dir, "metadata.jsonl")
    rng = random.Random(seed)

    with open(metadata_path, "a", encoding="utf-8") as meta_f:
        for book_name, pdf_path in _iter_books(books_dir):
            doc = fitz.open(pdf_path)
            n = len(doc)

            # Берём страницы из chapters-файла; если нет — случайные
            candidate_pages = _parse_chapters(books_dir, book_name)
            if not candidate_pages:
                candidate_pages = list(range(1, n + 1))

            pages = rng.sample(candidate_pages, min(pages_per_book, len(candidate_pages)))
            pages.sort()

            print(f"\n{book_name}: {len(pages)} страниц")
            for page_1 in pages:
                stem = f"{book_name}_p{page_1:04d}"
                img_path = os.path.join(output_dir, f"{stem}.png")
                txt_path = os.path.join(output_dir, f"{stem}.txt")

                if os.path.isfile(txt_path):
                    print(f"  [skip] {stem}")
                    continue

                img = _page_to_pil(doc[page_1 - 1], dpi=dpi)
                img.save(img_path)

                try:
                    gt = _get_ground_truth_gpt(img, client)
                except Exception as e:
                    print(f"  [ERR] {stem}: {e}")
                    continue

                with open(txt_path, "w", encoding="utf-8") as f:
                    f.write(gt)

                meta_f.write(json.dumps({"image": img_path, "text": gt}, ensure_ascii=False) + "\n")
                print(f"  [ok] {stem} ({len(gt)} симв.)")

            doc.close()

    print(f"\nGround truth сохранён в {output_dir}")
    print(f"Метаданные: {metadata_path}")


def prepare_detection(books_dir: str, output_dir: str, pages_per_book: int, dpi: int, seed: int):
    """Экспорт страниц + YOLO-аннотации через Surya (для обучения detection-модели)."""
    imgs_dir = os.path.join(output_dir, "images")
    labels_dir = os.path.join(output_dir, "labels")
    os.makedirs(imgs_dir, exist_ok=True)
    os.makedirs(labels_dir, exist_ok=True)

    rng = random.Random(seed)

    for book_name, pdf_path in _iter_books(books_dir):
        doc = fitz.open(pdf_path)
        n = len(doc)

        candidate_pages = _parse_chapters(books_dir, book_name)
        if not candidate_pages:
            candidate_pages = list(range(1, n + 1))

        pages = rng.sample(candidate_pages, min(pages_per_book, len(candidate_pages)))
        pages.sort()

        print(f"\n{book_name}: {len(pages)} страниц")
        for page_1 in pages:
            stem = f"{book_name}_p{page_1:04d}"
            img_out = os.path.join(imgs_dir, f"{stem}.png")
            lbl_out = os.path.join(labels_dir, f"{stem}.txt")

            if os.path.isfile(lbl_out):
                print(f"  [skip] {stem}")
                continue

            img = _page_to_pil(doc[page_1 - 1], dpi=dpi)
            img.save(img_out)

            try:
                regions = _get_surya_bboxes(img)
            except Exception as e:
                print(f"  [ERR] {stem}: {e}")
                continue

            yolo_str = _bboxes_to_yolo(regions, img.width, img.height)
            with open(lbl_out, "w", encoding="utf-8") as f:
                f.write(yolo_str)

            print(f"  [ok] {stem} ({len(regions)} регионов)")

        doc.close()

    # dataset.yaml для YOLOv8
    yaml_path = os.path.join(output_dir, "dataset.yaml")
    with open(yaml_path, "w", encoding="utf-8") as f:
        f.write(f"path: {os.path.abspath(output_dir)}\n")
        f.write("train: images\n")
        f.write("val: images\n")
        f.write("nc: 1\n")
        f.write("names: ['text']\n")

    print(f"\nАннотации сохранены в {output_dir}")
    print(f"Конфиг датасета: {yaml_path}")


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def main():
    parser = argparse.ArgumentParser(description="Подготовка данных для OCR")
    parser.add_argument("--books_dir", required=True, help="Папка с PDF и chapters.txt")
    parser.add_argument("--output_dir", required=True, help="Куда сохранять данные")
    parser.add_argument(
        "--mode",
        choices=("recognition", "detection"),
        default="recognition",
        help="recognition — PNG + ground truth текст; detection — PNG + YOLO bbox",
    )
    parser.add_argument("--pages_per_book", type=int, default=30, help="Страниц на книгу")
    parser.add_argument("--dpi", type=int, default=300, help="DPI рендера")
    parser.add_argument("--seed", type=int, default=42, help="Random seed")
    args = parser.parse_args()

    if args.mode == "recognition":
        prepare_recognition(args.books_dir, args.output_dir, args.pages_per_book, args.dpi, args.seed)
    else:
        prepare_detection(args.books_dir, args.output_dir, args.pages_per_book, args.dpi, args.seed)


if __name__ == "__main__":
    main()
