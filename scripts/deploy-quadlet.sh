#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# deploy-quadlet.sh — Build and deploy ynews via Podman Quadlet (runs on the VPS)
#
# Simplified contract (stoba1400-style, no registry, no dockernode):
#   DEPLOY_PATH  Path where the workflow has synced the source tree (required)
#
# Builds the image locally on the VPS, tags it as localhost/ynews:main-latest,
# links the quadlet unit, restarts ynews.service, and verifies the running
# container is using the freshly built image.
# =============================================================================

echo "=== deploy-quadlet starting ==="
echo "user=$(whoami) host=$(hostname) ts=$(date -u +%Y-%m-%dT%H:%M:%SZ) pid=$$"

DEPLOY_TARGET_PATH="${DEPLOY_PATH:-}"
: "${DEPLOY_TARGET_PATH:?DEPLOY_PATH is required}"
cd "${DEPLOY_TARGET_PATH}"

if ! command -v podman &>/dev/null; then
  echo "ERROR: podman not found on PATH." >&2
  exit 1
fi
echo "podman version: $(podman --version)"

podman network exists ynews-net || podman network create ynews-net

echo "Building localhost/ynews:main-latest..."
podman build --platform linux/amd64 --tag localhost/ynews:main-latest .
IMAGE_ID="$(podman image inspect localhost/ynews:main-latest --format '{{.Id}}')"
echo "Image ID: ${IMAGE_ID}"

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
