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

  actual=$(curl -sf -o /dev/null -w "%{http_code}" -X "$method" "$url" 2>/dev/null || echo "000")
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

  body=$(curl -s "$url" 2>/dev/null)
  if echo "$body" | grep -q "$pattern"; then
    echo "PASS  $desc"
    PASS=$((PASS + 1))
  else
    echo "FAIL  $desc — pattern '$pattern' not found"
    echo "      response: $(echo "$body" | head -c 300)"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Smoke tests: $BASE_URL ==="
echo ""

# Doorman
check_body   "doorman  JWKS returns public keys"        "$BASE_URL/doorman/v1/.well-known/jwks.json" '"keys"'
check_status "doorman  OTP send: empty body → 400"      400  POST  "$BASE_URL/doorman/v1/otp/send"

# Caesar
check_status "caesar   /users/me requires auth → 401"   401  GET   "$BASE_URL/caesar/v1/users/me"

# Riddler
check_status "riddler  /quizzes requires auth → 401"    401  GET   "$BASE_URL/riddler/v1/quizzes"

# Louvre
check_status "louvre   /images/upload requires auth → 401"  401  POST  "$BASE_URL/louvre/v1/images/upload"

# Herald
check_status "herald   /notifications requires auth → 401"  401  GET   "$BASE_URL/herald/v1/notifications"

# Caddy routing
check_status "caddy    unknown route → 404"              404  GET   "$BASE_URL/does-not-exist"

echo ""
if [ "$FAIL" -eq 0 ]; then
  echo "All $PASS tests passed"
else
  echo "$PASS passed, $FAIL failed"
  exit 1
fi
