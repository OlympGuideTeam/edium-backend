FROM python:3.11-slim AS builder

WORKDIR /app

# Системные зависимости для OpenCV, PyMuPDF
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libgl1 \
    libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

FROM python:3.11-slim

WORKDIR /app

# Runtime зависимости
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 \
    libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

COPY ocr/ ./ocr/
COPY quiz/ ./quiz/

# Директория для кэша моделей
ENV HF_HOME=/app/models
ENV YOLO_CONFIG_DIR=/app/models
VOLUME /app/models

EXPOSE 8000

CMD ["python", "-m", "uvicorn", "ocr.api:app", "--host", "0.0.0.0", "--port", "8000"]
