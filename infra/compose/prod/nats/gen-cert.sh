#!/usr/bin/env bash
# Генерация самоподписанного TLS-сертификата для stunnel (NATS external endpoint).
# Запускать один раз на VM:
#   chmod +x gen-cert.sh && ./gen-cert.sh api.edium.online
#
# Результат:
#   tls.crt  — сертификат сервера (монтировать в stunnel контейнер)
#   tls.key  — приватный ключ (монтировать в stunnel контейнер)
#   ca.crt   — корневой CA (скопировать в sphinx/.env как NATS_TLS_CA_CONTENT или передать файлом)

set -euo pipefail

DOMAIN="${1:-api.edium.online}"
DAYS=825   # максимум для самоподписанного
DIR="$(cd "$(dirname "$0")" && pwd)"

echo "[gen-cert] domain=${DOMAIN}, output=${DIR}"

# CA
openssl genrsa -out "${DIR}/ca.key" 4096
openssl req -new -x509 -days "${DAYS}" \
    -key "${DIR}/ca.key" \
    -out "${DIR}/ca.crt" \
    -subj "/CN=Edium NATS CA"

# Server key + CSR
openssl genrsa -out "${DIR}/tls.key" 2048
openssl req -new \
    -key "${DIR}/tls.key" \
    -out "${DIR}/tls.csr" \
    -subj "/CN=${DOMAIN}"

# Sign
cat > "${DIR}/tls.ext" <<EOF
subjectAltName=DNS:${DOMAIN},IP:$(hostname -I | awk '{print $1}')
EOF

openssl x509 -req -days "${DAYS}" \
    -in  "${DIR}/tls.csr" \
    -CA  "${DIR}/ca.crt" \
    -CAkey "${DIR}/ca.key" \
    -CAcreateserial \
    -extfile "${DIR}/tls.ext" \
    -out "${DIR}/tls.crt"

rm -f "${DIR}/tls.csr" "${DIR}/tls.ext" "${DIR}/ca.key" "${DIR}/ca.srl"
echo "[gen-cert] готово: tls.crt, tls.key, ca.crt"
echo "[gen-cert] ca.crt — передать на Sphinx VM (NATS_TLS_CA_PATH)"
