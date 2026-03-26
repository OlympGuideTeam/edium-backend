run:
	docker-compose up -d --build

run-podman:
	podman compose up -d --build

down:
	docker-compose down

down-podman:
	podman compose down

clean:
	docker-compose down --volumes

clean-podman:
	podman compose down --volumes

genrsa:
	@python3 scripts/genrsa.py

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
