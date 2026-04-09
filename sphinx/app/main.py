import asyncio
import logging
import signal
import sys

from app.config import settings
from app.pipeline import pipeline
from app.infra.db import create_pool
from app.infra.nats_client import connect as nats_connect
from app.worker.consumer import GenerationConsumer
from app.worker.processor import GenerationProcessor
from app.worker.publisher import GenerationPublisher

logging.basicConfig(
    level=logging.INFO,
    format='{"time":"%(asctime)s","level":"%(levelname)s","logger":"%(name)s","msg":"%(message)s"}',
)
logger = logging.getLogger(__name__)


async def main() -> None:
    logger.info("Starting Sphinx")

    pipeline.load()
    logger.info("Model loaded successfully")

    pool = await create_pool(settings.postgres_dsn)
    nc   = await nats_connect()

    consumer  = GenerationConsumer(nc, pool)
    processor = GenerationProcessor(pool)
    publisher = GenerationPublisher(nc, pool)

    stop_event = asyncio.Event()

    def _shutdown(signum, frame):
        logger.info("Shutdown signal received")
        stop_event.set()

    signal.signal(signal.SIGTERM, _shutdown)
    signal.signal(signal.SIGINT, _shutdown)

    workers = asyncio.gather(
        consumer.run(),
        processor.run(),
        publisher.run(),
    )
    workers = asyncio.ensure_future(workers)

    await stop_event.wait()
    workers.cancel()
    try:
        await workers
    except asyncio.CancelledError:
        pass

    await nc.drain()
    await pool.close()
    logger.info("Sphinx stopped")
    sys.exit(0)


if __name__ == "__main__":
    asyncio.run(main())
