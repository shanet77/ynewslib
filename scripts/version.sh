#!/usr/bin/env bash
# Generates a semver version string from Git metadata.
# Same scheme as the hoopnerd repo: latest tag major.minor + commit count as patch.
set -euo pipefail

LATEST_TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")"
LATEST_TAG="${LATEST_TAG#v}"

COMMIT_COUNT="$(git rev-list --count HEAD 2>/dev/null || echo "0")"

MAJOR="$(echo "${LATEST_TAG}" | cut -d. -f1)"
MINOR="$(echo "${LATEST_TAG}" | cut -d. -f2)"

echo "${MAJOR}.${MINOR}.${COMMIT_COUNT}"
