#!/usr/bin/env bash
set -euo pipefail
base="${SEARCH_BASE_URL:-http://127.0.0.1:8080}"
tenant="smoke-tenant"
curl -fsS "$base/health/live" >/dev/null
curl -fsS "$base/health/ready" >/dev/null
body='{"name":"smoke-docs","shards":1,"fields":[{"name":"title","type":"text","searchable":true,"stored":true},{"name":"body","type":"text","searchable":true,"stored":true}]}'
created=$(curl -fsS -X POST "$base/api/v1/collections" -H "X-Tenant-ID: $tenant" -H 'Content-Type: application/json' -H 'Idempotency-Key: smoke-collection' -d "$body")
id=$(printf '%s' "$created" | sed -n 's/.*"ID":"\([^"]*\)".*/\1/p')
test -n "$id"
curl -fsS -X PUT "$base/api/v1/collections/$id/documents/one" -H "X-Tenant-ID: $tenant" -H 'Content-Type: application/json' -d '{"version":1,"fields":{"title":"Go","body":"中文搜索服务"}}' >/dev/null
curl -fsS -X POST "$base/api/v1/search" -H "X-Tenant-ID: $tenant" -H 'Content-Type: application/json' -d "{\"collection_id\":\"$id\",\"query\":{\"kind\":\"match\",\"field\":\"body\",\"value\":\"中文\"}}" >/dev/null
echo "smoke passed: collection=$id"
