#!/bin/bash
# Генерация Python и Go stubs из sphinx.proto
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Generating Python stubs..."
python3 -m grpc_tools.protoc \
    -I"${SCRIPT_DIR}/proto" \
    --python_out="${SCRIPT_DIR}/app/generated" \
    --grpc_python_out="${SCRIPT_DIR}/app/generated" \
    "${SCRIPT_DIR}/proto/sphinx.proto"

# Фикс импортов для Python (grpc_tools генерирует абсолютные импорты)
sed -i.bak 's/^import sphinx_pb2/from app.generated import sphinx_pb2/' \
    "${SCRIPT_DIR}/app/generated/sphinx_pb2_grpc.py"
rm -f "${SCRIPT_DIR}/app/generated/sphinx_pb2_grpc.py.bak"

echo "Generating Go stubs..."
if command -v protoc &> /dev/null; then
    protoc \
        -I"${SCRIPT_DIR}/proto" \
        --go_out="${SCRIPT_DIR}/pkg/sphinxpb" --go_opt=paths=source_relative \
        --go-grpc_out="${SCRIPT_DIR}/pkg/sphinxpb" --go-grpc_opt=paths=source_relative \
        "${SCRIPT_DIR}/proto/sphinx.proto"
    echo "Go stubs generated in pkg/sphinxpb/"
else
    echo "protoc not found — skipping Go stubs (install protoc + protoc-gen-go + protoc-gen-go-grpc)"
fi

echo "Done."
