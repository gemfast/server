#!/bin/bash
# Drive heterogeneous traffic against a running gemfast-server with local-auth
# config so the OTel custom attributes (user.*, error.category, gem.*, mirror.*)
# populate. Requires `jq`.
#
# Usage:
#   bash scripts/exercise_otel.sh [BASE_URL]
#
# Defaults to http://localhost:2020.

set -euo pipefail

BASE_URL="${1:-http://localhost:2020}"
SAMPLE_GEM="${SAMPLE_GEM:-test/fixtures/spec/nokogiri-1.15.3-arm64-darwin.gem}"

if ! command -v jq >/dev/null 2>&1; then
  echo "exercise_otel.sh requires jq" >&2
  exit 1
fi

if [[ ! -f "$SAMPLE_GEM" ]]; then
  echo "sample gem not found at $SAMPLE_GEM" >&2
  exit 1
fi

log() { printf '\n[exercise] %s\n' "$*"; }

log "1. Health probes (anonymous)"
for _ in 1 2 3 4 5; do
  curl -fsS -o /dev/null "$BASE_URL/up"
done

log "2. Mirror redirects (anonymous; auto-instrumented)"
for gem in rails rake bundler minitest activerecord; do
  curl -fsS -o /dev/null "$BASE_URL/api/v1/dependencies?gems=$gem" || true
  curl -fsS -o /dev/null "$BASE_URL/info/$gem" || true
  curl -fsS -o /dev/null "$BASE_URL/versions" || true
done

log "3. Bad auth attempts (populate error.category=auth, user not yet set)"
curl -fsS -o /dev/null -u 'eve:wrong-password' "$BASE_URL/private/api/v1/dependencies?gems=foo" || true
curl -fsS -o /dev/null -u 'eve:wrong-password' "$BASE_URL/private/names" || true
curl -fsS -o /dev/null "$BASE_URL/private/info/missing-gem-x" || true

log "4. Login as alice (write role)"
ALICE_TOKEN=$(curl -fsS -X POST -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"alice-pw"}' \
  "$BASE_URL/admin/api/v1/login" | jq -r '.token')
[[ -n "$ALICE_TOKEN" && "$ALICE_TOKEN" != "null" ]] || { echo "login failed"; exit 1; }

log "5. Mint an API token for alice"
ALICE_API_TOKEN=$(curl -fsS -X POST -H "Authorization: Bearer $ALICE_TOKEN" \
  "$BASE_URL/admin/api/v1/token" | jq -r '.token')
[[ -n "$ALICE_API_TOKEN" && "$ALICE_API_TOKEN" != "null" ]] || { echo "token mint failed"; exit 1; }

log "6. Browse admin endpoints as alice (user.* attrs populate via JWT path)"
for _ in 1 2 3; do
  curl -fsS -o /dev/null -H "Authorization: Bearer $ALICE_TOKEN" "$BASE_URL/admin/api/v1/auth"
  curl -fsS -o /dev/null -H "Authorization: Bearer $ALICE_TOKEN" "$BASE_URL/admin/api/v1/users"
  curl -fsS -o /dev/null -H "Authorization: Bearer $ALICE_TOKEN" "$BASE_URL/admin/api/v1/stats/db"
done

log "7. Upload sample gem as alice (token-auth; user.* via fixed token middleware)"
curl -fsS -o /dev/null -u "alice:$ALICE_API_TOKEN" \
  --data-binary @"$SAMPLE_GEM" \
  -H 'Content-Type: application/octet-stream' \
  "$BASE_URL/private/api/v1/gems" || true

log "8. Hit private read endpoints as alice"
for _ in 1 2 3 4 5; do
  curl -fsS -o /dev/null -u "alice:$ALICE_API_TOKEN" "$BASE_URL/private/info/nokogiri" || true
  curl -fsS -o /dev/null -u "alice:$ALICE_API_TOKEN" "$BASE_URL/private/api/v1/dependencies?gems=nokogiri" || true
  curl -fsS -o /dev/null -u "alice:$ALICE_API_TOKEN" "$BASE_URL/private/names" || true
done

log "9. Bob (read role) tries to mint a token — should 403 → error.category=auth"
BOB_TOKEN=$(curl -fsS -X POST -H 'Content-Type: application/json' \
  -d '{"username":"bob","password":"bob-pw"}' \
  "$BASE_URL/admin/api/v1/login" | jq -r '.token' || echo "")
if [[ -n "$BOB_TOKEN" && "$BOB_TOKEN" != "null" ]]; then
  curl -fsS -o /dev/null -H "Authorization: Bearer $BOB_TOKEN" "$BASE_URL/admin/api/v1/users" || true
fi

log "10. Trigger a not-found path (error.category=not_found)"
curl -fsS -o /dev/null "$BASE_URL/this-route-does-not-exist" || true
curl -fsS -o /dev/null -u "alice:$ALICE_API_TOKEN" "$BASE_URL/private/info/does-not-exist-zzz" || true

log "11. Yank the gem we uploaded"
curl -fsS -X DELETE -u "alice:$ALICE_API_TOKEN" \
  "$BASE_URL/private/api/v1/gems/yank?gem=nokogiri&version=1.15.3&platform=arm64-darwin" || true

log "Done. Generated ~60 spans across HTTP, indexer, advisorydb, and outbound mirror calls."
