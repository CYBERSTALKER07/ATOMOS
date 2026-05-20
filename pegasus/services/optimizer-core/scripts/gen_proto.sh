#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

python3 -m grpc_tools.protoc \
  -I ./proto \
  --python_out=./server \
  --grpc_python_out=./server \
  ./proto/optimizer_core.proto

echo "Generated Python protobuf and gRPC stubs under server/."
