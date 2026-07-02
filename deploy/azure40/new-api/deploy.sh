#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

if [[ -f .env ]]; then
  # shellcheck disable=SC1091
  source .env
fi

IMAGE_TAG="${1:-${NEW_API_IMAGE_TAG:-latest}}"
export NEW_API_IMAGE_TAG="${IMAGE_TAG}"

if [[ -n "${GHCR_USERNAME:-}" && -n "${GHCR_TOKEN:-}" ]]; then
  echo "${GHCR_TOKEN}" | docker login ghcr.io -u "${GHCR_USERNAME}" --password-stdin
fi

docker compose pull new-api
docker compose up -d

echo "Waiting for new-api health check..."
for _ in $(seq 1 24); do
  if curl -fsS http://127.0.0.1:3000/api/status >/dev/null; then
    echo "new-api is healthy."
    exit 0
  fi
  sleep 5
done

echo "new-api did not become healthy in time." >&2
exit 1
