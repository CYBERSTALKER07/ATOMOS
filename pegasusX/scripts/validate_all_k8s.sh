#!/usr/bin/env bash
# Universal Kubernetes manifest and overlay validation gate.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> Running PegasusX Universal Kubernetes Validator..."
cd "$ROOT_DIR/apps/backend-go"
go run ./cmd/validate-k8s
