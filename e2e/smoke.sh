#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-https://test.edium.online}"
PASS=0
FAIL=0

check_status() {
  local desc="$1"
  local expected="$2"
  local method="${3:-GET}"
  local url="$4"

  actual=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 10 --retry 3 --retry-delay 3 \
    -X "$method" "$url" 2>/dev/null || echo "000")
  if [ "$actual" = "$expected" ]; then
    echo "PASS  $desc"
    PASS=$((PASS + 1))
  else
    echo "FAIL  $desc — expected HTTP $expected, got $actual"
    FAIL=$((FAIL + 1))
  fi
}

check_body() {
  local desc="$1"
  local url="$2"
  local pattern="$3"

  body=$(curl -s --connect-timeout 10 --retry 3 --retry-delay 3 "$url" 2>/dev/null)
  if echo "$body" | grep -q "$pattern"; then
    echo "PASS  $desc"
    PASS=$((PASS + 1))
  else
    echo "FAIL  $desc — pattern '$pattern' not found"
    echo "      response: $(echo "$body" | head -c 300)"
    FAIL=$((FAIL + 1))
  fi
}

# Проверяет что ответ содержит JSON с полем "error" (auth middleware работает корректно)
check_auth_error() {
  local desc="$1"
  local method="${2:-GET}"
  local url="$3"

  body=$(curl -s --connect-timeout 10 -X "$method" \
    -H "Content-Type: application/json" "$url" 2>/dev/null)
  if echo "$body" | grep -q '"error"'; then
    echo "PASS  $desc"
    PASS=$((PASS + 1))
  else
    echo "FAIL  $desc — no 'error' field in 401 body"
    echo "      response: $(echo "$body" | head -c 300)"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Smoke tests: $BASE_URL ==="
echo ""

# Doorman — публичные эндпоинты
check_body   "doorman  JWKS: статус 200 и поле 'keys'"        "$BASE_URL/doorman/v1/.well-known/jwks.json" '"keys"'
check_body   "doorman  JWKS: первый ключ содержит 'kty'"      "$BASE_URL/doorman/v1/.well-known/jwks.json" '"kty"'
check_status "doorman  OTP send: пустое тело → 400"           400  POST  "$BASE_URL/doorman/v1/otp/send"
check_status "doorman  refresh: невалидный токен → 400/401"   400  POST  "$BASE_URL/doorman/v1/auth/refresh"

# Caesar — auth middleware
check_status     "caesar   /users/me без токена → 401"         401  GET   "$BASE_URL/caesar/v1/users/me"
check_auth_error "caesar   /users/me возвращает JSON с error"  GET        "$BASE_URL/caesar/v1/users/me"

# Riddler
check_status     "riddler  /quizzes без токена → 401"          401  GET   "$BASE_URL/riddler/v1/quizzes"
check_auth_error "riddler  /quizzes возвращает JSON с error"   GET        "$BASE_URL/riddler/v1/quizzes"

# Louvre
check_status "louvre   /images/upload без токена → 401"        401  POST  "$BASE_URL/louvre/v1/images/upload"

# Herald
check_status "herald   /notifications без токена → 401"        401  GET   "$BASE_URL/herald/v1/notifications"

# Caddy routing
check_status "caddy    неизвестный роут → 404"                 404  GET   "$BASE_URL/does-not-exist"

echo ""
if [ "$FAIL" -eq 0 ]; then
  echo "All $PASS tests passed"
else
  echo "$PASS passed, $FAIL failed"
  exit 1
fi
