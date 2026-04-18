import asyncio
import logging
import signal
import sys

from app.config import settings
from app.pipeline import pipeline
from app.infra.db import create_pool
from app.infra.nats_client import connect, connect_optional
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

    pool    = await create_pool(settings.postgres_dsn)
    nc_prod = await connect()
    nc_test = await connect_optional(settings.nats_url_test, settings.nats_tls_ca_path_test)

    if nc_test:
        logger.info("Test NATS connected: %s", settings.nats_url_test)
    else:
        logger.info("Test NATS not configured, skipping")

    consumer_prod = GenerationConsumer(nc_prod, pool, env="prod")
    processor     = GenerationProcessor(pool)
    publisher     = GenerationPublisher(nc_prod, pool, nc_test=nc_test)

    workers_list = [
        consumer_prod.run(),
        processor.run(),
        publisher.run(),
    ]

    if nc_test:
        consumer_test = GenerationConsumer(nc_test, pool, env="test")
        workers_list.append(consumer_test.run())

    stop_event = asyncio.Event()

    def _shutdown(signum, frame):
        logger.info("Shutdown signal received")
        stop_event.set()

    signal.signal(signal.SIGTERM, _shutdown)
    signal.signal(signal.SIGINT, _shutdown)

    workers = asyncio.ensure_future(asyncio.gather(*workers_list))

    await stop_event.wait()
    workers.cancel()
    try:
        await workers
    except asyncio.CancelledError:
        pass

    await nc_prod.drain()
    if nc_test:
        await nc_test.drain()
    await pool.close()
    logger.info("Sphinx stopped")
    sys.exit(0)


if __name__ == "__main__":
    asyncio.run(main())
