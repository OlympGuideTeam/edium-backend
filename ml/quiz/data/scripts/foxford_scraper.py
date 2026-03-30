"""
Парсер Foxford Wiki для сбора учебных текстов.

Предметы: Биология, Обществознание, География, История, Информатика
Исключения: "ПОДГОТОВКА К ЭКЗАМЕНУ" (История), "ЕГЭ" (Информатика)

Логика агрегации:
- Если текст статьи < 2000 символов — объединяем со следующими статьями из той же темы
- Если после всех статей темы всё равно < 2000 символов — пропускаем
- Иначе сохраняем как отдельный чанк

Вывод: ml/foxford_data/{subject}/{topic_slug}/chunk_{N:03d}.txt
"""

import asyncio
import re
from pathlib import Path

from playwright.async_api import BrowserContext, Page, async_playwright

# Конфигурация

SUBJECTS = {
    "obschestvoznanie": "https://foxford.ru/wiki/obschestvoznanie",
    "biologiya": "https://foxford.ru/wiki/biologiya",
    "geografiya": "https://foxford.ru/wiki/geografiya",
    "istoriya": "https://foxford.ru/wiki/istoriya",
    "informatika": "https://foxford.ru/wiki/informatika",
}

# Глобальные темы, которые нужно пропустить (subject -> set of topic names upper)
EXCLUDED_TOPICS: dict[str, set[str]] = {
    "istoriya": {"ПОДГОТОВКА К ЭКЗАМЕНУ"},
    "informatika": {"ЕГЭ"},
}

MIN_CHARS = 2000
OUTPUT_DIR = Path(__file__).parent / "foxford_data"
DELAY_BETWEEN_PAGES = 1.5  # секунды между запросами


# Стоп-фразы для обрезки конца статьи

# Всё от этой фразы (включительно) в тексте будет отрезано
STOP_HEADINGS = re.compile(
    r"^(Вопросы и задания|Задания|Тест|Проверь себя|Задачи для самопроверки)\s*$",
    re.IGNORECASE,
)


# Playwright helpers


def make_context_kwargs() -> dict:
    return dict(
        user_agent=(
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
            "AppleWebKit/537.36 (KHTML, like Gecko) "
            "Chrome/120.0.0.0 Safari/537.36"
        ),
        viewport={"width": 1280, "height": 800},
        extra_http_headers={"Accept-Language": "ru-RU,ru;q=0.9,en;q=0.8"},
    )


async def new_page(context: BrowserContext) -> Page:
    page = await context.new_page()
    return page


# Извлечение ссылок со страницы предмета


async def get_subject_topics(page: Page, subject: str, url: str) -> list[dict]:
    """
    Возвращает список тем:
    [
        {"topic": "ОБЩЕСТВО И ЧЕЛОВЕК", "links": [{"href": ..., "text": ...}, ...]},
        ...
    ]
    Темы из EXCLUDED_TOPICS пропускаются.
    """
    excluded = {t.upper() for t in EXCLUDED_TOPICS.get(subject, set())}

    await page.goto(url, timeout=40000)
    await page.wait_for_timeout(3000)

    topics = await page.evaluate("""() => {
        const items = document.querySelectorAll('li:has(h2)');
        return Array.from(items).map(li => {
            const h2 = li.querySelector('h2');
            const topicName = h2 ? h2.innerText.trim() : '';
            const links = li.querySelectorAll('a[href*="/wiki/"]');
            return {
                topic: topicName,
                links: Array.from(links).map(a => ({
                    href: a.href,
                    text: a.innerText.trim()
                }))
            };
        }).filter(t => t.topic && t.links.length > 0);
    }""")

    result = []
    for t in topics:
        if t["topic"].upper() in excluded:
            print(f"  [пропуск] {t['topic']}")
            continue
        result.append(t)

    return result


# Извлечение текста со страницы статьи


async def get_article_text(page: Page, url: str) -> str:
    """
    Возвращает очищенный текст статьи.
    Исключает рекламные баннеры и всё после раздела "Вопросы и задания".
    """
    try:
        await page.goto(url, timeout=40000)
        await page.wait_for_timeout(2000)
    except Exception as e:
        print(f"    [ошибка загрузки] {url}: {e}")
        return ""

    raw_text: str = await page.evaluate("""() => {
        const content = document.querySelector('.InteractiveContent');
        if (!content) return '';

        // Контентная обёртка — второй дочерний элемент (первый — пустой div)
        const wrapper = content.children.length > 1 ? content.children[1] : content;

        const parts = [];
        for (const child of wrapper.children) {
            const cls = child.className || '';

            // Пропускаем рекламные баннеры
            if (/embedded-banner-wrapper/i.test(cls)) continue;

            // Пропускаем блоки с ответами на задания
            if (/toggle_element/i.test(cls)) continue;

            const text = (child.innerText || '').trim();
            if (text) parts.push(text);
        }
        return parts.join('\\n\\n');
    }""")

    if not raw_text:
        return ""

    return _cut_after_stop_heading(raw_text)


def _cut_after_stop_heading(text: str) -> str:
    """Обрезает текст начиная с раздела вопросов/заданий."""
    lines = text.split("\n")
    result_lines = []
    for line in lines:
        stripped = line.strip()
        if STOP_HEADINGS.match(stripped):
            break
        result_lines.append(line)
    return "\n".join(result_lines).strip()


# Сохранение чанка


def slugify(text: str) -> str:
    text = text.lower()
    text = re.sub(r"[^а-яёa-z0-9]+", "_", text)
    text = text.strip("_")
    return text[:60]


def save_chunk(subject: str, topic: str, chunk_n: int, text: str) -> Path:
    topic_dir = OUTPUT_DIR / subject / slugify(topic)
    topic_dir.mkdir(parents=True, exist_ok=True)
    path = topic_dir / f"chunk_{chunk_n:03d}.txt"
    path.write_text(text, encoding="utf-8")
    return path


# Основная логика обхода предмета


async def process_subject(context: BrowserContext, subject: str, subject_url: str) -> None:
    page = await new_page(context)
    print(f"\n{'=' * 60}")
    print(f"Предмет: {subject}")
    print(f"{'=' * 60}")

    topics = await get_subject_topics(page, subject, subject_url)
    print(f"Тем: {len(topics)}")

    for topic_info in topics:
        topic_name = topic_info["topic"]
        links = topic_info["links"]
        print(f"\n  Тема: {topic_name} ({len(links)} статей)")

        chunk_n = 1
        accumulated_texts: list[str] = []
        accumulated_len = 0
        accumulated_sources: list[str] = []

        for i, link in enumerate(links):
            await asyncio.sleep(DELAY_BETWEEN_PAGES)
            try:
                text = await get_article_text(page, link["href"])
            except Exception as e:
                print(f"    [ошибка] {link['text']}: {e}")
                continue
            if not text:
                print(f"    [пусто] {link['text']}")
                continue

            accumulated_texts.append(text)
            accumulated_sources.append(link["text"])
            accumulated_len += len(text)

            print(f"    [{i + 1}/{len(links)}] {link['text'][:50]} — {len(text)} симв.")

            if accumulated_len >= MIN_CHARS:
                combined = "\n\n".join(accumulated_texts)
                path = save_chunk(subject, topic_name, chunk_n, combined)
                print(f"    → сохранено: {path.name} ({accumulated_len} симв.)")
                chunk_n += 1
                accumulated_texts = []
                accumulated_len = 0
                accumulated_sources = []

        # Остаток в конце темы
        if accumulated_texts:
            if accumulated_len >= MIN_CHARS:
                combined = "\n\n".join(accumulated_texts)
                path = save_chunk(subject, topic_name, chunk_n, combined)
                print(f"    → сохранено: {path.name} ({accumulated_len} симв.)")
            else:
                print(f"    [пропуск] остаток {accumulated_len} симв. < {MIN_CHARS}")

    await page.close()


# Точка входа


async def main(subjects_filter: list[str] | None = None) -> None:
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    subjects = {k: v for k, v in SUBJECTS.items() if subjects_filter is None or k in subjects_filter}

    async with async_playwright() as p:
        browser = await p.chromium.launch(
            headless=True,
            args=["--no-sandbox", "--disable-blink-features=AutomationControlled"],
        )
        context = await browser.new_context(**make_context_kwargs())
        await context.add_init_script("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")

        for subject, url in subjects.items():
            await process_subject(context, subject, url)

        await browser.close()

    print("\nГотово!")
    total_files = list(OUTPUT_DIR.rglob("*.txt"))
    total_chars = sum(f.read_text(encoding="utf-8").__len__() for f in total_files)
    print(f"Файлов: {len(total_files)}, суммарно символов: {total_chars:,}")


if __name__ == "__main__":
    import sys

    filter_subjects = sys.argv[1:] if len(sys.argv) > 1 else None
    if filter_subjects:
        unknown = [s for s in filter_subjects if s not in SUBJECTS]
        if unknown:
            print(f"Неизвестные предметы: {unknown}")
            print(f"Доступные: {list(SUBJECTS.keys())}")
            sys.exit(1)
        print(f"Запуск для: {filter_subjects}")
    asyncio.run(main(filter_subjects))
