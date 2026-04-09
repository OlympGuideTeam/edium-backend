import json
import logging
import re

import os

import torch
# from peft import PeftModel  # TODO: раскомментировать после обучения LoRA-адаптеров
from transformers import AutoModelForCausalLM, AutoTokenizer, BitsAndBytesConfig

from app.config import settings

logger = logging.getLogger(__name__)

EXTRACTION_SYSTEM = """\
Извлеки 5-15 ключевых фактов из текста.

Формат — JSON-массив:
[
  {"question": "...", "answer": "...", "type": "person | date | term | number | location | event | definition | process"}
]

Правила:
- Только факты из текста
- Короткие ответы (1-5 слов)
- Отвечай на том же языке, что и входной текст
- Без объяснений, верни ТОЛЬКО JSON"""

GENERATION_SYSTEM = """\
Создай образовательный квиз по фактам.

ФОРМАТ — строгий JSON:
{
  "questions": [
    {"type": "single_choice", "question": "строка", "answer": "строка", "options": ["строка", "строка", "строка", "строка"]},
    {"type": "multiple_choice", "question": "строка", "answer": ["строка", "строка"], "options": ["строка", "строка", "строка", "строка", "строка"]},
    {"type": "short_answer", "question": "строка", "answer": "строка"}
  ]
}

Правила:
- 5-10 вопросов, используй разные типы
- single_choice: answer — строка, options — 4 варианта (включая правильный)
- multiple_choice: answer — массив из 2-3 правильных ответов, options — 4-5 вариантов
- short_answer: answer — краткий ответ (1-5 слов), без options
- answer должен быть точным фактом из предоставленных данных
- Отвечай на том же языке, что и входной текст
- Верни ТОЛЬКО JSON"""


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
        # TODO: раскомментировать после обучения адаптеров
        # self._extraction_adapter = None
        # self._generation_adapter = None

    @property
    def is_loaded(self) -> bool:
        return self._model is not None

    def load(self):
        """Загрузка модели. При наличии кэша квантизованной модели — грузит из него (~30 сек),
        иначе квантизует из HuggingFace и сохраняет кэш (~10 мин первый раз)."""
        token = settings.hf_token or None
        cache_dir = settings.quantized_model_cache_dir

        self._tokenizer = AutoTokenizer.from_pretrained(settings.model_id, token=token)
        self._tokenizer.pad_token = self._tokenizer.eos_token
        self._tokenizer.padding_side = "right"

        if cache_dir and os.path.isdir(cache_dir):
            logger.info("Loading quantized model from cache: %s", cache_dir)
            self._model = AutoModelForCausalLM.from_pretrained(
                cache_dir,
                device_map="auto",
                torch_dtype=torch.bfloat16,
            )
        else:
            logger.info("Quantizing base model: %s (first run, will save to cache)", settings.model_id)
            bnb_config = BitsAndBytesConfig(
                load_in_4bit=True,
                bnb_4bit_quant_type="nf4",
                bnb_4bit_compute_dtype=torch.bfloat16,
                bnb_4bit_use_double_quant=True,
            )
            self._model = AutoModelForCausalLM.from_pretrained(
                settings.model_id,
                quantization_config=bnb_config,
                device_map="auto",
                torch_dtype=torch.bfloat16,
                token=token,
            )
            if cache_dir:
                logger.info("Saving quantized model to cache: %s", cache_dir)
                self._model.save_pretrained(cache_dir)
                self._tokenizer.save_pretrained(cache_dir)
                logger.info("Quantized model cached")

        # TODO: раскомментировать после обучения адаптеров
        # token = settings.hf_token or None
        # logger.info("Loading extraction adapter: %s", settings.extraction_adapter)
        # self._extraction_adapter = settings.extraction_adapter
        # PeftModel.from_pretrained(self._model, self._extraction_adapter, token=token)
        # self._model = self._model.base_model if hasattr(self._model, "base_model") else self._model
        # logger.info("Loading generation adapter: %s", settings.generation_adapter)
        # self._generation_adapter = settings.generation_adapter
        # PeftModel.from_pretrained(self._model, self._generation_adapter, token=token)
        # self._model = self._model.base_model if hasattr(self._model, "base_model") else self._model

        logger.info("Pipeline ready. VRAM used: %.1f GB", torch.cuda.memory_allocated() / 1024**3)

    def _generate(self, messages: list[dict], max_new_tokens: int | None = None) -> str:
        """Генерация ответа через chat template."""
        prompt = self._tokenizer.apply_chat_template(
            messages, tokenize=False, add_generation_prompt=True
        )
        inputs = self._tokenizer(prompt, return_tensors="pt").to(self._model.device)

        with torch.no_grad():
            outputs = self._model.generate(
                **inputs,
                max_new_tokens=max_new_tokens or settings.max_new_tokens,
                temperature=settings.temperature,
                top_p=settings.top_p,
                do_sample=True,
                pad_token_id=self._tokenizer.eos_token_id,
            )

        response = self._tokenizer.decode(
            outputs[0][inputs["input_ids"].shape[1] :], skip_special_tokens=True
        )
        return response.strip()

    # TODO: раскомментировать после обучения адаптеров
    # def _with_adapter(self, adapter_path: str):
    #     token = settings.hf_token or None
    #     model = PeftModel.from_pretrained(self._model, adapter_path, token=token)
    #     model.eval()
    #     return model

    # def _unwrap_adapter(self):
    #     if hasattr(self._model, "base_model"):
    #         self._model = self._model.base_model

    def extract_facts(self, text: str) -> list[dict]:
        """Извлечение фактов из текста."""
        # TODO: заменить на self._with_adapter(self._extraction_adapter) после обучения
        messages = [
            {"role": "system", "content": EXTRACTION_SYSTEM},
            {"role": "user", "content": text},
        ]
        response = self._generate(messages)
        return _parse_json(response)

    def generate_quiz(self, facts: list[dict]) -> dict:
        """Генерация квиза из фактов."""
        # TODO: заменить на self._with_adapter(self._generation_adapter) после обучения
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
