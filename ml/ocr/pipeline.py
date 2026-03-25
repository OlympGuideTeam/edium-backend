"""
Полный OCR-пайплайн: изображение страницы → текст параграфов.

Шаги:
  1. LayoutDetector находит текстовые блоки на странице (YOLOv8).
  2. TextRecognizer распознаёт текст в каждом блоке (TrOCR).
  3. Блоки объединяются в читаемый текст.

Использование:
    from ocr.pipeline import OCRPipeline

    pipeline = OCRPipeline()
    text = pipeline.process_page(page_image)
    pages = pipeline.process_pdf("books/biology_8.pdf", pages=[10, 11, 12])

Запуск из командной строки:
    python -m ocr.pipeline --pdf ../books/biology_8.pdf --page 234
    python -m ocr.pipeline --image page.png
"""

from __future__ import annotations

import io
import sys
from pathlib import Path
from typing import Optional

from PIL import Image


class OCRPipeline:
    def __init__(
        self,
        detector_weights: Optional[str] = None,
        recognizer_weights: Optional[str] = None,
        conf_threshold: float = 0.3,
        device: Optional[str] = None,
    ):
        """
        Args:
            detector_weights: путь к .pt для YOLOv8 (None = дефолтный yolov8m.pt).
            recognizer_weights: путь к TrOCR checkpoint (None = trocr-base-printed).
            conf_threshold: минимальный confidence детектора.
            device: 'cuda', 'mps', 'cpu' или None (авто).
        """
        from ocr.detection.model import LayoutDetector
        from ocr.recognition.model import TextRecognizer

        self.detector = LayoutDetector(detector_weights, conf_threshold=conf_threshold)
        self.recognizer = TextRecognizer(recognizer_weights, device=device)

    def process_page(self, image: Image.Image) -> str:
        """
        Обрабатывает одну страницу: изображение → текст.

        Returns:
            Строка с текстом параграфов, разделённых переносами строк.
        """
        detections = self.detector.predict(image, only_text=True)
        if not detections:
            return ""

        crops = self.detector.crop_regions(image, detections)
        texts = self.recognizer.predict_batch(crops)

        return "\n".join(t for t in texts if t.strip())

    def process_pdf(
        self,
        pdf_path: str,
        pages: Optional[list[int]] = None,
        dpi: int = 300,
        max_side: int = 2048,
    ) -> list[str]:
        """
        Обрабатывает страницы PDF.

        Args:
            pdf_path: путь к PDF.
            pages: список номеров страниц (1-based). None = все страницы.
            dpi: DPI рендера.
            max_side: максимальный размер стороны изображения.

        Returns:
            Список строк — по одной на каждую страницу.
        """
        import fitz

        doc = fitz.open(pdf_path)
        n = len(doc)

        if pages is None:
            pages = list(range(1, n + 1))

        results: list[str] = []
        for page_1 in pages:
            if page_1 < 1 or page_1 > n:
                results.append("")
                continue

            page = doc[page_1 - 1]
            zoom = dpi / 72
            pix = page.get_pixmap(matrix=fitz.Matrix(zoom, zoom))
            img = Image.open(io.BytesIO(pix.tobytes("png"))).convert("RGB")

            w, h = img.size
            if max(w, h) > max_side:
                ratio = max_side / max(w, h)
                img = img.resize((int(w * ratio), int(h * ratio)), Image.Resampling.LANCZOS)

            text = self.process_page(img)
            results.append(text)
            print(f"  Стр {page_1}: {len(text)} символов")

        doc.close()
        return results


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main():
    import argparse

    parser = argparse.ArgumentParser(description="OCR страницы или PDF")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--image", help="Путь к PNG/JPG изображению")
    group.add_argument("--pdf", help="Путь к PDF файлу")
    parser.add_argument("--page", type=int, help="Номер страницы PDF (1-based, по умолчанию = 1)")
    parser.add_argument("--pages", help="Диапазон страниц через запятую, например: 10,11,12")
    parser.add_argument("--detector", default=None, help="Путь к весам YOLOv8 (.pt)")
    parser.add_argument("--recognizer", default=None, help="Путь к TrOCR checkpoint")
    parser.add_argument("--device", default=None)
    args = parser.parse_args()

    pipeline = OCRPipeline(
        detector_weights=args.detector,
        recognizer_weights=args.recognizer,
        device=args.device,
    )

    if args.image:
        img = Image.open(args.image)
        text = pipeline.process_page(img)
        print(text)
    else:
        if args.pages:
            pages = [int(p.strip()) for p in args.pages.split(",")]
        elif args.page:
            pages = [args.page]
        else:
            pages = [1]

        texts = pipeline.process_pdf(args.pdf, pages=pages)
        for i, (p, t) in enumerate(zip(pages, texts)):
            print(f"\n=== Страница {p} ===")
            print(t)


if __name__ == "__main__":
    main()
