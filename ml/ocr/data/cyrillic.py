"""
Загрузка публичных кириллических OCR-датасетов с HuggingFace.

Поддерживаемые датасеты:
  - ai-forever/cyrillic-handwriting-dataset   — рукописные кириллические символы
  - ai-forever/ocr                            — печатный OCR (русский)
  - sberbank-ai/ru_alpaca                     — текстовые данные (резерв)

Запуск:
    python -m ocr.data.cyrillic --dataset ai-forever/ocr --output_dir ../../data/recognition/hf
    python -m ocr.data.cyrillic --list
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path


KNOWN_DATASETS: dict[str, dict] = {
    "ai-forever/ocr": {
        "description": "Печатный OCR, русский язык. Пары изображение→текст.",
        "image_col": "image",
        "text_col": "text",
    },
    "ai-forever/cyrillic-handwriting-dataset": {
        "description": "Рукописные кириллические символы и слова.",
        "image_col": "image",
        "text_col": "label",
    },
    "sberbank-ai/ru_wikitext": {
        "description": "Чистый русский текст (Википедия). Без изображений — для языковой модели.",
        "image_col": None,
        "text_col": "text",
    },
}


def list_datasets():
    print("Доступные датасеты:\n")
    for name, info in KNOWN_DATASETS.items():
        print(f"  {name}")
        print(f"    {info['description']}\n")


def download_dataset(dataset_name: str, output_dir: str, split: str, max_samples: int):
    """
    Скачивает датасет с HuggingFace и сохраняет в output_dir:
      - images/{idx}.png  — изображения
      - metadata.jsonl    — {image: path, text: gt}
    """
    try:
        from datasets import load_dataset
    except ImportError:
        raise ImportError("Установите: pip install datasets")

    info = KNOWN_DATASETS.get(dataset_name)
    if info is None:
        raise ValueError(
            f"Неизвестный датасет: {dataset_name}. "
            f"Используйте один из: {list(KNOWN_DATASETS)}"
        )

    if info["image_col"] is None:
        print(f"Датасет {dataset_name} не содержит изображений, пропускаем.")
        return

    print(f"Загрузка {dataset_name} (split={split}, max={max_samples})...")
    ds = load_dataset(dataset_name, split=split, trust_remote_code=True)
    if max_samples and max_samples < len(ds):
        ds = ds.select(range(max_samples))

    imgs_dir = Path(output_dir) / "images"
    imgs_dir.mkdir(parents=True, exist_ok=True)
    meta_path = Path(output_dir) / "metadata.jsonl"

    img_col = info["image_col"]
    txt_col = info["text_col"]

    with open(meta_path, "a", encoding="utf-8") as f:
        for i, sample in enumerate(ds):
            img = sample[img_col]
            text = sample.get(txt_col, "")

            img_path = imgs_dir / f"{i:06d}.png"
            if hasattr(img, "save"):
                img.save(img_path)
            else:
                # bytes или PIL-совместимый объект
                from PIL import Image as PILImage
                import io
                PILImage.open(io.BytesIO(img)).save(img_path)

            f.write(
                json.dumps(
                    {"image": str(img_path), "text": text}, ensure_ascii=False
                )
                + "\n"
            )

            if (i + 1) % 500 == 0:
                print(f"  {i + 1}/{len(ds)}")

    print(f"\nСохранено {len(ds)} примеров в {output_dir}")
    print(f"Метаданные: {meta_path}")


def main():
    parser = argparse.ArgumentParser(description="Загрузка кириллических OCR-датасетов")
    parser.add_argument("--dataset", default="ai-forever/ocr", help="Имя датасета на HuggingFace")
    parser.add_argument("--output_dir", default="../../data/recognition/hf", help="Куда сохранять")
    parser.add_argument("--split", default="train", help="Сплит датасета")
    parser.add_argument("--max_samples", type=int, default=10000, help="Максимум примеров (0 = все)")
    parser.add_argument("--list", action="store_true", help="Показать список датасетов и выйти")
    args = parser.parse_args()

    if args.list:
        list_datasets()
        return

    download_dataset(args.dataset, args.output_dir, args.split, args.max_samples)


if __name__ == "__main__":
    main()
