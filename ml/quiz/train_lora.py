"""
Обучение двух LoRA-адаптеров для квиз-пайплайна Sphinx.

Пайплайн:
    TEXT → [Fact Extraction LoRA] → FACTS (JSON) → [Quiz Generation LoRA] → QUIZ

Base модель: Qwen/Qwen2.5-14B-Instruct
Метод:       QLoRA (4-bit NF4)
Целевое железо: Tesla T4 (16 GB VRAM)

Запуск:
    python train_lora.py                          # обучить оба адаптера
    python train_lora.py --task extraction        # только extraction
    python train_lora.py --task generation        # только generation
    python train_lora.py --upload                 # обучить + залить на HuggingFace Hub
"""

import argparse
import gc
import json
from pathlib import Path

import torch
from datasets import Dataset
from peft import LoraConfig, get_peft_model
from transformers import (
    AutoModelForCausalLM,
    AutoTokenizer,
    BitsAndBytesConfig,
    TrainingArguments,
)
from trl import SFTTrainer

# ── Пути ─────────────────────────────────────────────────────────────────────

BASE_DIR = Path(__file__).resolve().parent
DATA_DIR = BASE_DIR / "data" / "datasets"
MODELS_DIR = BASE_DIR / "models"
MODELS_DIR.mkdir(parents=True, exist_ok=True)

# ── Модель ────────────────────────────────────────────────────────────────────

MODEL_ID = "Qwen/Qwen2.5-14B-Instruct"

# ── Гиперпараметры (Tesla T4, 16 GB VRAM) ─────────────────────────────────────
# Qwen2.5-14B в 4-bit NF4 ≈ 7.5 GB весов.
# batch_size=1 + grad_checkpointing + LoRA ≈ 13–14 GB при seq_len=2048.

MAX_SEQ_LENGTH = 2048
LORA_R = 16
LORA_ALPHA = 32
LORA_DROPOUT = 0.05
LEARNING_RATE = 2e-4
NUM_EPOCHS = 3
BATCH_SIZE = 1
GRAD_ACCUM = 16  # effective batch = 16
WARMUP_RATIO = 0.05

# ── Промпты (идентичны sphinx/app/pipeline.py) ──────────────────────────────

EXTRACTION_SYSTEM = """\
Извлеки 5-15 ключевых фактов из текста.

Формат — строгий JSON-массив:
[
  {"question": "...", "answer": "...", "type": "person | date | term | number | location | event | definition | process"}
]

ЖЕСТКИЕ ПРАВИЛА:
1. Только факты из текста.
2. Ответ (answer) должен быть УЛЬТРАКОРОТКИМ: максимум 1-3 слова. Это должно быть конкретное имя, год, место, число или термин.
3. НИКАКИХ предложений, описаний и причастных оборотов в answer.
   - ПЛОХО: "На территории, где находились хижины строителей"
   - ХОРОШО: "Возле Луксора"
   - ПЛОХО: "Золотая посмертная маска властелина Египта"
   - ХОРОШО: "Золотая маска"
4. Формулируй вопрос (question) так, чтобы на него можно было ответить одним-двумя словами.
5. Используй русский язык.
6. Выведи ТОЛЬКО валидный JSON, без вступительных слов и без разметки ```json."""

GENERATION_SYSTEM = """\
Создай образовательный квиз по фактам.

Формат — строгий JSON:
{
  "questions": [
    {"type": "single_choice", "question": "строка", "answer": "строка", "options": ["строка", "строка", "строка", "строка"]},
    {"type": "multiple_choice", "question": "строка", "answer": ["строка", "строка"], "options": ["строка", "строка", "строка", "строка", "строка"]},
    {"type": "short_answer", "question": "строка", "answer": "строка"}
  ]
}

ЖЕСТКИЕ ПРАВИЛА (ШТРАФ ЗА НАРУШЕНИЕ):
1. Сгенерируй РОВНО 6-8 вопросов. Не меньше 6, не больше 8.
2. ОБЯЗАТЕЛЬНОЕ РАСПРЕДЕЛЕНИЕ ТИПОВ:
   - single_choice: МИНИМУМ 3 вопроса (самый частый тип)
   - multiple_choice: МИНИМУМ 1 вопрос
   - short_answer: НЕ БОЛЕЕ 2 вопросов (редкий тип — только для имён, дат, терминов)
   Пример допустимого распределения: 4 single_choice + 2 multiple_choice + 2 short_answer.
   ЗАПРЕЩЕНО: более 2 short_answer подряд или более 2 short_answer всего.
3. Для short_answer: "answer" — 1-3 слова (имя, дата, термин). ЗАПРЕЩЕНЫ предложения. Если факт сложнее — делай single_choice.
4. Для single_choice: "answer" должен БУКВА В БУКВУ совпадать с одним из элементов "options".
5. Для multiple_choice:
   - "answer" — ровно 2 или 3 элемента.
   - Каждый элемент "answer" должен БУКВА В БУКВУ совпадать с элементом из "options".
6. Варианты ответов "options" — правдоподобные, без явных подсказок.
7. Используй русский язык.
8. Верни ТОЛЬКО валидный JSON, начинающийся с символа {, без маркдауна и комментариев."""


# ── Data ──────────────────────────────────────────────────────────────────────


def load_jsonl(path: Path) -> list[dict]:
    data = []
    with open(path, encoding="utf-8") as f:
        for line in f:
            if line.strip():
                data.append(json.loads(line))
    return data


def print_stats(name: str, train: list, val: list):
    print(f"{name} — train: {len(train)}, val: {len(val)}")


# ── Model ─────────────────────────────────────────────────────────────────────


def make_bnb_config() -> BitsAndBytesConfig:
    return BitsAndBytesConfig(
        load_in_4bit=True,
        bnb_4bit_quant_type="nf4",
        bnb_4bit_compute_dtype=torch.bfloat16,
        bnb_4bit_use_double_quant=True,
    )


def load_tokenizer() -> AutoTokenizer:
    tok = AutoTokenizer.from_pretrained(MODEL_ID, trust_remote_code=True)
    if tok.pad_token is None:
        tok.pad_token = tok.eos_token
    tok.padding_side = "right"
    return tok


def load_base_model() -> AutoModelForCausalLM:
    model = AutoModelForCausalLM.from_pretrained(
        MODEL_ID,
        quantization_config=make_bnb_config(),
        device_map="auto",
        torch_dtype=torch.bfloat16,
        trust_remote_code=True,
    )
    model.config.use_cache = False
    return model


def make_lora_config() -> LoraConfig:
    return LoraConfig(
        r=LORA_R,
        lora_alpha=LORA_ALPHA,
        lora_dropout=LORA_DROPOUT,
        target_modules=["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"],
        bias="none",
        task_type="CAUSAL_LM",
    )


# ── Training ──────────────────────────────────────────────────────────────────


def make_training_args(output_dir: str, num_train_samples: int) -> TrainingArguments:
    steps_per_epoch = max(1, num_train_samples // (BATCH_SIZE * GRAD_ACCUM))
    eval_steps = max(1, steps_per_epoch // 4)

    return TrainingArguments(
        output_dir=output_dir,
        num_train_epochs=NUM_EPOCHS,
        per_device_train_batch_size=BATCH_SIZE,
        per_device_eval_batch_size=BATCH_SIZE,
        gradient_accumulation_steps=GRAD_ACCUM,
        learning_rate=LEARNING_RATE,
        lr_scheduler_type="cosine",
        warmup_ratio=WARMUP_RATIO,
        bf16=True,
        gradient_checkpointing=True,
        gradient_checkpointing_kwargs={"use_reentrant": False},
        logging_steps=10,
        eval_strategy="steps",
        eval_steps=eval_steps,
        save_strategy="steps",
        save_steps=eval_steps,
        save_total_limit=2,
        load_best_model_at_end=True,
        metric_for_best_model="eval_loss",
        greater_is_better=False,
        report_to="none",
        optim="paged_adamw_8bit",
        max_grad_norm=0.3,
        seed=42,
        dataloader_num_workers=0,
    )


def train_lora(
    tokenizer: AutoTokenizer,
    train_data: list[dict],
    val_data: list[dict],
    output_dir: str,
    task_name: str,
) -> str:
    print(f"\n{'=' * 60}")
    print(f"Training: {task_name}")
    print(f"Train: {len(train_data)}, Val: {len(val_data)}")
    print(f"{'=' * 60}\n")

    model = load_base_model()
    model = get_peft_model(model, make_lora_config())
    model.print_trainable_parameters()

    def formatting_func(example):
        return tokenizer.apply_chat_template(example["messages"], tokenize=False)

    trainer = SFTTrainer(
        model=model,
        train_dataset=Dataset.from_list(train_data),
        eval_dataset=Dataset.from_list(val_data),
        args=make_training_args(output_dir + "_checkpoints", len(train_data)),
        formatting_func=formatting_func,
        max_seq_length=MAX_SEQ_LENGTH,
        packing=False,
        tokenizer=tokenizer,
    )

    trainer.train()
    model.save_pretrained(output_dir)
    tokenizer.save_pretrained(output_dir)
    print(f"\nAdapter saved: {output_dir}")

    del model, trainer
    gc.collect()
    torch.cuda.empty_cache()

    return output_dir


# ── Upload ────────────────────────────────────────────────────────────────────


def upload_to_hub(adapter_dir: str, repo_id: str):
    from huggingface_hub import HfApi, login

    login()
    api = HfApi()
    api.create_repo(repo_id, private=True, exist_ok=True)
    print(f"Uploading {adapter_dir} → {repo_id} ...")
    api.upload_folder(
        folder_path=adapter_dir,
        repo_id=repo_id,
        commit_message=f"Upload LoRA adapter from {Path(adapter_dir).name}",
    )
    print(f"Uploaded: {repo_id}")


# ── Main ──────────────────────────────────────────────────────────────────────


def main():
    parser = argparse.ArgumentParser(description="Train LoRA adapters for Sphinx quiz pipeline")
    parser.add_argument("--task", choices=["extraction", "generation", "both"], default="both")
    parser.add_argument("--upload", action="store_true", help="Push adapters to HuggingFace Hub after training")
    parser.add_argument("--extraction-repo", default="edium/sphinx-extraction-lora")
    parser.add_argument("--generation-repo", default="edium/sphinx-generation-lora")
    args = parser.parse_args()

    print(f"PyTorch: {torch.__version__}")
    print(f"CUDA: {torch.cuda.is_available()}")
    if torch.cuda.is_available():
        print(f"GPU: {torch.cuda.get_device_name(0)}")
        print(f"VRAM: {torch.cuda.get_device_properties(0).total_memory / 1024**3:.1f} GB")

    tokenizer = load_tokenizer()
    print(f"Tokenizer loaded. Vocab: {tokenizer.vocab_size}")

    extraction_dir = str(MODELS_DIR / "extraction_lora")
    generation_dir = str(MODELS_DIR / "generation_lora")

    if args.task in ("extraction", "both"):
        ext_train = load_jsonl(DATA_DIR / "extraction_train.jsonl")
        ext_val = load_jsonl(DATA_DIR / "extraction_val.jsonl")
        print_stats("Extraction", ext_train, ext_val)
        train_lora(tokenizer, ext_train, ext_val, extraction_dir, "Fact Extraction")
        if args.upload:
            upload_to_hub(extraction_dir, args.extraction_repo)

    if args.task in ("generation", "both"):
        gen_train = load_jsonl(DATA_DIR / "generation_train.jsonl")
        gen_val = load_jsonl(DATA_DIR / "generation_val.jsonl")
        print_stats("Generation", gen_train, gen_val)
        train_lora(tokenizer, gen_train, gen_val, generation_dir, "Quiz Generation")
        if args.upload:
            upload_to_hub(generation_dir, args.generation_repo)

    print("\nDone!")
    if torch.cuda.is_available():
        print(f"VRAM used: {torch.cuda.memory_allocated() / 1024**3:.1f} GB")

    print("\nAdapter paths:")
    print(f"  Extraction: {extraction_dir}/")
    print(f"  Generation: {generation_dir}/")
    print(f"\nBase model ({MODEL_ID}) loads from HuggingFace automatically.")
    print("T4 (16GB) memory estimate:")
    print("  4-bit model weights: ~7.5 GB")
    print("  LoRA + optimizer + activations: ~5-6 GB")
    print("  Total: ~13-14 GB  ✓ fits T4")


if __name__ == "__main__":
    main()
