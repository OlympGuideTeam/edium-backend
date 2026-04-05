"""
GenerationConsumer — подписывается на riddler.generation.requested,
сохраняет задачи в таблицу generation_task (inbox).
"""

import asyncio
import json
import logging

import asyncpg
import nats.aio.client

from app.config import settings

logger = logging.getLogger(__name__)

SUBJECT = "riddler.generation.requested"
QUEUE   = "sphinx-generation-consumers"


class GenerationConsumer:
    def __init__(self, nc: nats.aio.client.Client, pool: asyncpg.Pool):
        self._nc   = nc
        self._pool = pool

    async def run(self) -> None:
        await self._nc.subscribe(SUBJECT, queue=QUEUE, cb=self._handle)
        logger.info("GenerationConsumer subscribed to %s (queue=%s)", SUBJECT, QUEUE)
        # Держим воркер живым — сам nats-клиент обрабатывает входящие сообщения
        while True:
            await asyncio.sleep(3600)

    async def _handle(self, msg: nats.aio.client.Msg) -> None:
        try:
            data = json.loads(msg.data)
        except json.JSONDecodeError as e:
            logger.error("GenerationConsumer: invalid JSON: %s", e)
            return

        job_id   = data.get("job_id")
        quiz_id  = data.get("quiz_id")
        text     = data.get("text")
        trace_ctx = data.get("trace_ctx", "")

        if not job_id or not quiz_id or not text:
            logger.error("GenerationConsumer: missing fields in message: %s", data)
            return

        async with self._pool.acquire() as conn:
            try:
                await conn.execute(
                    """
                    INSERT INTO generation_task (job_id, quiz_id, text, trace_ctx)
                    VALUES ($1, $2, $3, $4)
                    ON CONFLICT (job_id) DO NOTHING
                    """,
                    job_id, quiz_id, text, trace_ctx,
                )
                logger.info("GenerationConsumer: saved task job_id=%s quiz_id=%s", job_id, quiz_id)
            except Exception as e:
                logger.error("GenerationConsumer: db error: %s", e)
