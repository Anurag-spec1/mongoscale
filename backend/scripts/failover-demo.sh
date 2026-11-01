#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.replica.yml}"
API_URL="${API_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@mongoscale.dev}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Admin@12345}"

token=$(curl -fsS -X POST "$API_URL/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" | sed -E 's/.*"token":"([^"]+)".*/\1/')

show_members() {
  curl -fsS "$API_URL/api/v1/system/mongodb-status" -H "Authorization: Bearer $token" \
    | tr '{' '\n' | grep -E '"state":"(PRIMARY|SECONDARY)"' | sed -E 's/.*"name":"([^"]+)".*"state":"([^"]+)".*/\1 \2/'
}

echo "Replica set before failover:"
show_members

primary=$(docker compose -f "$COMPOSE_FILE" exec -T mongo1 mongosh --quiet --eval 'db.hello().primary' | tr -d '\r' | cut -d: -f1)
echo "Stopping primary container: $primary"
docker compose -f "$COMPOSE_FILE" stop "$primary"

echo "Waiting for election..."
sleep 12
echo "Replica set after failover:"
show_members

echo "Restarting $primary"
docker compose -f "$COMPOSE_FILE" start "$primary"
