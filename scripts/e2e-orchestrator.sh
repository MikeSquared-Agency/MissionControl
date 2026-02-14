#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:8080/api"
TOKEN="${MC_API_TOKEN:-e2e-test-token}"
AUTH=(-H "Authorization: Bearer $TOKEN")
FAIL=0

pass() { echo "  PASS: $1"; }
fail() { echo "  FAIL: $1 — $2"; FAIL=1; }

echo "=== MissionControl Orchestrator E2E Smoke Tests ==="

# 1. Health
echo "--- Health ---"
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' "${AUTH[@]}" "$BASE/health")
if [ "$HTTP" = "200" ]; then
  pass "GET /api/health -> 200"
else
  fail "GET /api/health" "expected 200, got $HTTP"
fi

# 2. Status (requires auth)
echo "--- Status ---"
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' "${AUTH[@]}" "$BASE/status")
if [ "$HTTP" = "200" ]; then
  pass "GET /api/status -> 200"
else
  fail "GET /api/status" "expected 200, got $HTTP"
fi

# 3. List tasks
echo "--- Tasks ---"
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' "${AUTH[@]}" "$BASE/tasks")
if [ "$HTTP" = "200" ]; then
  pass "GET /api/tasks -> 200"
  if grep -q "mc-test1" /tmp/e2e_body; then
    pass "seed task mc-test1 present"
  else
    fail "seed task" "mc-test1 not found in response"
  fi
else
  fail "GET /api/tasks" "expected 200, got $HTTP"
fi

# 4. Get single task
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' "${AUTH[@]}" "$BASE/tasks/mc-test1")
if [ "$HTTP" = "200" ]; then
  pass "GET /api/tasks/mc-test1 -> 200"
else
  fail "GET /api/tasks/mc-test1" "expected 200, got $HTTP"
fi

# 5. Create task
echo "--- Task CRUD ---"
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' -X POST \
  "${AUTH[@]}" \
  -H "Content-Type: application/json" \
  -d '{"title":"E2E smoke task","stage":"implement","zone":"core"}' \
  "$BASE/tasks")
if [ "$HTTP" = "201" ] || [ "$HTTP" = "200" ]; then
  pass "POST /api/tasks -> $HTTP"
else
  fail "POST /api/tasks" "expected 201 or 200, got $HTTP"
fi

# 6. Gates
echo "--- Gates ---"
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' "${AUTH[@]}" "$BASE/gates")
if [ "$HTTP" = "200" ]; then
  pass "GET /api/gates -> 200"
else
  fail "GET /api/gates" "expected 200, got $HTTP"
fi

# 7. Audit log
echo "--- Audit ---"
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' "${AUTH[@]}" "$BASE/audit")
if [ "$HTTP" = "200" ]; then
  pass "GET /api/audit -> 200"
else
  fail "GET /api/audit" "expected 200, got $HTTP"
fi

# 8. Graph
echo "--- Graph ---"
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' "${AUTH[@]}" "$BASE/graph")
if [ "$HTTP" = "200" ]; then
  pass "GET /api/graph -> 200"
else
  fail "GET /api/graph" "expected 200, got $HTTP"
fi

# 9. Tokens
echo "--- Tokens ---"
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' "${AUTH[@]}" "$BASE/tokens")
if [ "$HTTP" = "200" ]; then
  pass "GET /api/tokens -> 200"
else
  fail "GET /api/tokens" "expected 200, got $HTTP"
fi

# 10. Auth enforcement
echo "--- Auth ---"
HTTP=$(curl -s -o /tmp/e2e_body -w '%{http_code}' "$BASE/status")
if [ "$HTTP" = "401" ]; then
  pass "GET /api/status (no token) -> 401"
else
  fail "Auth enforcement" "expected 401, got $HTTP"
fi

echo ""
if [ "$FAIL" -eq 0 ]; then
  echo "All MissionControl E2E tests passed."
else
  echo "Some MissionControl E2E tests FAILED."
  exit 1
fi
