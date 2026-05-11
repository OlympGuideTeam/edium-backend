#!/usr/bin/env bash
# Usage: smoke.sh [base_url] [service]
#   service: doorman | caesar | riddler | louvre | herald | all (default: all)
set -euo pipefail

BASE_URL="${1:-https://test.edium.online}"
SERVICE="${2:-all}"
PASS=0
FAIL=0

run_for() { [ "$SERVICE" = "all" ] || [ "$SERVICE" = "$1" ]; }

check_status() {
  local desc="$1" expected="$2" method="${3:-GET}" url="$4"
  actual=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 10 --retry 3 --retry-delay 3 \
    -X "$method" "$url" 2>/dev/null || echo "000")
  if [ "$actual" = "$expected" ]; then
    echo "PASS  $desc"; PASS=$((PASS + 1))
  else
    echo "FAIL  $desc — expected HTTP $expected, got $actual"; FAIL=$((FAIL + 1))
  fi
}

check_body() {
  local desc="$1" url="$2" pattern="$3"
  body=$(curl -s --connect-timeout 10 --retry 3 --retry-delay 3 "$url" 2>/dev/null)
  if echo "$body" | grep -q "$pattern"; then
    echo "PASS  $desc"; PASS=$((PASS + 1))
  else
    echo "FAIL  $desc — pattern '$pattern' not found"
    echo "      response: $(echo "$body" | head -c 300)"; FAIL=$((FAIL + 1))
  fi
}

check_auth_json() {
  local desc="$1" method="${2:-GET}" url="$3"
  body=$(curl -s --connect-timeout 10 -X "$method" -H "Content-Type: application/json" "$url" 2>/dev/null)
  if echo "$body" | grep -q '"error"'; then
    echo "PASS  $desc"; PASS=$((PASS + 1))
  else
    echo "FAIL  $desc — no 'error' field in body"
    echo "      response: $(echo "$body" | head -c 300)"; FAIL=$((FAIL + 1))
  fi
}

echo "=== Smoke tests [${SERVICE}]: $BASE_URL ==="
echo ""

if run_for doorman; then
  check_body   "doorman  JWKS: статус 200 и поле 'keys'"   "$BASE_URL/doorman/v1/.well-known/jwks.json" '"keys"'
  check_body   "doorman  JWKS: первый ключ содержит 'kty'" "$BASE_URL/doorman/v1/.well-known/jwks.json" '"kty"'
  check_status "doorman  OTP send: пустое тело → 400"      400 POST "$BASE_URL/doorman/v1/otp/send"
  check_status "doorman  refresh: нет токена → 400"        400 POST "$BASE_URL/doorman/v1/auth/refresh"
  check_status "doorman  неизвестный роут → 404"           404 GET  "$BASE_URL/does-not-exist"
fi

if run_for caesar; then
  check_status     "caesar   /users/me без токена → 401"        401 GET "$BASE_URL/caesar/v1/users/me"
  check_auth_json  "caesar   /users/me возвращает JSON с error" GET     "$BASE_URL/caesar/v1/users/me"
  check_status     "caesar   /classes/me без токена → 401"      401 GET "$BASE_URL/caesar/v1/classes/me"
fi

if run_for riddler; then
  check_status     "riddler  /quizzes без токена → 401"         401 GET "$BASE_URL/riddler/v1/quizzes"
  check_auth_json  "riddler  /quizzes возвращает JSON с error"  GET     "$BASE_URL/riddler/v1/quizzes"
  check_status     "riddler  /quizzes/my без токена → 401"      401 GET "$BASE_URL/riddler/v1/quizzes/my"
fi

if run_for louvre; then
  check_status "louvre   /images/upload без токена → 401"       401 POST "$BASE_URL/louvre/v1/images/upload"
fi

if run_for herald; then
  check_status "herald   /notifications без токена → 401"       401 GET "$BASE_URL/herald/v1/notifications"
  check_status "herald   /notifications/count без токена → 401" 401 GET "$BASE_URL/herald/v1/notifications/count"
fi

echo ""
if [ "$FAIL" -eq 0 ]; then
  echo "All $PASS tests passed"
else
  echo "$PASS passed, $FAIL failed"
  exit 1
fi
