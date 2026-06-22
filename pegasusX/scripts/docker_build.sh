#!/usr/bin/env bash
# Build pegasusX container images. Context is always the pegasusX module root.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

TARGET="${1:-all}"
TAG="${IMAGE_TAG:-local}"

build_backend() {
  docker build \
    --platform linux/amd64 \
    -f apps/backend-go/Dockerfile \
    -t "pegasusx-backend:${TAG}" \
    .
}

build_ai_worker() {
  docker build \
    --platform linux/amd64 \
    -f apps/ai-worker/Dockerfile \
    -t "pegasusx-ai-worker:${TAG}" \
    .
}

build_optimizer_core() {
  docker build \
    --platform linux/amd64 \
    -f services/optimizer-core/Dockerfile \
    -t "pegasusx-optimizer-core:${TAG}" \
    services/optimizer-core
}

case "$TARGET" in
  backend|backend-go) build_backend ;;
  ai-worker) build_ai_worker ;;
  optimizer-core) build_optimizer_core ;;
  all)
    build_backend
    build_ai_worker
    build_optimizer_core
    ;;
  *)
    echo "usage: $0 [all|backend|ai-worker|optimizer-core]" >&2
    exit 1
    ;;
esac

echo "docker-build-ok target=${TARGET} tag=${TAG}"
