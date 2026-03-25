"""
Бенчмарк OCR-моделей на страницах учебников.

Сравниваем: Tesseract, Surya, EasyOCR, PaddleOCR, наш пайплайн (YOLOv8 + TrOCR).
Метрики: CER, WER, время инференса.

Для метрик нужен ground truth (файл .txt рядом с PDF или флаг --ground_truth).

Запуск (только скорость, без метрик):
    python -m ocr.benchmark --pdf ../books/biology_8.pdf --page 234

Запуск с метриками (нужен ground truth):
    python -m ocr.benchmark --pdf ../books/biology_8.pdf --page 234 \
        --ground_truth "Текст страницы для сравнения..."
"""

from __future__ import annotations

import argparse
import io
import time
from dataclasses import dataclass, field
from typing import Optional

import fitz
from PIL import Image


@dataclass
class BenchmarkResult:
    model: str
    text: str
    elapsed: float
    cer: Optional[float] = None
    wer: Optional[float] = None


def _pdf_to_pil(pdf_path: str, page_num: int, dpi: int = 300) -> tuple[Image.Image, fitz.Page, fitz.Document]:
    doc = fitz.open(pdf_path)
    page = doc[page_num - 1]
    zoom = dpi / 72
    pix = page.get_pixmap(matrix=fitz.Matrix(zoom, zoom))
    img = Image.open(io.BytesIO(pix.tobytes("png"))).convert("RGB")
    return img, page, doc


def _compute_metrics(pred: str, reference: str) -> tuple[float, float]:
    try:
        import evaluate
        cer_m = evaluate.load("cer")
        wer_m = evaluate.load("wer")
        cer = cer_m.compute(predictions=[pred], references=[reference])
        wer = wer_m.compute(predictions=[pred], references=[reference])
        return cer, wer
    except Exception:
        return None, None


# ---------------------------------------------------------------------------
# Модели
# ---------------------------------------------------------------------------

def run_tesseract(page: fitz.Page) -> tuple[str, float]:
    t0 = time.time()
    tp = page.get_textpage_ocr(language="rus+eng", flags=fitz.TEXT_PRESERVE_WHITESPACE)
    text = page.get_text("text", textpage=tp).strip()
    return text, time.time() - t0


def run_surya(image: Image.Image) -> tuple[str, float]:
    from surya.foundation import FoundationPredictor
    from surya.recognition import RecognitionPredictor
    from surya.detection import DetectionPredictor

    fp = FoundationPredictor()
    rp = RecognitionPredictor(fp)
    dp = DetectionPredictor()

    t0 = time.time()
    preds = rp([image], det_predictor=dp)
    elapsed = time.time() - t0

    text = ""
    if preds and hasattr(preds[0], "text_lines"):
        text = "\n".join(line.text for line in preds[0].text_lines)
    return text, elapsed


def run_easyocr(image: Image.Image) -> tuple[str, float]:
    import easyocr
    import numpy as np

    reader = easyocr.Reader(["ru", "en"], gpu=True)
    t0 = time.time()
    result = reader.readtext(np.array(image), detail=0, paragraph=True)
    return "\n".join(result), time.time() - t0


def run_paddleocr(image: Image.Image) -> tuple[str, float]:
    from paddleocr import PaddleOCR
    import numpy as np

    ocr = PaddleOCR(use_angle_cls=True, lang="ru")
    t0 = time.time()
    result = ocr.ocr(np.array(image), cls=True)
    elapsed = time.time() - t0

    lines = []
    if result and result[0]:
        for line in result[0]:
            if line and len(line) >= 2:
                lines.append(line[1][0])
    return "\n".join(lines), elapsed


def run_our_pipeline(image: Image.Image, detector_weights=None, recognizer_weights=None, device=None) -> tuple[str, float]:
    from ocr.pipeline import OCRPipeline

    pipeline = OCRPipeline(detector_weights, recognizer_weights, device=device)
    t0 = time.time()
    text = pipeline.process_page(image)
    return text, time.time() - t0


# ---------------------------------------------------------------------------
# Бенчмарк
# ---------------------------------------------------------------------------

def run_benchmark(
    pdf_path: str,
    page_num: int,
    models: list[str],
    ground_truth: Optional[str],
    dpi: int,
    detector_weights: Optional[str],
    recognizer_weights: Optional[str],
    device: Optional[str],
) -> list[BenchmarkResult]:
    image, page, doc = _pdf_to_pil(pdf_path, page_num, dpi)
    print(f"PDF: {pdf_path}, страница: {page_num}, размер: {image.size}")

    results: list[BenchmarkResult] = []

    for model_name in models:
        print(f"\n>>> {model_name}...")
        try:
            if model_name == "tesseract":
                text, elapsed = run_tesseract(page)
            elif model_name == "surya":
                text, elapsed = run_surya(image)
            elif model_name == "easyocr":
                text, elapsed = run_easyocr(image)
            elif model_name == "paddleocr":
                text, elapsed = run_paddleocr(image)
            elif model_name == "ours":
                text, elapsed = run_our_pipeline(image, detector_weights, recognizer_weights, device)
            else:
                print(f"  Неизвестная модель: {model_name}")
                continue

            cer, wer = (None, None)
            if ground_truth:
                cer, wer = _compute_metrics(text, ground_truth)

            results.append(BenchmarkResult(model_name, text, elapsed, cer, wer))
            print(f"  Время: {elapsed:.2f}с | Символов: {len(text)} | Строк: {len(text.splitlines())}")
            if cer is not None:
                print(f"  CER: {cer:.4f} | WER: {wer:.4f}")

        except ImportError as e:
            print(f"  [SKIP] {model_name} не установлен: {e}")
        except Exception as e:
            print(f"  [ERR] {model_name}: {e}")

    doc.close()
    return results


def print_table(results: list[BenchmarkResult]):
    has_metrics = any(r.cer is not None for r in results)

    print("\n" + "=" * 80)
    print("  ИТОГИ")
    print("=" * 80)

    if has_metrics:
        print(f"{'Модель':<15} {'Время (с)':<12} {'Символов':<12} {'Строк':<8} {'CER':<10} {'WER'}")
        print("-" * 65)
        for r in results:
            cer_s = f"{r.cer:.4f}" if r.cer is not None else "—"
            wer_s = f"{r.wer:.4f}" if r.wer is not None else "—"
            print(f"{r.model:<15} {r.elapsed:<12.2f} {len(r.text):<12} {len(r.text.splitlines()):<8} {cer_s:<10} {wer_s}")
    else:
        print(f"{'Модель':<15} {'Время (с)':<12} {'Символов':<12} {'Строк'}")
        print("-" * 50)
        for r in results:
            print(f"{r.model:<15} {r.elapsed:<12.2f} {len(r.text):<12} {len(r.text.splitlines())}")


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Бенчмарк OCR-моделей")
    parser.add_argument("--pdf", required=True, help="Путь к PDF")
    parser.add_argument("--page", type=int, default=1, help="Номер страницы (1-based)")
    parser.add_argument(
        "--models",
        nargs="+",
        default=["tesseract", "surya", "easyocr", "paddleocr", "ours"],
        help="Модели для тестирования",
    )
    parser.add_argument("--ground_truth", default="", help="Эталонный текст для расчёта CER/WER")
    parser.add_argument("--ground_truth_file", default="", help="Файл с эталонным текстом")
    parser.add_argument("--dpi", type=int, default=300)
    parser.add_argument("--detector", default=None, help="Веса YOLOv8 для нашей модели")
    parser.add_argument("--recognizer", default=None, help="Веса TrOCR для нашей модели")
    parser.add_argument("--device", default=None)
    parser.add_argument("--preview", action="store_true", help="Показать первые 300 символов каждой модели")
    args = parser.parse_args()

    gt = args.ground_truth
    if args.ground_truth_file:
        with open(args.ground_truth_file, encoding="utf-8") as f:
            gt = f.read().strip()

    results = run_benchmark(
        pdf_path=args.pdf,
        page_num=args.page,
        models=args.models,
        ground_truth=gt or None,
        dpi=args.dpi,
        detector_weights=args.detector,
        recognizer_weights=args.recognizer,
        device=args.device,
    )

    print_table(results)

    if args.preview:
        for r in results:
            print(f"\n--- {r.model} ---")
            print(r.text[:300])
            if len(r.text) > 300:
                print("... [обрезано]")


if __name__ == "__main__":
    main()
