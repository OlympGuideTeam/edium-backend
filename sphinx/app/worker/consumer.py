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

logger = logging.getLogger(__name__)

SUBJECT  = "riddler.generation.requested"
STREAM   = "RIDDLER_GEN"


def _consumer_name(env: str) -> str:
    return f"sphinx-generation-{env}"


class GenerationConsumer:
    def __init__(self, nc: nats.aio.client.Client, pool: asyncpg.Pool, env: str):
        self._nc   = nc
        self._pool = pool
        self._env  = env

    async def run(self) -> None:
        js       = self._nc.jetstream()
        consumer = _consumer_name(self._env)

        await self._ensure_stream(js)

        await js.subscribe(
            SUBJECT,
            durable=consumer,
            cb=self._handle,
            manual_ack=True,
        )
        logger.info(
            "GenerationConsumer[%s]: JetStream durable consumer %s/%s subscribed to %s",
            self._env, STREAM, consumer, SUBJECT,
        )

        while True:
            await asyncio.sleep(3600)

    async def _ensure_stream(self, js) -> None:
        try:
            await js.stream_info(STREAM)
            logger.info("GenerationConsumer[%s]: stream %s already exists", self._env, STREAM)
        except nats.js.errors.NotFoundError:
            await js.add_stream(name=STREAM, subjects=[SUBJECT])
            logger.info("GenerationConsumer[%s]: stream %s created", self._env, STREAM)

    async def _handle(self, msg: nats.aio.client.Msg) -> None:
        try:
            data = json.loads(msg.data)
        except json.JSONDecodeError as e:
            logger.error("GenerationConsumer[%s]: invalid JSON: %s", self._env, e)
            await msg.term()
            return

        job_id    = data.get("job_id")
        quiz_id   = data.get("quiz_id")
        text      = data.get("text")
        trace_ctx = data.get("trace_ctx", "")

        if not job_id or not quiz_id or not text:
            logger.error("GenerationConsumer[%s]: missing fields in message: %s", self._env, data)
            await msg.term()
            return

        async with self._pool.acquire() as conn:
            try:
                result = await conn.execute(
                    """
                    INSERT INTO generation_task (job_id, quiz_id, text, trace_ctx, env)
                    VALUES ($1, $2, $3, $4, $5)
                    ON CONFLICT (job_id) DO NOTHING
                    """,
                    job_id, quiz_id, text, trace_ctx, self._env,
                )
                await msg.ack()
                if result == "INSERT 0 1":
                    logger.info(
                        "GenerationConsumer[%s]: saved task job_id=%s quiz_id=%s",
                        self._env, job_id, quiz_id,
                    )
                else:
                    logger.info(
                        "GenerationConsumer[%s]: task already exists job_id=%s, skipping",
                        self._env, job_id,
                    )
            except Exception as e:
                logger.error(
                    "GenerationConsumer[%s]: db error, will redeliver job_id=%s: %s",
                    self._env, job_id, e,
                )
                await msg.nak()
