"""
Text Recognition — обёртка над TrOCR.

Принимает PIL Image вырезанного текстового региона, возвращает строку текста.

Использование:
    from ocr.recognition.model import TextRecognizer

    recognizer = TextRecognizer()                             # базовый trocr-base-printed
    recognizer = TextRecognizer("runs/recognition/checkpoint-1000")  # свои веса

    text = recognizer.predict(cropped_image)
    texts = recognizer.predict_batch([img1, img2, img3])
"""

from __future__ import annotations

from PIL import Image

_DEFAULT_MODEL = "microsoft/trocr-base-printed"
# После fine-tune на русском меняем на путь к своим весам:
# _DEFAULT_MODEL = "runs/recognition/best"


class TextRecognizer:
    """TrOCR-based распознаватель текста на изображении строки/региона."""

    def __init__(self, model_name_or_path: str | None = None, device: str | None = None):
        """
        Args:
            model_name_or_path: HuggingFace model id или путь к локальным весам.
            device: 'cuda', 'mps', 'cpu' или None (авто).
        """
        try:
            from transformers import TrOCRProcessor, VisionEncoderDecoderModel
        except ImportError:
            raise ImportError("Установите transformers: pip install transformers")

        import torch

        if model_name_or_path is None:
            model_name_or_path = _DEFAULT_MODEL

        self.processor = TrOCRProcessor.from_pretrained(model_name_or_path)
        self.model = VisionEncoderDecoderModel.from_pretrained(model_name_or_path)

        if device is None:
            if torch.cuda.is_available():
                device = "cuda"
            elif torch.backends.mps.is_available():
                device = "mps"
            else:
                device = "cpu"

        self.device = device
        self.model.to(device)
        self.model.eval()

    def predict(self, image: Image.Image) -> str:
        """Распознаёт текст на одном изображении."""
        import torch

        if image.mode != "RGB":
            image = image.convert("RGB")

        pixel_values = self.processor(images=image, return_tensors="pt").pixel_values
        pixel_values = pixel_values.to(self.device)

        with torch.no_grad():
            generated_ids = self.model.generate(pixel_values)

        return self.processor.batch_decode(generated_ids, skip_special_tokens=True)[0]

    def predict_batch(self, images: list[Image.Image], batch_size: int = 8) -> list[str]:
        """Распознаёт текст на списке изображений батчами."""
        import torch

        results: list[str] = []
        for i in range(0, len(images), batch_size):
            batch = [img.convert("RGB") for img in images[i : i + batch_size]]
            pixel_values = self.processor(images=batch, return_tensors="pt").pixel_values
            pixel_values = pixel_values.to(self.device)

            with torch.no_grad():
                generated_ids = self.model.generate(pixel_values)

            texts = self.processor.batch_decode(generated_ids, skip_special_tokens=True)
            results.extend(texts)

        return results
