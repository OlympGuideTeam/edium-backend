"""
GenerationConsumer — JetStream durable-подписчик на riddler.generation.requested.

Использует JetStream вместо Core NATS, чтобы сообщения не терялись при прерывании VM.
При старте создаёт стрим RIDDLER_GEN (если ещё не существует).
Явное ACK только после успешной записи в БД; NAK при ошибке БД (NATS переотправит);
TERM для невалидных сообщений (переотправлять не нужно).
"""

import asyncio
import json
import logging

import asyncpg
import nats
import nats.aio.client
import nats.js.errors

from app.config import settings

logger = logging.getLogger(__name__)

SUBJECT  = "riddler.generation.requested"
STREAM   = "RIDDLER_GEN"
CONSUMER = "sphinx-generation"


class GenerationConsumer:
    def __init__(self, nc: nats.aio.client.Client, pool: asyncpg.Pool):
        self._nc   = nc
        self._pool = pool

    async def run(self) -> None:
        js = self._nc.jetstream()

        await self._ensure_stream(js)

        await js.subscribe(
            SUBJECT,
            durable=CONSUMER,
            cb=self._handle,
            manual_ack=True,
        )
        logger.info(
            "GenerationConsumer: JetStream durable consumer %s/%s subscribed to %s",
            STREAM, CONSUMER, SUBJECT,
        )

        while True:
            await asyncio.sleep(3600)

    async def _ensure_stream(self, js) -> None:
        try:
            await js.stream_info(STREAM)
            logger.info("GenerationConsumer: stream %s already exists", STREAM)
        except nats.js.errors.NotFoundError:
            await js.add_stream(name=STREAM, subjects=[SUBJECT])
            logger.info("GenerationConsumer: stream %s created", STREAM)

    async def _handle(self, msg: nats.aio.client.Msg) -> None:
        try:
            data = json.loads(msg.data)
        except json.JSONDecodeError as e:
            logger.error("GenerationConsumer: invalid JSON: %s", e)
            await msg.term()
            return

        job_id    = data.get("job_id")
        quiz_id   = data.get("quiz_id")
        text      = data.get("text")
        trace_ctx = data.get("trace_ctx", "")

        if not job_id or not quiz_id or not text:
            logger.error("GenerationConsumer: missing fields in message: %s", data)
            await msg.term()
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
                await msg.ack()
                logger.info("GenerationConsumer: saved task job_id=%s quiz_id=%s", job_id, quiz_id)
            except Exception as e:
                logger.error("GenerationConsumer: db error, will redeliver job_id=%s: %s", job_id, e)
                await msg.nak()
