#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# deploy-quadlet.sh — Deploy ynews via Podman Quadlet (runs on the VPS)
#
# Single-service trim of the hoopnerd deploy-quadlet.sh contract:
#   DEPLOY_PATH            Path to cloned repo on VPS  (required)
#   IMAGE_TAG              Git SHA / tag for image     (required)
#   REGISTRY_URL           Container registry URL      (required)
#   REGISTRY_USERNAME      Registry login user         (required)
#   REGISTRY_PASSWORD_B64  Base64-encoded password     (required)
#   REGISTRY_NAMESPACE     Registry project namespace  (required)
#   TARGET_SHA             Git ref/SHA to sync to      (default origin/main)
# =============================================================================

echo "=== deploy-quadlet starting ==="
echo "user=$(whoami) host=$(hostname) ts=$(date -u +%Y-%m-%dT%H:%M:%SZ) pid=$$"
echo "image_tag=${IMAGE_TAG:-UNSET}"

DEPLOY_TARGET_PATH="${DEPLOY_PATH:-}"
: "${DEPLOY_TARGET_PATH:?DEPLOY_PATH is required}"
: "${IMAGE_TAG:?IMAGE_TAG is required}"
: "${REGISTRY_URL:?REGISTRY_URL is required}"
: "${REGISTRY_USERNAME:?REGISTRY_USERNAME is required}"
: "${REGISTRY_PASSWORD_B64:?REGISTRY_PASSWORD_B64 is required}"
: "${REGISTRY_NAMESPACE:?REGISTRY_NAMESPACE is required}"

# First-time setup: clone if the path does not exist yet.
if [ ! -d "${DEPLOY_TARGET_PATH}/.git" ]; then
  mkdir -p "$(dirname "${DEPLOY_TARGET_PATH}")"
  git clone "${REPO_URL:?REPO_URL is required}" "${DEPLOY_TARGET_PATH}"
fi

cd "${DEPLOY_TARGET_PATH}"

SYNC_REF="${TARGET_SHA:-origin/main}"
git fetch --all --prune --tags --force
git clean -fd -- quadlet scripts/deploy-quadlet.sh scripts/version.sh
git checkout --detach "${SYNC_REF}"
echo "Deploying commit: $(git rev-parse HEAD)"

if ! command -v podman &>/dev/null; then
  echo "ERROR: podman not found on PATH." >&2
  exit 1
fi
echo "podman version: $(podman --version)"

podman network exists hoopnerd-edge || podman network create hoopnerd-edge

# --- registry login -----------------------------------------------------------
REGISTRY_HOST="${REGISTRY_URL#https://}"
REGISTRY_HOST="${REGISTRY_HOST#http://}"
REGISTRY_HOST="${REGISTRY_HOST%%/*}"
REGISTRY_HOST="${REGISTRY_HOST%/}"

REGISTRY_PASSWORD="$(printf '%s' "${REGISTRY_PASSWORD_B64}" | base64 -d)"
echo "Logging into ${REGISTRY_HOST}..."
if ! printf '%s' "${REGISTRY_PASSWORD}" | podman login "${REGISTRY_HOST}" -u "${REGISTRY_USERNAME}" --password-stdin; then
  echo "ERROR: Registry login failed" >&2
  exit 1
fi

# --- pull + tag ----------------------------------------------------------------
IMAGE_SHA="${REGISTRY_HOST}/${REGISTRY_NAMESPACE}/ynews:${IMAGE_TAG}"
IMAGE_LATEST="localhost/ynews:main-latest"

echo "Pulling ${IMAGE_SHA}..."
podman pull "${IMAGE_SHA}"
podman tag "${IMAGE_SHA}" "${IMAGE_LATEST}"
IMAGE_ID="$(podman image inspect "${IMAGE_SHA}" --format '{{.Id}}')"
echo "Image ID: ${IMAGE_ID}"

# --- deploy quadlet -------------------------------------------------------------
QUADLET_DIR="${HOME}/.config/containers/systemd"
mkdir -p "${QUADLET_DIR}"

target="${QUADLET_DIR}/ynews.container"
src="${DEPLOY_TARGET_PATH}/quadlet/ynews.container"
if [ ! -L "${target}" ] || [ "$(readlink "${target}")" != "${src}" ]; then
  rm -f "${target}"
  ln -s "${src}" "${target}"
  echo "Linked ynews.container"
fi

systemctl --user daemon-reload

echo "Restarting ynews.service..."
systemctl --user restart ynews.service

sleep 3
if ! systemctl --user is-active --quiet ynews.service; then
  echo "ERROR: ynews.service not active after restart" >&2
  journalctl --user -u ynews.service --no-pager -n 30 >&2 || true
  exit 1
fi

running_image="$(podman container inspect ynews --format '{{.Image}}')"
if [ "${running_image}" != "${IMAGE_ID}" ]; then
  echo "ERROR: ynews running image ${running_image}, expected ${IMAGE_ID}" >&2
  exit 1
fi

echo ""
echo "=== deploy-quadlet complete ==="
podman ps --filter name=ynews
