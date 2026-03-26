run:
	docker-compose up -d --build

down:
	docker-compose down

clean:
	docker-compose down --volumes

genrsa:
	openssl genrsa -out private.pem 2048

# === Линтеры ===

lint:
	pre-commit run --all-files

lint-go:
	cd doorman && golangci-lint run
	cd herald && golangci-lint run

lint-python:
	ruff check ml/
	ruff format --check ml/

fmt:
	cd doorman && goimports -w . && gofmt -w .
	cd herald && goimports -w . && gofmt -w .
	ruff format ml/

install-hooks:
	pre-commit install
