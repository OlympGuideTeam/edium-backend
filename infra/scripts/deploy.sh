#!/usr/bin/env bash
# deploy.sh — Скрипт деплоя на удалённую VM
# Использование: ./deploy.sh <environment> <vm_host> <ssh_key_path>
# Пример: ./deploy.sh test 51.250.1.1 ~/.ssh/edium_deploy
set -euo pipefail

ENV="${1:?Укажите окружение: test или prod}"
VM_HOST="${2:?Укажите IP адрес VM}"
SSH_KEY="${3:?Укажите путь к SSH-ключу}"

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REMOTE_DIR="/opt/edium"
SSH_OPTS="-o StrictHostKeyChecking=no -i ${SSH_KEY}"

echo "=== Деплой на ${ENV} (${VM_HOST}) ==="

# Определяем Caddyfile
if [ "$ENV" = "test" ]; then
  CADDYFILE="${REPO_ROOT}/infra/caddy/Caddyfile.test"
elif [ "$ENV" = "prod" ]; then
  CADDYFILE="${REPO_ROOT}/infra/caddy/Caddyfile.prod"
else
  echo "Неизвестное окружение: ${ENV}" >&2
  exit 1
fi

echo "--- Копирование файлов ---"

# Docker Compose
scp ${SSH_OPTS} "${REPO_ROOT}/infra/compose/${ENV}/docker-compose.yml" "deploy@${VM_HOST}:${REMOTE_DIR}/docker-compose.yml"

# Caddyfile
scp ${SSH_OPTS} "${CADDYFILE}" "deploy@${VM_HOST}:${REMOTE_DIR}/Caddyfile"

# Миграции
ssh ${SSH_OPTS} "deploy@${VM_HOST}" "mkdir -p ${REMOTE_DIR}/migrations/doorman"
scp ${SSH_OPTS} -r "${REPO_ROOT}/doorman/migrations/"* "deploy@${VM_HOST}:${REMOTE_DIR}/migrations/doorman/"

echo "--- Запуск обновления ---"

ssh ${SSH_OPTS} "deploy@${VM_HOST}" bash -s <<'REMOTE'
set -e
cd /opt/edium

docker compose pull
docker compose up -d --remove-orphans
docker image prune -f

echo "Запущенные контейнеры:"
docker compose ps
REMOTE

echo "=== Деплой на ${ENV} завершён ==="
