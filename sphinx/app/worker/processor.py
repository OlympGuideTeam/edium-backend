"""
GenerationProcessor — опрашивает generation_task, запускает pipeline,
сохраняет результат в generation_result (outbox).
"""

import asyncio
import json
import logging
from datetime import datetime, timezone

import asyncpg

from app.config import settings
from app.pipeline import pipeline

logger = logging.getLogger(__name__)


class GenerationProcessor:
    def __init__(self, pool: asyncpg.Pool):
        self._pool = pool

    async def run(self) -> None:
        logger.info("GenerationProcessor started (poll_interval=%.1fs)", settings.worker_poll_interval)
        while True:
            try:
                await self._process_one()
            except Exception as e:
                logger.error("GenerationProcessor: unexpected error: %s", e)
            await asyncio.sleep(settings.worker_poll_interval)

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

                task_id  = row["id"]
                job_id   = row["job_id"]
                quiz_id  = row["quiz_id"]
                text     = row["text"]
                trace_ctx = row["trace_ctx"] or ""

                await conn.execute(
                    "UPDATE generation_task SET status='processing', started_at=$1 WHERE id=$2",
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
                    SET status='failed', finished_at=$1, error=$2
                    WHERE id=$3
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
                    "UPDATE generation_task SET status='done', finished_at=$1 WHERE id=$2",
                    now, task_id,
                )

        logger.info("GenerationProcessor: result saved job_id=%s", job_id)
