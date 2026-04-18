"""
GenerationPublisher — опрашивает generation_result, публикует в NATS,
помечает запись как опубликованную.
"""

import asyncio
import logging
from datetime import datetime, timezone
from typing import Optional

import asyncpg
import nats.aio.client

from app.config import settings

logger = logging.getLogger(__name__)

SUBJECT = "sphinx.generation.completed"


class GenerationPublisher:
    def __init__(
        self,
        nc_prod: nats.aio.client.Client,
        pool: asyncpg.Pool,
        nc_test: Optional[nats.aio.client.Client] = None,
    ):
        self._nc   = {"prod": nc_prod, "test": nc_test}
        self._pool = pool

    async def run(self) -> None:
        logger.info("GenerationPublisher started (poll_interval=%.1fs)", settings.worker_poll_interval)
        while True:
            try:
                await self._publish_one()
            except Exception as e:
                logger.error("GenerationPublisher: unexpected error: %s", e)
            await asyncio.sleep(settings.worker_poll_interval)

    async def _publish_one(self) -> None:
        async with self._pool.acquire() as conn:
            async with conn.transaction():
                row = await conn.fetchrow(
                    """
                    SELECT id, job_id, payload, env
                    FROM generation_result
                    WHERE published_at IS NULL
                    ORDER BY created_at
                    LIMIT 1
                    FOR UPDATE SKIP LOCKED
                    """
                )
                if row is None:
                    return

                result_id = row["id"]
                job_id    = row["job_id"]
                payload   = row["payload"]
                env       = row["env"]

                nc = self._nc.get(env)
                if nc is None:
                    logger.error(
                        "GenerationPublisher: no NATS connection for env=%s job_id=%s, skipping",
                        env, job_id,
                    )
                    return

                try:
                    await nc.publish(SUBJECT, payload.encode())
                except Exception as e:
                    logger.error("GenerationPublisher: publish failed job_id=%s env=%s: %s", job_id, env, e)
                    return

                await conn.execute(
                    "UPDATE generation_result SET published_at=$1 WHERE id=$2",
                    datetime.now(timezone.utc), result_id,
                )

        logger.info("GenerationPublisher: published job_id=%s env=%s → %s", job_id, env, SUBJECT)
