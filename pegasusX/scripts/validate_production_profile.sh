#!/usr/bin/env bash
# Assert Kubernetes prod manifests match the production env contract.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

ruby <<'RUBY'
require 'yaml'

def assert(condition, message)
  raise message unless condition
end

configmap = YAML.load_file('infra/k8s/backend-go/configmap.yaml')
data = configmap['data'] || {}

assert(data['PEGASUSX_ENV'] == 'production', 'PEGASUSX_ENV must be production')
assert(data['REQUIRE_INFRA_ADAPTERS'] == 'true', 'REQUIRE_INFRA_ADAPTERS must be true')
assert(data['PEGASUSX_RUN_MODE'] == 'api', 'API ConfigMap must set PEGASUSX_RUN_MODE=api')
assert(data['GLOBAL_PAY_ENV'] == 'production', 'GLOBAL_PAY_ENV must be production')

# Pilot default: single main topic; domain migration is staging-only until cutover.
assert(data['KAFKA_TOPIC_DUAL_WRITE'].to_s != 'true', 'prod ConfigMap must not enable KAFKA_TOPIC_DUAL_WRITE')
assert(data['KAFKA_TOPIC_CONSUME_DOMAIN'].to_s != 'true', 'prod ConfigMap must not enable KAFKA_TOPIC_CONSUME_DOMAIN')

worker = YAML.load_file('infra/k8s/backend-go-worker/deployment.yaml')
worker_container = worker.dig('spec', 'template', 'spec', 'containers')&.find { |c| c['name'] == 'backend-go-worker' }
worker_env = (worker_container['env'] || []).each_with_object({}) { |e, h| h[e['name']] = e['value'] }
assert(worker_env['PEGASUSX_RUN_MODE'] == 'worker', 'worker Deployment must set PEGASUSX_RUN_MODE=worker')

api_deploy = YAML.load_file('infra/k8s/backend-go/deployment.yaml')
replicas = api_deploy.dig('spec', 'replicas')
assert(replicas.to_i >= 2, 'API Deployment should run at least 2 replicas for prod')

pilot_overlay = File.read('infra/k8s/overlays/pilot/kustomization.yaml')
assert(!pilot_overlay.include?('KAFKA_TOPIC_DUAL_WRITE=true'), 'pilot overlay must not enable KAFKA_TOPIC_DUAL_WRITE')
assert(pilot_overlay.include?('KAFKA_TOPIC_DUAL_WRITE=false'), 'pilot overlay must set KAFKA_TOPIC_DUAL_WRITE=false')
assert(pilot_overlay.include?('KAFKA_TOPIC_CONSUME_DOMAIN=false'), 'pilot overlay must set KAFKA_TOPIC_CONSUME_DOMAIN=false')

puts 'production-profile-ok'
RUBY
