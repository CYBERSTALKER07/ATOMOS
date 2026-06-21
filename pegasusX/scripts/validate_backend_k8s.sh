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
worker_deployment_path = 'infra/k8s/backend-go-worker/deployment.yaml'
worker_service_path = 'infra/k8s/backend-go-worker/service.yaml'
pdb_path = 'infra/k8s/backend-go/pdb.yaml'
hpa_path = 'infra/k8s/backend-go/hpa.yaml'

deployment = YAML.load_file(deployment_path)
service = YAML.load_file(service_path)
configmap = YAML.load_file(configmap_path)
worker_deployment = YAML.load_file(worker_deployment_path)
worker_service = YAML.load_file(worker_service_path)
pdb = YAML.load_file(pdb_path)
hpa = YAML.load_file(hpa_path)

assert(deployment.dig('metadata', 'name') == 'backend-go', 'deployment name must be backend-go')
assert(deployment.dig('spec', 'template', 'spec', 'serviceAccountName') == 'backend-go', 'deployment must use backend-go service account')

container = deployment.dig('spec', 'template', 'spec', 'containers')&.find { |c| c['name'] == 'backend-go' }
assert(!container.nil?, 'backend-go container required')
assert(container.dig('livenessProbe', 'httpGet', 'path') == '/healthz', 'liveness must use /healthz')
assert(container.dig('readinessProbe', 'httpGet', 'path') == '/ready', 'readiness must use /ready')

assert(configmap.dig('data', 'PEGASUSX_ENV') == 'production', 'configmap must set PEGASUSX_ENV=production')
assert(configmap.dig('data', 'HTTP_PORT') == '8080', 'configmap must set HTTP_PORT=8080 (backend-go reads HTTP_PORT, not PORT)')
assert(configmap.dig('data', 'PEGASUSX_RUN_MODE') == 'api', 'configmap must set PEGASUSX_RUN_MODE=api for API pods')
assert(configmap.dig('data', 'ROUTING_OSRM_URL'), 'configmap must set ROUTING_OSRM_URL for route geometry')
assert(configmap.dig('data', 'GLOBAL_PAY_ENV'), 'configmap must set GLOBAL_PAY_ENV')
assert(service.dig('spec', 'ports', 0, 'port') == 80, 'service port must be 80')

assert(pdb.dig('spec', 'selector', 'matchLabels', 'app') == 'backend-go', 'PDB must target backend-go')
assert(hpa.dig('spec', 'scaleTargetRef', 'name') == 'backend-go', 'HPA must target backend-go deployment')

assert(worker_deployment.dig('metadata', 'name') == 'backend-go-worker', 'worker deployment name must be backend-go-worker')
worker_container = worker_deployment.dig('spec', 'template', 'spec', 'containers')&.find { |c| c['name'] == 'backend-go-worker' }
assert(!worker_container.nil?, 'backend-go-worker container required')
worker_env = worker_container['env'] || []
run_mode = worker_env.find { |e| e['name'] == 'PEGASUSX_RUN_MODE' }
assert(run_mode && run_mode['value'] == 'worker', 'worker deployment must set PEGASUSX_RUN_MODE=worker')
assert(worker_container.dig('livenessProbe', 'httpGet', 'path') == '/healthz', 'worker liveness must use /healthz')
assert(worker_container.dig('readinessProbe', 'httpGet', 'path') == '/ready', 'worker readiness must use /ready')
assert(worker_service.dig('spec', 'ports', 0, 'port') == 8081, 'worker service port must be 8081')

assert(File.file?('apps/backend-go/Dockerfile'), 'apps/backend-go/Dockerfile must exist')

puts 'backend-k8s-ok'
RUBY
