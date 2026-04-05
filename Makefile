run:
	docker-compose up -d --build

run-podman: build-snowflake
	podman compose up -d --build

down:
	docker-compose down

down-podman:
	podman compose down

clean:
	docker-compose down --volumes

clean-podman:
	podman compose down --volumes

build-snowflake:
	@if [ ! -f tor/snowflake-client ]; then \
		echo "Сборка snowflake-client для linux/arm64..."; \
		git clone --depth=1 https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/snowflake.git /tmp/snowflake-build && \
		cd /tmp/snowflake-build/client && GOOS=linux GOARCH=arm64 go build -o $(CURDIR)/tor/snowflake-client . && \
		rm -rf /tmp/snowflake-build; \
	fi

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
