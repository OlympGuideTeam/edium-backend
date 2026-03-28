"""
Fine-tune YOLOv8 для детекции текстовых регионов на страницах учебников.

Предполагает наличие датасета в YOLO-формате (подготовленного через ocr/data/prepare.py):
    data/detection/
    ├── images/          # PNG-страницы
    ├── labels/          # YOLO-аннотации (.txt)
    └── dataset.yaml     # конфиг датасета

Запуск на RTX 3080ti:
    python -m ocr.detection.train --data ../../data/detection/dataset.yaml --epochs 50 --batch 16
"""

from __future__ import annotations

import argparse
from pathlib import Path


def split_dataset(data_dir: str, val_ratio: float = 0.15, seed: int = 42):
    """
    Разбивает dataset.yaml на train/val сплиты на месте.
    Создаёт images/train, images/val, labels/train, labels/val.
    """
    import random
    import shutil

    data_path = Path(data_dir)
    imgs = sorted((data_path / "images").glob("*.png"))
    random.seed(seed)
    random.shuffle(imgs)
    n_val = max(1, int(len(imgs) * val_ratio))
    val_imgs = set(p.stem for p in imgs[:n_val])

    for split in ("train", "val"):
        (data_path / "images" / split).mkdir(parents=True, exist_ok=True)
        (data_path / "labels" / split).mkdir(parents=True, exist_ok=True)

    for img_path in imgs:
        split = "val" if img_path.stem in val_imgs else "train"
        dst_img = data_path / "images" / split / img_path.name
        dst_lbl = data_path / "labels" / split / (img_path.stem + ".txt")
        src_lbl = data_path / "labels" / (img_path.stem + ".txt")

        if not dst_img.exists():
            shutil.copy2(img_path, dst_img)
        if src_lbl.exists() and not dst_lbl.exists():
            shutil.copy2(src_lbl, dst_lbl)

    # Обновляем dataset.yaml
    yaml_path = data_path / "dataset.yaml"
    with open(yaml_path, "w", encoding="utf-8") as f:
        f.write(f"path: {data_path.resolve()}\n")
        f.write("train: images/train\n")
        f.write("val: images/val\n")
        f.write("nc: 1\n")
        f.write("names: ['text']\n")

    print(f"Сплит: {len(imgs) - n_val} train / {n_val} val")
    return str(yaml_path)


def train(
    data_yaml: str,
    weights: str,
    epochs: int,
    batch: int,
    imgsz: int,
    device: str,
    project: str,
    name: str,
):
    try:
        from ultralytics import YOLO
    except ImportError:
        raise ImportError("Установите ultralytics: pip install ultralytics")

    print(f"Загрузка модели: {weights}")
    model = YOLO(weights)

    print(f"Обучение: epochs={epochs}, batch={batch}, imgsz={imgsz}, device={device}")
    results = model.train(
        data=data_yaml,
        epochs=epochs,
        batch=batch,
        imgsz=imgsz,
        device=device,
        project=project,
        name=name,
        patience=15,  # early stopping
        save=True,
        plots=True,  # сохраняет графики loss/metrics
        val=True,
    )
    print(f"\nОбучение завершено. Результаты: {results}")
    best = Path(project) / name / "weights" / "best.pt"
    print(f"Лучшие веса: {best}")
    return str(best)


def evaluate(weights: str, data_yaml: str, imgsz: int, device: str):
    from ultralytics import YOLO

    model = YOLO(weights)
    metrics = model.val(data=data_yaml, imgsz=imgsz, device=device)
    print(f"\nmAP50: {metrics.box.map50:.4f}")
    print(f"mAP50-95: {metrics.box.map:.4f}")
    return metrics


def main():
    parser = argparse.ArgumentParser(description="Fine-tune YOLOv8 для детекции текстовых регионов")
    parser.add_argument("--data", required=True, help="Путь к dataset.yaml")
    parser.add_argument("--weights", default="yolov8m.pt", help="Стартовые веса (YOLOv8)")
    parser.add_argument("--epochs", type=int, default=50)
    parser.add_argument("--batch", type=int, default=16)
    parser.add_argument("--imgsz", type=int, default=1024, help="Размер изображения при обучении")
    parser.add_argument("--device", default="0", help="cuda device (0, 1, ...) или cpu")
    parser.add_argument("--project", default="runs/detection", help="Папка для результатов")
    parser.add_argument("--name", default="textbook_v1", help="Имя эксперимента")
    parser.add_argument("--split", action="store_true", help="Разбить датасет на train/val перед обучением")
    parser.add_argument("--eval_only", default="", help="Только оценка: путь к .pt файлу")
    args = parser.parse_args()

    if args.eval_only:
        evaluate(args.eval_only, args.data, args.imgsz, args.device)
        return

    data_yaml = args.data
    if args.split:
        data_dir = str(Path(args.data).parent)
        data_yaml = split_dataset(data_dir)

    best_weights = train(
        data_yaml=data_yaml,
        weights=args.weights,
        epochs=args.epochs,
        batch=args.batch,
        imgsz=args.imgsz,
        device=args.device,
        project=args.project,
        name=args.name,
    )

    print("\nОценка на val:")
    evaluate(best_weights, data_yaml, args.imgsz, args.device)


if __name__ == "__main__":
    main()
