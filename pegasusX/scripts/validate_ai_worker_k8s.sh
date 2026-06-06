#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

cd "$ROOT_DIR"

ruby <<'RUBY'
require 'yaml'

def assert(condition, message)
  raise message unless condition
end

configmap_path = 'infra/k8s/ai-worker/configmap.yaml'
deployment_path = 'infra/k8s/ai-worker/deployment.yaml'
service_path = 'infra/k8s/ai-worker/service.yaml'

configmap = YAML.load_file(configmap_path)
deployment = YAML.load_file(deployment_path)
service = YAML.load_file(service_path)

assert(configmap['kind'] == 'ConfigMap', "#{configmap_path} must be a ConfigMap")
assert(configmap.dig('metadata', 'name') == 'ai-worker-config', 'config map name must be ai-worker-config')
assert(configmap.dig('data', 'HEALTH_PORT') == '8081', 'config map HEALTH_PORT must be 8081')

assert(deployment['kind'] == 'Deployment', "#{deployment_path} must be a Deployment")
assert(deployment.dig('metadata', 'name') == 'ai-worker', 'deployment name must be ai-worker')

container = deployment.dig('spec', 'template', 'spec', 'containers')&.find { |entry| entry['name'] == 'ai-worker' }
assert(!container.nil?, 'deployment must define an ai-worker container')
assert(container['image'].to_s.include?('PEGASUSX_AI_WORKER_IMAGE_PLACEHOLDER'), 'deployment image placeholder must be present until environment wiring replaces it')
assert(container.dig('livenessProbe', 'httpGet', 'path') == '/healthz', 'deployment liveness probe must target /healthz')
assert(container.dig('readinessProbe', 'httpGet', 'path') == '/ready', 'deployment readiness probe must target /ready')
assert(container.dig('ports', 0, 'containerPort') == 8081, 'deployment container port must be 8081')

env_from = container['envFrom'] || []
config_ref = env_from.find { |entry| entry.dig('configMapRef', 'name') == 'ai-worker-config' }
assert(!config_ref.nil?, 'deployment must source env from ai-worker-config')

assert(service['kind'] == 'Service', "#{service_path} must be a Service")
assert(service.dig('metadata', 'name') == 'ai-worker', 'service name must be ai-worker')
assert(service.dig('spec', 'ports', 0, 'port') == 8081, 'service port must be 8081')
assert(service.dig('spec', 'ports', 0, 'targetPort') == 'http', 'service targetPort must be http')

puts 'ai-worker-k8s-ok'
RUBY