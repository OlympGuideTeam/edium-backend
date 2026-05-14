import json
import logging
import re
import shutil
import os

import torch
from peft import PeftModel
from transformers import AutoModelForCausalLM, AutoTokenizer, BitsAndBytesConfig

from app.config import settings

logger = logging.getLogger(__name__)

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


def _parse_json(text: str):
    """Парсинг JSON из ответа модели с очисткой."""
    text = text.strip()
    # Убрать ```json ... ```
    if "```" in text:
        match = re.search(r"```(?:json)?\s*(.*?)```", text, re.DOTALL)
        if match:
            text = match.group(1).strip()
    return json.loads(text)


class QuizPipeline:
    def __init__(self):
        self._model = None
        self._tokenizer = None

    @staticmethod
    def _cleanup_stale_caches(cache_base: str, current_slug: str) -> None:
        """Удаляет quantized-кэши моделей, которые не совпадают с текущей."""
        for entry in os.scandir(cache_base):
            if entry.is_dir() and entry.name != current_slug:
                logger.info("Removing stale quantized cache: %s", entry.path)
                shutil.rmtree(entry.path, ignore_errors=True)

    @property
    def is_loaded(self) -> bool:
        return self._model is not None

    def load(self):
        """Загрузка модели. При наличии кэша квантизованной модели — грузит из него (~30 сек),
        иначе квантизует из HuggingFace и сохраняет кэш (~10 мин первый раз)."""
        token = settings.hf_token or None
        model_slug = settings.model_id.replace("/", "--")
        cache_dir = os.path.join(settings.quantized_model_cache_dir, model_slug) if settings.quantized_model_cache_dir else ""

        if settings.quantized_model_cache_dir and os.path.isdir(settings.quantized_model_cache_dir):
            self._cleanup_stale_caches(settings.quantized_model_cache_dir, model_slug)

        self._tokenizer = AutoTokenizer.from_pretrained(settings.model_id, token=token, trust_remote_code=True)
        if self._tokenizer.pad_token is None:
            self._tokenizer.pad_token = self._tokenizer.eos_token
        self._tokenizer.padding_side = "left"

        bnb_config = BitsAndBytesConfig(
            load_in_4bit=True,
            bnb_4bit_quant_type="nf4",
            bnb_4bit_compute_dtype=torch.bfloat16,
            bnb_4bit_use_double_quant=True,
        )

        if cache_dir and os.path.isdir(cache_dir):
            logger.info("Loading quantized model from cache: %s", cache_dir)
            self._model = AutoModelForCausalLM.from_pretrained(
                cache_dir,
                device_map="auto",
                torch_dtype=torch.bfloat16,
                quantization_config=bnb_config,
            )
        else:
            logger.info("Quantizing base model: %s (first run, will save to cache)", settings.model_id)
            self._model = AutoModelForCausalLM.from_pretrained(
                settings.model_id,
                device_map="auto",
                quantization_config=bnb_config,
                torch_dtype=torch.bfloat16,
                token=token,
                trust_remote_code=True,
            )
            if cache_dir:
                logger.info("Saving quantized model to cache: %s", cache_dir)
                self._model.save_pretrained(cache_dir)
                self._tokenizer.save_pretrained(cache_dir)
                logger.info("Quantized model cached")

        token = settings.hf_token or None
        logger.info("Loading extraction adapter: %s", settings.extraction_adapter)
        self._model = PeftModel.from_pretrained(
            self._model,
            settings.extraction_adapter,
            adapter_name="extraction",
            token=token,
        )
        logger.info("Loading generation adapter: %s", settings.generation_adapter)
        self._model.load_adapter(
            settings.generation_adapter,
            adapter_name="generation",
            token=token,
        )
        self._model.eval()

        logger.info("Pipeline ready. VRAM used: %.1f GB", torch.cuda.memory_allocated() / 1024**3)

    def _generate(self, messages: list[dict], max_new_tokens: int | None = None) -> str:
        """Генерация ответа через chat template."""
        prompt = self._tokenizer.apply_chat_template(
            messages, tokenize=False, add_generation_prompt=True
        )
        inputs = self._tokenizer(prompt, return_tensors="pt").to(self._model.device)

        eos_token_id = self._tokenizer.eos_token_id
        pad_token_id = eos_token_id[0] if isinstance(eos_token_id, list) else eos_token_id

        with torch.no_grad():
            outputs = self._model.generate(
                **inputs,
                max_new_tokens=max_new_tokens or settings.max_new_tokens,
                temperature=settings.temperature,
                top_p=settings.top_p,
                do_sample=True,
                eos_token_id=eos_token_id,
                pad_token_id=pad_token_id,
            )

        response = self._tokenizer.decode(
            outputs[0][inputs["input_ids"].shape[1] :], skip_special_tokens=True
        )
        return response.strip()

    def extract_facts(self, text: str) -> list[dict]:
        """Извлечение фактов из текста (extraction LoRA)."""
        self._model.set_adapter("extraction")
        messages = [
            {"role": "system", "content": EXTRACTION_SYSTEM},
            {"role": "user", "content": text},
        ]
        response = self._generate(messages)
        return _parse_json(response)

    def generate_quiz(self, facts: list[dict]) -> dict:
        """Генерация квиза из фактов (generation LoRA)."""
        self._model.set_adapter("generation")
        facts_str = json.dumps(facts, ensure_ascii=False)
        messages = [
            {"role": "system", "content": GENERATION_SYSTEM},
            {"role": "user", "content": f"Факты:\n{facts_str}"},
        ]
        response = self._generate(messages)
        return _parse_json(response)

    def run(self, text: str) -> dict:
        """Полный пайплайн: text → facts → quiz."""
        last_error = None
        for attempt in range(settings.max_retries + 1):
            try:
                facts = self.extract_facts(text)
                if not isinstance(facts, list) or len(facts) == 0:
                    raise ValueError(f"Invalid facts output: expected non-empty list, got {type(facts)}")

                quiz = self.generate_quiz(facts)
                if not isinstance(quiz, dict) or "questions" not in quiz:
                    raise ValueError(f"Invalid quiz output: missing 'questions' key")

                return {"facts": facts, "quiz": quiz}
            except (json.JSONDecodeError, ValueError) as e:
                last_error = e
                logger.warning("Attempt %d failed: %s", attempt + 1, e)

        raise RuntimeError(f"Failed after {settings.max_retries + 1} attempts: {last_error}")


# Singleton
pipeline = QuizPipeline()
