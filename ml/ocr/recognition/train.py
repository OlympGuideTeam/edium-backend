"""
Fine-tune TrOCR для распознавания текста на русских учебниках.

Ожидает датасет в формате metadata.jsonl (из ocr/data/prepare.py или ocr/data/cyrillic.py):
    data/recognition/
    ├── metadata.jsonl   # {"image": "path.png", "text": "ground truth"}
    └── *.png

Запуск на RTX 3080ti:
    python -m ocr.recognition.train \
        --data_dir ../../data/recognition \
        --output_dir runs/recognition \
        --model microsoft/trocr-base-printed \
        --epochs 10 \
        --batch 8
"""

from __future__ import annotations

import argparse
import json
import os
import random
from pathlib import Path


# ---------------------------------------------------------------------------
# Dataset
# ---------------------------------------------------------------------------

class OCRDataset:
    def __init__(self, metadata_path: str, processor, max_target_length: int = 128):
        self.processor = processor
        self.max_target_length = max_target_length
        self.samples: list[dict] = []

        with open(metadata_path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if line:
                    self.samples.append(json.loads(line))

    def __len__(self):
        return len(self.samples)

    def __getitem__(self, idx):
        from PIL import Image

        sample = self.samples[idx]
        img = Image.open(sample["image"]).convert("RGB")
        text = sample["text"]

        pixel_values = self.processor(images=img, return_tensors="pt").pixel_values.squeeze(0)

        labels = self.processor.tokenizer(
            text,
            padding="max_length",
            max_length=self.max_target_length,
            truncation=True,
            return_tensors="pt",
        ).input_ids.squeeze(0)

        # Заменяем padding token id на -100 (игнорируется при вычислении loss)
        labels[labels == self.processor.tokenizer.pad_token_id] = -100

        return {"pixel_values": pixel_values, "labels": labels}


def split_metadata(metadata_path: str, val_ratio: float = 0.1, seed: int = 42) -> tuple[str, str]:
    """Разбивает metadata.jsonl на train/val, сохраняет рядом."""
    base = Path(metadata_path).parent
    with open(metadata_path, encoding="utf-8") as f:
        lines = [line for line in f.readlines() if line.strip()]

    random.seed(seed)
    random.shuffle(lines)
    n_val = max(1, int(len(lines) * val_ratio))

    train_path = base / "metadata_train.jsonl"
    val_path = base / "metadata_val.jsonl"

    with open(train_path, "w", encoding="utf-8") as f:
        f.writelines(lines[n_val:])
    with open(val_path, "w", encoding="utf-8") as f:
        f.writelines(lines[:n_val])

    print(f"Train: {len(lines) - n_val} / Val: {n_val}")
    return str(train_path), str(val_path)


# ---------------------------------------------------------------------------
# CER/WER метрика
# ---------------------------------------------------------------------------

def compute_metrics_fn(processor):
    import evaluate

    cer_metric = evaluate.load("cer")
    wer_metric = evaluate.load("wer")

    def compute_metrics(pred):
        labels_ids = pred.label_ids
        pred_ids = pred.predictions

        # Декодируем предсказания
        pred_str = processor.batch_decode(pred_ids, skip_special_tokens=True)

        # Заменяем -100 на pad token id для декодирования labels
        labels_ids[labels_ids == -100] = processor.tokenizer.pad_token_id
        label_str = processor.batch_decode(labels_ids, skip_special_tokens=True)

        cer = cer_metric.compute(predictions=pred_str, references=label_str)
        wer = wer_metric.compute(predictions=pred_str, references=label_str)
        return {"cer": cer, "wer": wer}

    return compute_metrics


# ---------------------------------------------------------------------------
# Обучение
# ---------------------------------------------------------------------------

def train(
    data_dir: str,
    output_dir: str,
    model_name: str,
    epochs: int,
    batch_size: int,
    learning_rate: float,
    max_target_length: int,
    val_ratio: float,
    fp16: bool,
    seed: int,
):
    from transformers import (
        TrOCRProcessor,
        VisionEncoderDecoderModel,
        Seq2SeqTrainer,
        Seq2SeqTrainingArguments,
        default_data_collator,
    )

    print(f"Загрузка модели: {model_name}")
    processor = TrOCRProcessor.from_pretrained(model_name)
    model = VisionEncoderDecoderModel.from_pretrained(model_name)

    # Настройка токенизатора для decoder
    model.config.decoder_start_token_id = processor.tokenizer.cls_token_id
    model.config.pad_token_id = processor.tokenizer.pad_token_id
    model.config.vocab_size = model.config.decoder.vocab_size

    # Параметры генерации
    model.config.eos_token_id = processor.tokenizer.sep_token_id
    model.config.max_length = max_target_length
    model.config.early_stopping = True
    model.config.no_repeat_ngram_size = 3
    model.config.length_penalty = 2.0
    model.config.num_beams = 4

    metadata_path = os.path.join(data_dir, "metadata.jsonl")
    if not os.path.isfile(metadata_path):
        raise FileNotFoundError(f"Не найден файл: {metadata_path}")

    train_meta, val_meta = split_metadata(metadata_path, val_ratio=val_ratio, seed=seed)

    train_ds = OCRDataset(train_meta, processor, max_target_length)
    val_ds = OCRDataset(val_meta, processor, max_target_length)
    print(f"Train: {len(train_ds)} / Val: {len(val_ds)}")

    training_args = Seq2SeqTrainingArguments(
        output_dir=output_dir,
        num_train_epochs=epochs,
        per_device_train_batch_size=batch_size,
        per_device_eval_batch_size=batch_size,
        learning_rate=learning_rate,
        warmup_steps=200,
        weight_decay=0.01,
        logging_steps=50,
        evaluation_strategy="epoch",
        save_strategy="epoch",
        load_best_model_at_end=True,
        metric_for_best_model="cer",
        greater_is_better=False,
        predict_with_generate=True,
        fp16=fp16,
        seed=seed,
        report_to="none",
        save_total_limit=2,
    )

    trainer = Seq2SeqTrainer(
        model=model,
        args=training_args,
        train_dataset=train_ds,
        eval_dataset=val_ds,
        data_collator=default_data_collator,
        compute_metrics=compute_metrics_fn(processor),
    )

    print("Обучение...")
    trainer.train()

    # Сохраняем финальную модель
    best_dir = os.path.join(output_dir, "best")
    trainer.save_model(best_dir)
    processor.save_pretrained(best_dir)
    print(f"\nЛучшая модель сохранена в: {best_dir}")
    return best_dir


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Fine-tune TrOCR на русских учебниках")
    parser.add_argument("--data_dir", required=True, help="Папка с metadata.jsonl и изображениями")
    parser.add_argument("--output_dir", default="runs/recognition", help="Куда сохранять модель")
    parser.add_argument("--model", default="microsoft/trocr-base-printed", help="Базовая модель")
    parser.add_argument("--epochs", type=int, default=10)
    parser.add_argument("--batch", type=int, default=8)
    parser.add_argument("--lr", type=float, default=5e-5)
    parser.add_argument("--max_target_length", type=int, default=128)
    parser.add_argument("--val_ratio", type=float, default=0.1)
    parser.add_argument("--fp16", action="store_true", help="Mixed precision (RTX 30xx+)")
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    train(
        data_dir=args.data_dir,
        output_dir=args.output_dir,
        model_name=args.model,
        epochs=args.epochs,
        batch_size=args.batch,
        learning_rate=args.lr,
        max_target_length=args.max_target_length,
        val_ratio=args.val_ratio,
        fp16=args.fp16,
        seed=args.seed,
    )


if __name__ == "__main__":
    main()
