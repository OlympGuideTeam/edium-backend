import logging
import signal
import sys
from concurrent import futures

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc

from app.config import settings
from app.pipeline import pipeline
from app.generated import sphinx_pb2, sphinx_pb2_grpc

logging.basicConfig(
    level=logging.INFO,
    format='{"time":"%(asctime)s","level":"%(levelname)s","logger":"%(name)s","msg":"%(message)s"}',
)
logger = logging.getLogger(__name__)

# Методы, не требующие API-ключ
_PUBLIC_METHODS = {
    "/grpc.health.v1.Health/Check",
    "/grpc.health.v1.Health/Watch",
}


class ApiKeyInterceptor(grpc.ServerInterceptor):
    """Проверка API-ключа в metadata (header) x-api-key."""

    def __init__(self, api_key: str):
        self._api_key = api_key

    def intercept_service(self, continuation, handler_call_details):
        method = handler_call_details.method
        if method in _PUBLIC_METHODS:
            return continuation(handler_call_details)

        metadata = dict(handler_call_details.invocation_metadata or [])
        token = metadata.get("x-api-key", "")
        if token != self._api_key:
            return grpc.unary_unary_rpc_method_handler(
                lambda req, ctx: ctx.abort(grpc.StatusCode.UNAUTHENTICATED, "Invalid API key")
            )

        return continuation(handler_call_details)


class SphinxServicer(sphinx_pb2_grpc.SphinxServiceServicer):
    def Generate(self, request, context):
        if not pipeline.is_loaded:
            context.abort(grpc.StatusCode.UNAVAILABLE, "Model not loaded yet")

        text = request.text.strip()
        if len(text) < 50:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "Text must be at least 50 characters")

        try:
            result = pipeline.run(text)
        except RuntimeError as e:
            logger.error("Generation failed: %s", e)
            context.abort(grpc.StatusCode.INTERNAL, str(e))

        # Конвертация в protobuf
        facts = []
        for f in result["facts"]:
            facts.append(sphinx_pb2.Fact(
                question=f.get("question", ""),
                answer=f.get("answer", ""),
                type=f.get("type", ""),
            ))

        questions = []
        for q in result["quiz"].get("questions", []):
            answer = ""
            answers = []
            raw_answer = q.get("answer", "")
            if isinstance(raw_answer, list):
                answers = raw_answer
            else:
                answer = str(raw_answer)

            questions.append(sphinx_pb2.Question(
                type=q.get("type", ""),
                question=q.get("question", ""),
                answer=answer,
                answers=answers,
                options=q.get("options", []),
            ))

        return sphinx_pb2.GenerateResponse(
            facts=facts,
            quiz=sphinx_pb2.Quiz(questions=questions),
        )


def serve():
    logger.info("Starting Sphinx gRPC server on port %d", settings.grpc_port)

    # Загрузка модели
    health_servicer = health.HealthServicer()
    health_servicer.set(
        "sphinx.SphinxService",
        health_pb2.HealthCheckResponse.NOT_SERVING,
    )

    try:
        pipeline.load()
        logger.info("Model loaded successfully")
        health_servicer.set(
            "sphinx.SphinxService",
            health_pb2.HealthCheckResponse.SERVING,
        )
    except Exception as e:
        logger.error("Failed to load model: %s", e)
        logger.warning("Server starting without model")

    # Interceptors
    interceptors = []
    if settings.api_key:
        interceptors.append(ApiKeyInterceptor(settings.api_key))
        logger.info("API key auth enabled")
    else:
        logger.warning("API key not set — auth disabled")

    # gRPC сервер
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=settings.max_workers),
        interceptors=interceptors,
    )
    sphinx_pb2_grpc.add_SphinxServiceServicer_to_server(SphinxServicer(), server)
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)

    # TLS терминируется на Caddy, Sphinx слушает plaintext
    server.add_insecure_port(f"[::]:{settings.grpc_port}")
    server.start()
    logger.info("gRPC server listening on port %d", settings.grpc_port)

    # Graceful shutdown
    def shutdown(signum, frame):
        logger.info("Shutting down gRPC server...")
        health_servicer.set(
            "sphinx.SphinxService",
            health_pb2.HealthCheckResponse.NOT_SERVING,
        )
        server.stop(grace=10)
        sys.exit(0)

    signal.signal(signal.SIGTERM, shutdown)
    signal.signal(signal.SIGINT, shutdown)

    server.wait_for_termination()


if __name__ == "__main__":
    serve()
