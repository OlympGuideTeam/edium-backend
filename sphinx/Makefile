.PHONY: proto-python proto-go proto

# Генерация Python stubs
proto-python:
	python3 -m grpc_tools.protoc \
		-Iproto \
		--python_out=app/generated \
		--grpc_python_out=app/generated \
		proto/sphinx.proto
	sed -i.bak 's/^import sphinx_pb2/from app.generated import sphinx_pb2/' \
		app/generated/sphinx_pb2_grpc.py
	rm -f app/generated/sphinx_pb2_grpc.py.bak

# Генерация Go stubs
proto-go:
	protoc \
		-Iproto \
		--go_out=pkg/sphinxpb --go_opt=paths=source_relative \
		--go-grpc_out=pkg/sphinxpb --go-grpc_opt=paths=source_relative \
		proto/sphinx.proto

# Все stubs
proto: proto-python proto-go
