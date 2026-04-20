import ssl
import logging
from typing import Optional

import nats

from app.config import settings

logger = logging.getLogger(__name__)


async def connect() -> nats.aio.client.Client:
    return await _connect(settings.nats_url, settings.nats_tls_ca_path)


async def connect_optional(url: str, ca_path: str) -> Optional[nats.aio.client.Client]:
    if not url:
        return None
    return await _connect(url, ca_path)


async def _connect(url: str, ca_path: str) -> nats.aio.client.Client:
    options: dict = {
        "servers": [url],
        "reconnect_time_wait": 2,
        "max_reconnect_attempts": -1,
        "error_cb": _on_error,
        "reconnected_cb": _on_reconnect,
        "disconnected_cb": _on_disconnect,
    }

    if ca_path:
        ctx = ssl.create_default_context(cafile=ca_path)
        ctx.check_hostname = False
        options["tls"] = ctx
        logger.info("NATS TLS enabled (CA: %s)", ca_path)

    nc = await nats.connect(**options)
    logger.info("NATS connected: %s", url)
    return nc


async def _on_error(e: Exception) -> None:
    logger.error("NATS error: %s", e)


async def _on_reconnect() -> None:
    logger.warning("NATS reconnected")


async def _on_disconnect() -> None:
    logger.warning("NATS disconnected")
