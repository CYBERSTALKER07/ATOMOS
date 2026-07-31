# Launch Readiness Runbook

Release ownership: platform engineering + on-call SRE.

Preflight:
- make test-ssmr-infra
- make validate-ai-worker-k8s
- make validate-launch-readiness
- make p0-preflight

rollback: revert deployment image tag and re-run cloud-smoke-ssmr.

launch support: follow hypercare checklist in docs/P0_LAUNCH_CHECKLIST.md.

