#!/usr/bin/env bash
# Run k6 natively or via grafana/k6 Docker when no local k6 binary exists.
set -euo pipefail

K6_DOCKER_IMAGE="${K6_DOCKER_IMAGE:-grafana/k6:latest}"

resolve_k6_bin() {
  local candidate prefix

  if command -v k6 >/dev/null 2>&1; then
    command -v k6
    return 0
  fi

  if [[ -x /opt/homebrew/bin/brew ]]; then
    # shellcheck disable=SC1090
    eval "$(/opt/homebrew/bin/brew shellenv 2>/dev/null || true)"
  elif [[ -x /usr/local/bin/brew ]]; then
    # shellcheck disable=SC1090
    eval "$(/usr/local/bin/brew shellenv 2>/dev/null || true)"
  fi
  if command -v k6 >/dev/null 2>&1; then
    command -v k6
    return 0
  fi

  if command -v brew >/dev/null 2>&1; then
    prefix="$(brew --prefix k6 2>/dev/null || true)"
    if [[ -n "$prefix" && -x "${prefix}/bin/k6" ]]; then
      echo "${prefix}/bin/k6"
      return 0
    fi
  fi

  for candidate in /opt/homebrew/bin/k6 /usr/local/bin/k6; do
    if [[ -x "$candidate" ]]; then
      echo "$candidate"
      return 0
    fi
  done
  return 1
}

k6_base_url_for_docker() {
  local url="${1:-http://localhost:8180}"
  url="${url%/}"
  if [[ "$url" == *"127.0.0.1"* ]]; then
    echo "${url//127.0.0.1/host.docker.internal}"
  elif [[ "$url" == *"localhost"* ]]; then
    echo "${url//localhost/host.docker.internal}"
  else
    echo "$url"
  fi
}

k6_docker_available() {
  command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

ensure_k6_docker_image() {
  if docker image inspect "$K6_DOCKER_IMAGE" >/dev/null 2>&1; then
    return 0
  fi
  echo "Pulling ${K6_DOCKER_IMAGE} (one-time) ..."
  docker pull "$K6_DOCKER_IMAGE"
}

# run_k6 <runner> <scripts_dir> <artifact_dir> <summary.json> <script.js> <k6 -e args...>
# runner: absolute path to k6 binary, or the literal string "docker".
run_k6() {
  local runner=$1
  local scripts_dir=$2
  local artifact_dir=$3
  local summary_name=$4
  local script_file=$5
  shift 5

  local -a k6_env=()
  while [[ $# -gt 0 ]]; do
    if [[ "$1" == "-e" && $# -ge 2 ]]; then
      if [[ "$2" == BASE_URL=* ]]; then
        local base_val="${2#BASE_URL=}"
        if [[ "$runner" == "docker" ]]; then
          k6_env+=(-e "BASE_URL=$(k6_base_url_for_docker "$base_val")")
        else
          k6_env+=(-e "$2")
        fi
      else
        k6_env+=(-e "$2")
      fi
      shift 2
    else
      k6_env+=("$1")
      shift
    fi
  done

  if [[ "$runner" == "docker" ]]; then
    ensure_k6_docker_image
    docker run --rm \
      -v "${scripts_dir}:/scripts:ro" \
      -v "${artifact_dir}:/artifacts" \
      "$K6_DOCKER_IMAGE" \
      run \
      "${k6_env[@]}" \
      --summary-export="/artifacts/${summary_name}" \
      "/scripts/${script_file}"
    return $?
  fi

  "$runner" run \
    "${k6_env[@]}" \
    --summary-export="${artifact_dir}/${summary_name}" \
    "${scripts_dir}/${script_file}"
}
