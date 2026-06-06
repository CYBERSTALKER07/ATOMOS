#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

ruby <<'RUBY'
require 'yaml'

def assert(condition, message)
  raise message unless condition
end

deployment_path = 'infra/k8s/backend-go/deployment.yaml'
service_path = 'infra/k8s/backend-go/service.yaml'
configmap_path = 'infra/k8s/backend-go/configmap.yaml'

deployment = YAML.load_file(deployment_path)
service = YAML.load_file(service_path)
configmap = YAML.load_file(configmap_path)

assert(deployment.dig('metadata', 'name') == 'backend-go', 'deployment name must be backend-go')
assert(deployment.dig('spec', 'template', 'spec', 'serviceAccountName') == 'backend-go', 'deployment must use backend-go service account')

container = deployment.dig('spec', 'template', 'spec', 'containers')&.find { |c| c['name'] == 'backend-go' }
assert(!container.nil?, 'backend-go container required')
assert(container.dig('livenessProbe', 'httpGet', 'path') == '/healthz', 'liveness must use /healthz')
assert(container.dig('readinessProbe', 'httpGet', 'path') == '/ready', 'readiness must use /ready')

assert(configmap.dig('data', 'PEGASUSX_ENV') == 'production', 'configmap must set PEGASUSX_ENV=production')
assert(service.dig('spec', 'ports', 0, 'port') == 80, 'service port must be 80')

assert(File.file?('apps/backend-go/Dockerfile'), 'apps/backend-go/Dockerfile must exist')

puts 'backend-k8s-ok'
RUBY
