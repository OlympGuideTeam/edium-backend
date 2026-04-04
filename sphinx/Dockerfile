FROM nvidia/cuda:12.1.1-runtime-ubuntu22.04

RUN apt-get update && \
    apt-get install -y --no-install-recommends python3.11 python3-pip python3.11-venv && \
    rm -rf /var/lib/apt/lists/* && \
    update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.11 1

WORKDIR /app

COPY requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

COPY proto/ proto/
COPY app/ app/

# Генерация Python stubs из proto + фикс импортов
RUN python3 -m grpc_tools.protoc \
    -Iproto \
    --python_out=app/generated \
    --grpc_python_out=app/generated \
    proto/sphinx.proto && \
    sed -i 's/^import sphinx_pb2/from app.generated import sphinx_pb2/' \
    app/generated/sphinx_pb2_grpc.py

EXPOSE 50051

CMD ["python3", "-m", "app.main"]
