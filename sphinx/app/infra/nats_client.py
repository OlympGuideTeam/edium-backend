import ssl
import logging

import nats

from app.config import settings

logger = logging.getLogger(__name__)


async def connect() -> nats.aio.client.Client:
    """Подключение к NATS с опциональным TLS (для внешнего stunnel-эндпоинта)."""
    options: dict = {
        "servers": [settings.nats_url],
        "reconnect_time_wait": 2,
        "max_reconnect_attempts": -1,  # переподключаться бесконечно
        "error_cb": _on_error,
        "reconnected_cb": _on_reconnect,
        "disconnected_cb": _on_disconnect,
    }

    if settings.nats_tls_ca_path:
        ctx = ssl.create_default_context(cafile=settings.nats_tls_ca_path)
        options["tls"] = ctx
        logger.info("NATS TLS enabled (CA: %s)", settings.nats_tls_ca_path)

    nc = await nats.connect(**options)
    logger.info("NATS connected: %s", settings.nats_url)
    return nc


async def _on_error(e: Exception) -> None:
    logger.error("NATS error: %s", e)


async def _on_reconnect() -> None:
    logger.warning("NATS reconnected")


async def _on_disconnect() -> None:
    logger.warning("NATS disconnected")
