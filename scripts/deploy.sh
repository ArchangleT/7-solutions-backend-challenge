#!/usr/bin/env bash
# Blue-Green deployment for user-management-api (docker-compose, single Docker host).
#
# Usage:
#   APP_ENV=prod IMAGE_NAME=ghcr.io/<owner>/user-management-api IMAGE_TAG=<git-sha> JWT_SECRET=... ./scripts/deploy.sh       # deploy
#   APP_ENV=prod IMAGE_NAME=ghcr.io/<owner>/user-management-api IMAGE_TAG=<previous-sha> JWT_SECRET=... ./scripts/deploy.sh rollback
#
# Strategy (D5-3 Blue-Green):
#   1. deploy_standby    — start standby instance (api-green) with the NEW image
#   2. healthcheck+smoke — verify readiness BEFORE any traffic moves
#   3. switch_traffic    — recreate active service (api) with the new image
#   4. decommission      — take the standby down after the switch
#   5. rollback          — switch active back to a previous IMAGE_TAG
#
# Run against the target Docker host (self-hosted runner on the host, or set
# DOCKER_HOST / DOCKER_CONTEXT for a remote engine). No secrets are committed;
# JWT_SECRET / MONGO_URI must come from the environment or CI secrets.
set -euo pipefail

APP_ENV=${APP_ENV:-prod}
IMAGE_NAME=${IMAGE_NAME:?IMAGE_NAME required (e.g. ghcr.io/<owner>/user-management-api)}
IMAGE_TAG=${IMAGE_TAG:?IMAGE_TAG required (git sha of the build to deploy)}
JWT_SECRET=${JWT_SECRET:?JWT_SECRET required}

ACTIVE_SERVICE="api"
STANDBY_SERVICE="api-green"

export IMAGE_NAME IMAGE_TAG APP_ENV JWT_SECRET

log() { printf '[deploy] %s\n' "$*"; }

compose() { docker compose -f docker-compose.yml "$@"; }

healthcheck() {
  local service="$1" attempt=1
  until compose exec -T "$service" wget --spider -q http://localhost:8080/health 2>/dev/null; do
    if [ "$attempt" -ge 30 ]; then
      log "health check FAILED for $service"
      return 1
    fi
    log "waiting for $service readiness (attempt $attempt/30)"
    attempt=$((attempt + 1))
    sleep 2
  done
  log "health OK: $service"
}

deploy_standby() {
  log "deploying standby $STANDBY_SERVICE with ${IMAGE_NAME}:${IMAGE_TAG}"
  compose --profile deploy up -d --no-deps --force-recreate "$STANDBY_SERVICE"
  healthcheck "$STANDBY_SERVICE"
  log "smoke test: GET /health -> $(compose exec -T "$STANDBY_SERVICE" wget --spider -q http://localhost:8080/health 2>&1 && echo OK)"
}

switch_traffic() {
  log "switching traffic to ${IMAGE_NAME}:${IMAGE_TAG}"
  compose up -d --no-deps --force-recreate "$ACTIVE_SERVICE"
  healthcheck "$ACTIVE_SERVICE"
}

decommission_standby() {
  log "decommissioning $STANDBY_SERVICE"
  compose --profile deploy down "$STANDBY_SERVICE"
}

rollback() {
  log "ROLLBACK: restoring ${IMAGE_NAME}:${IMAGE_TAG} as active"
  compose up -d --no-deps --force-recreate "$ACTIVE_SERVICE"
  healthcheck "$ACTIVE_SERVICE"
}

case "${1:-deploy}" in
  deploy)
    deploy_standby
    switch_traffic
    decommission_standby
    log "deployment complete (${IMAGE_TAG})"
    ;;
  rollback)
    rollback
    ;;
  *)
    echo "usage: $0 [deploy|rollback]" >&2
    exit 2
    ;;
esac