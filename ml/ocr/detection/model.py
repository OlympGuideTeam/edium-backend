"""
Layout Detection — обёртка над YOLOv8.

Принимает PIL Image страницы учебника, возвращает список обнаруженных текстовых блоков.

Использование:
    from ocr.detection.model import LayoutDetector

    detector = LayoutDetector()                        # предобученный PubLayNet
    detector = LayoutDetector("runs/detect/train/weights/best.pt")  # свои веса

    regions = detector.predict(page_image)
    # [{"bbox": [x0, y0, x1, y1], "conf": 0.92, "class": "text"}, ...]
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

import numpy as np
from PIL import Image


# Публичный checkpoint YOLOv8, предобученный на PubLayNet.
# При первом запуске ultralytics скачает файл автоматически.
_PUBLAYNET_WEIGHTS = "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8m.pt"
_DEFAULT_WEIGHTS = "yolov8m.pt"  # fallback: общий YOLOv8m


@dataclass
class DetectionResult:
    bbox: list[float]     # [x0, y0, x1, y1] в пикселях оригинального изображения
    conf: float
    class_id: int
    class_name: str


class LayoutDetector:
    """YOLOv8-based детектор текстовых регионов на странице документа."""

    CLASS_NAMES = {0: "text", 1: "title", 2: "list", 3: "figure", 4: "table"}

    def __init__(self, weights: Optional[str] = None, conf_threshold: float = 0.3):
        """
        Args:
            weights: путь к .pt файлу или None для использования дефолтных весов.
            conf_threshold: минимальный confidence для включения результата.
        """
        try:
            from ultralytics import YOLO
        except ImportError:
            raise ImportError("Установите ultralytics: pip install ultralytics")

        if weights is None:
            weights = _DEFAULT_WEIGHTS

        self.model = YOLO(weights)
        self.conf_threshold = conf_threshold

    def predict(self, image: Image.Image, only_text: bool = True) -> list[DetectionResult]:
        """
        Детектирует регионы на изображении страницы.

        Args:
            image: PIL Image страницы (RGB).
            only_text: если True — возвращать только регионы класса 'text' (0).

        Returns:
            Список DetectionResult, отсортированных сверху вниз.
        """
        img_array = np.array(image)
        results = self.model(img_array, conf=self.conf_threshold, verbose=False)

        detections: list[DetectionResult] = []
        for r in results:
            for box in r.boxes:
                cls_id = int(box.cls[0])
                if only_text and cls_id != 0:
                    continue
                x0, y0, x1, y1 = box.xyxy[0].tolist()
                detections.append(
                    DetectionResult(
                        bbox=[x0, y0, x1, y1],
                        conf=float(box.conf[0]),
                        class_id=cls_id,
                        class_name=self.CLASS_NAMES.get(cls_id, str(cls_id)),
                    )
                )

        # Сортируем сверху вниз (по y0), потом слева направо (по x0)
        detections.sort(key=lambda d: (d.bbox[1], d.bbox[0]))
        return detections

    def crop_regions(
        self, image: Image.Image, detections: list[DetectionResult], padding: int = 4
    ) -> list[Image.Image]:
        """Вырезает регионы из изображения по bbox."""
        crops = []
        w, h = image.size
        for det in detections:
            x0, y0, x1, y1 = det.bbox
            x0 = max(0, int(x0) - padding)
            y0 = max(0, int(y0) - padding)
            x1 = min(w, int(x1) + padding)
            y1 = min(h, int(y1) + padding)
            crops.append(image.crop((x0, y0, x1, y1)))
        return crops
