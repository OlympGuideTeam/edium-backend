"""
GenerationProcessor — опрашивает generation_task, запускает pipeline,
сохраняет результат в generation_result (outbox).

При старте и каждые RECOVERY_INTERVAL секунд сбрасывает задачи, застрявшие
в статусе 'processing' дольше processing_timeout_minutes (сценарий: VM была
прервана во время выполнения pipeline). Если attempts >= max_task_attempts —
задача помечается как 'failed'.
"""

import asyncio
import json
import logging
from datetime import datetime, timezone

import asyncpg

from app.config import settings
from app.pipeline import pipeline

logger = logging.getLogger(__name__)

RECOVERY_INTERVAL = 300  # секунды между проверками зависших задач


class GenerationProcessor:
    def __init__(self, pool: asyncpg.Pool):
        self._pool = pool

    async def run(self) -> None:
        logger.info("GenerationProcessor started (poll_interval=%.1fs)", settings.worker_poll_interval)

        try:
            await self._recover_orphaned_tasks()
            await self._recover_stuck_tasks()
        except Exception as e:
            logger.error("GenerationProcessor: startup recovery failed: %s", e)
        last_recovery = asyncio.get_event_loop().time()

        while True:
            try:
                await self._process_one()
            except Exception as e:
                logger.error("GenerationProcessor: unexpected error: %s", e)
            await asyncio.sleep(settings.worker_poll_interval)

            now = asyncio.get_event_loop().time()
            if now - last_recovery >= RECOVERY_INTERVAL:
                await self._recover_stuck_tasks()
                last_recovery = now

    async def _recover_orphaned_tasks(self) -> None:
        """При старте сбрасывает ВСЕ задачи в статусе 'processing' — они остались от
        предыдущего процесса и точно не обрабатываются."""
        async with self._pool.acquire() as conn:
            rows = await conn.fetch(
                """
                UPDATE generation_task
                SET status = 'pending', started_at = NULL
                WHERE status = 'processing'
                RETURNING job_id, attempts
                """
            )
        for row in rows:
            logger.info(
                "GenerationProcessor: reset orphaned task job_id=%s (attempts=%d) to pending on startup",
                row["job_id"], row["attempts"],
            )

    async def _recover_stuck_tasks(self) -> None:
        """
        Сбрасывает задачи, застрявшие в 'processing' из-за прерывания VM.
        Если задача уже исчерпала попытки — помечает как 'failed'.
        """
        async with self._pool.acquire() as conn:
            rows = await conn.fetch(
                """
                UPDATE generation_task
                SET
                    status     = CASE
                                   WHEN attempts >= $2 THEN 'failed'
                                   ELSE 'pending'
                                 END,
                    started_at = NULL,
                    error      = CASE
                                   WHEN attempts >= $2 THEN 'превышено max_task_attempts'
                                   ELSE error
                                 END
                WHERE
                    status = 'processing'
                    AND started_at < NOW() - ($1 * INTERVAL '1 minute')
                RETURNING job_id, attempts
                """,
                settings.processing_timeout_minutes,
                settings.max_task_attempts,
            )

        for row in rows:
            if row["attempts"] >= settings.max_task_attempts:
                logger.warning(
                    "GenerationProcessor: task job_id=%s marked failed after %d attempts",
                    row["job_id"], row["attempts"],
                )
            else:
                logger.info(
                    "GenerationProcessor: recovered stuck task job_id=%s (attempts=%d, will retry)",
                    row["job_id"], row["attempts"],
                )

    async def _process_one(self) -> None:
        async with self._pool.acquire() as conn:
            async with conn.transaction():
                row = await conn.fetchrow(
                    """
                    SELECT id, job_id, quiz_id, text, trace_ctx
                    FROM generation_task
                    WHERE status = 'pending'
                    ORDER BY created_at
                    LIMIT 1
                    FOR UPDATE SKIP LOCKED
                    """
                )
                if row is None:
                    return

                task_id   = row["id"]
                job_id    = row["job_id"]
                quiz_id   = row["quiz_id"]
                text      = row["text"]
                trace_ctx = row["trace_ctx"] or ""

                await conn.execute(
                    """
                    UPDATE generation_task
                    SET status = 'processing', started_at = $1, attempts = attempts + 1
                    WHERE id = $2
                    """,
                    datetime.now(timezone.utc), task_id,
                )

        logger.info("GenerationProcessor: running pipeline job_id=%s", job_id)
        try:
            loop   = asyncio.get_event_loop()
            result = await loop.run_in_executor(None, pipeline.run, text)
            await self._save_result(job_id, quiz_id, result, trace_ctx, task_id)
        except Exception as e:
            logger.error("GenerationProcessor: pipeline failed job_id=%s: %s", job_id, e)
            async with self._pool.acquire() as conn:
                await conn.execute(
                    """
                    UPDATE generation_task
                    SET status = 'failed', finished_at = $1, error = $2
                    WHERE id = $3
                    """,
                    datetime.now(timezone.utc), str(e), task_id,
                )

    async def _save_result(
        self,
        job_id: str,
        quiz_id: str,
        result: dict,
        trace_ctx: str,
        task_id,
    ) -> None:
        payload = json.dumps(
            {
                "job_id":    job_id,
                "quiz_id":   quiz_id,
                "questions": result.get("quiz", {}).get("questions", []),
                "trace_ctx": trace_ctx,
            },
            ensure_ascii=False,
        )

        now = datetime.now(timezone.utc)
        async with self._pool.acquire() as conn:
            async with conn.transaction():
                await conn.execute(
                    """
                    INSERT INTO generation_result (job_id, quiz_id, payload, trace_ctx)
                    VALUES ($1, $2, $3::jsonb, $4)
                    ON CONFLICT (job_id) DO NOTHING
                    """,
                    job_id, quiz_id, payload, trace_ctx,
                )
                await conn.execute(
                    "UPDATE generation_task SET status = 'done', finished_at = $1 WHERE id = $2",
                    now, task_id,
                )

        logger.info("GenerationProcessor: result saved job_id=%s", job_id)
