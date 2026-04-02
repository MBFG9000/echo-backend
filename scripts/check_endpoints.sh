#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
COMPOSE_CMD="${COMPOSE_CMD:-docker compose}"

RESP_BODY="/tmp/echo_api_resp.json"
RESP_WS="/tmp/echo_api_ws.txt"

req() {
  local name="$1"
  local method="$2"
  local url="$3"
  local expected="$4"
  local data="${5:-}"
  local auth="${6:-}"

  local curl_args=(
    -sS
    --max-time 10
    -o "$RESP_BODY"
    -w "%{http_code}"
    -X "$method"
    "$url"
  )

  if [[ -n "$auth" ]]; then
    curl_args+=( -H "Authorization: Bearer $auth" )
  fi

  if [[ -n "$data" ]]; then
    curl_args+=( -H "Content-Type: application/json" -d "$data" )
  fi

  local code
  code="$(curl "${curl_args[@]}")"

  if [[ "$code" != "$expected" ]]; then
    echo "FAIL: $name -> expected $expected, got $code"
    echo "Body: $(tr -d '\n' <"$RESP_BODY" | head -c 400)"
    exit 1
  fi

  echo "OK: $name -> $code"
}

req_any() {
  local name="$1"
  local method="$2"
  local url="$3"
  local expected_csv="$4"
  local data="${5:-}"
  local auth="${6:-}"

  local curl_args=(
    -sS
    --max-time 10
    -o "$RESP_BODY"
    -w "%{http_code}"
    -X "$method"
    "$url"
  )

  if [[ -n "$auth" ]]; then
    curl_args+=( -H "Authorization: Bearer $auth" )
  fi

  if [[ -n "$data" ]]; then
    curl_args+=( -H "Content-Type: application/json" -d "$data" )
  fi

  local code
  code="$(curl "${curl_args[@]}")"

  local ok=1
  IFS=',' read -r -a codes <<<"$expected_csv"
  for c in "${codes[@]}"; do
    if [[ "$code" == "$c" ]]; then
      ok=0
      break
    fi
  done

  if [[ $ok -ne 0 ]]; then
    echo "FAIL: $name -> expected one of [$expected_csv], got $code"
    echo "Body: $(tr -d '\n' <"$RESP_BODY" | head -c 400)"
    exit 1
  fi

  echo "OK: $name -> $code"
}

json_get() {
  local path="$1"
  python3 - "$RESP_BODY" "$path" <<'PY'
import json, sys
file_path, path = sys.argv[1], sys.argv[2]
with open(file_path, "r", encoding="utf-8") as f:
    obj = json.load(f)
cur = obj
for part in path.split('.'):
    if not part:
        continue
    if part.endswith(']') and '[' in part:
        name, idx = part[:-1].split('[', 1)
        if name:
            cur = cur.get(name, {})
        cur = cur[int(idx)]
    else:
        cur = cur.get(part, "") if isinstance(cur, dict) else ""
print(cur if cur is not None else "")
PY
}

token_claim() {
  local token="$1"
  local key="$2"
  python3 - "$token" "$key" <<'PY'
import base64, json, sys
raw, key = sys.argv[1], sys.argv[2]
parts = raw.split('.')
if len(parts) < 2:
    print("")
    raise SystemExit(0)
payload = parts[1] + '=' * (-len(parts[1]) % 4)
obj = json.loads(base64.urlsafe_b64decode(payload.encode()))
print(obj.get(key, ""))
PY
}

probe_ws_upgrade() {
  python3 - "$BASE_URL" <<'PY'
import socket
import sys
from urllib.parse import urlparse

base = sys.argv[1]
u = urlparse(base)
host = u.hostname or "localhost"
port = u.port or (443 if u.scheme == "https" else 80)
path = "/ws/feed"

req = (
  f"GET {path} HTTP/1.1\r\n"
  f"Host: {host}:{port}\r\n"
  "Connection: Upgrade\r\n"
  "Upgrade: websocket\r\n"
  "Sec-WebSocket-Version: 13\r\n"
  "Sec-WebSocket-Key: SGVsbG9Xb3JsZDEyMzQ1Ng==\r\n"
  "\r\n"
)

try:
  sock = socket.create_connection((host, port), timeout=4)
  sock.settimeout(4)
  sock.sendall(req.encode())

  data = b""
  while b"\r\n\r\n" not in data:
    chunk = sock.recv(1024)
    if not chunk:
      break
    data += chunk
  sock.close()

  head = data.split(b"\r\n", 1)[0].decode(errors="replace")
  if " 101 " in head or head.endswith(" 101"):
    print("OK: websocket upgrade -> 101")
    raise SystemExit(0)

  print(f"FAIL: websocket probe -> unexpected status line: {head}")
  raise SystemExit(1)
except Exception as e:
  print(f"FAIL: websocket probe -> {e}")
  raise SystemExit(1)
PY
}

require_non_empty() {
  local label="$1"
  local value="$2"
  if [[ -z "$value" ]]; then
    echo "FAIL: empty value for $label"
    exit 1
  fi
}

echo "BASE_URL=$BASE_URL"

req "health" GET "$BASE_URL/health" 200

ROUTE_PROBE="$(curl -sS --max-time 10 -o "$RESP_BODY" -w "%{http_code}" -X POST "$BASE_URL/posts/get" -H 'Content-Type: application/json' -d '{}')"
OBJECT_MODE=0
if [[ "$ROUTE_PROBE" == "400" ]]; then
  OBJECT_MODE=1
fi

if [[ "$OBJECT_MODE" == "1" ]]; then
  echo "Route mode: object"
else
  echo "Route mode: legacy"
fi

req "auth register" POST "$BASE_URL/auth/register" 201
TOKEN="$(json_get token)"
require_non_empty "token" "$TOKEN"

req "posts create" POST "$BASE_URL/posts" 201 '{"content":"curl full check"}' "$TOKEN"
POST_ID="$(json_get id)"
require_non_empty "post id" "$POST_ID"

if [[ "$OBJECT_MODE" == "1" ]]; then
  req "posts get" POST "$BASE_URL/posts/get" 200 '{"id":"'"$POST_ID"'"}'
  req "replies create" POST "$BASE_URL/posts/replies/create" 201 '{"postId":"'"$POST_ID"'","content":"reply"}' "$TOKEN"
  req "replies list" POST "$BASE_URL/posts/replies/list" 200 '{"postId":"'"$POST_ID"'","limit":10}'
  req "react" POST "$BASE_URL/posts/react" 200 '{"postId":"'"$POST_ID"'","kind":"upvote"}' "$TOKEN"
  req "report" POST "$BASE_URL/posts/report" 201 '{"postId":"'"$POST_ID"'","reason":"moderation check"}' "$TOKEN"
else
  req "posts get" GET "$BASE_URL/posts/$POST_ID" 200
  req "replies create" POST "$BASE_URL/posts/$POST_ID/replies" 201 '{"content":"reply"}' "$TOKEN"
  req "replies list" GET "$BASE_URL/posts/$POST_ID/replies?limit=10" 200
  req "react" POST "$BASE_URL/posts/$POST_ID/react" 200 '{"kind":"upvote"}' "$TOKEN"
  req "report" POST "$BASE_URL/posts/$POST_ID/report" 201 '{"reason":"moderation check"}' "$TOKEN"
fi

if [[ "$OBJECT_MODE" == "1" ]]; then
  req "feed latest" POST "$BASE_URL/feed/latest" 200 '{"limit":5}'
  req "feed trending" POST "$BASE_URL/feed/trending" 200 '{"limit":5}'
else
  req "feed latest" GET "$BASE_URL/feed/latest?limit=5" 200
  req "feed trending" GET "$BASE_URL/feed/trending?limit=5" 200
fi

REFRESH_CODE="$(curl -sS --max-time 10 -o "$RESP_BODY" -w "%{http_code}" -X POST "$BASE_URL/auth/refresh" -H 'Content-Type: application/json' -d '{"token":"'"$TOKEN"'"}')"
if [[ "$REFRESH_CODE" == "200" ]]; then
  echo "OK: auth refresh -> 200 (json body)"
else
  REFRESH_CODE="$(curl -sS --max-time 10 -o "$RESP_BODY" -w "%{http_code}" -X POST "$BASE_URL/auth/refresh" -H "Authorization: Bearer $TOKEN")"
  if [[ "$REFRESH_CODE" != "200" ]]; then
    echo "FAIL: auth refresh -> expected 200, got $REFRESH_CODE"
    echo "Body: $(tr -d '\n' <"$RESP_BODY" | head -c 400)"
    exit 1
  fi
  echo "OK: auth refresh -> 200 (bearer header)"
fi
TOKEN2="$(json_get token)"
require_non_empty "token2" "$TOKEN2"

if [[ "$OBJECT_MODE" == "1" ]]; then
  req "admin reports non-admin" POST "$BASE_URL/admin/reports/list" 403 '{"limit":10,"offset":0}' "$TOKEN2"
else
  req "admin reports non-admin" GET "$BASE_URL/admin/reports?limit=10&offset=0" 403 '' "$TOKEN2"
fi

USER_ID="$(token_claim "$TOKEN2" user_id)"
require_non_empty "user_id" "$USER_ID"

$COMPOSE_CMD exec -T postgres psql -U echo -d echo -c "UPDATE users SET is_admin = true WHERE id = '$USER_ID';" >/dev/null

REFRESH_ADMIN_CODE="$(curl -sS --max-time 10 -o "$RESP_BODY" -w "%{http_code}" -X POST "$BASE_URL/auth/refresh" -H 'Content-Type: application/json' -d '{"token":"'"$TOKEN2"'"}')"
if [[ "$REFRESH_ADMIN_CODE" == "200" ]]; then
  echo "OK: auth refresh admin -> 200 (json body)"
else
  REFRESH_ADMIN_CODE="$(curl -sS --max-time 10 -o "$RESP_BODY" -w "%{http_code}" -X POST "$BASE_URL/auth/refresh" -H "Authorization: Bearer $TOKEN2")"
  if [[ "$REFRESH_ADMIN_CODE" != "200" ]]; then
    echo "FAIL: auth refresh admin -> expected 200, got $REFRESH_ADMIN_CODE"
    echo "Body: $(tr -d '\n' <"$RESP_BODY" | head -c 400)"
    exit 1
  fi
  echo "OK: auth refresh admin -> 200 (bearer header)"
fi
ADMIN_TOKEN="$(json_get token)"
require_non_empty "admin token" "$ADMIN_TOKEN"

if [[ "$OBJECT_MODE" == "1" ]]; then
  req "admin reports list" POST "$BASE_URL/admin/reports/list" 200 '{"limit":10,"offset":0}' "$ADMIN_TOKEN"
else
  req "admin reports list" GET "$BASE_URL/admin/reports?limit=10&offset=0" 200 '' "$ADMIN_TOKEN"
fi
REPORT_ID="$(json_get reports[0].id)"
require_non_empty "report id" "$REPORT_ID"

if [[ "$OBJECT_MODE" == "1" ]]; then
  req "admin reports action" POST "$BASE_URL/admin/reports/action" 200 '{"reportId":"'"$REPORT_ID"'","action":"dismiss","note":"checked by script"}' "$ADMIN_TOKEN"
  req "posts delete" POST "$BASE_URL/posts/delete" 200 '{"id":"'"$POST_ID"'"}' "$ADMIN_TOKEN"
else
  req "admin reports action" POST "$BASE_URL/admin/reports/$REPORT_ID/action" 200 '{"action":"dismiss","note":"checked by script"}' "$ADMIN_TOKEN"
  req "posts delete" DELETE "$BASE_URL/posts/$POST_ID" 200 '' "$ADMIN_TOKEN"
fi

probe_ws_upgrade

echo "ALL ENDPOINTS PASSED"
