# Step 6 — External Secrets (SSMR) — DONE

**Cluster:** `pegasusx-ssmr-gke` (asia-south1)  
**Namespace:** `pegasusx-ssmr`  
**Date:** 2026-07-26

## Result
- External Secrets Operator installed (Helm chart, CRDs v1)
- SecretStore `pegasusx-gsm`: **Valid / Ready**
- ExternalSecret `backend-go-secrets`: **SecretSynced / Ready=True**
- K8s Secret `backend-go-secrets` materialised from GSM

## GSM mapping (all had versions already)
- jwt-secret ← pegasusx-ssmr-jwt-secret
- internal-api-key ← pegasusx-ssmr-internal-api-key
- global-pay-webhook-secret / service-id / username / password
- adyen / stripe / payme / click webhook secrets
- google-maps-api-key
- redis-auth, redis-addr
- kafka-bootstrap-servers

## Manifest
`infra/k8s/overlays/ssmr/backend-go-externalsecret.yaml`  
(API `external-secrets.io/v1`, WI with clusterLocation/Name/ProjectID)

## WI
- KSA `backend-go` → GSA `ssmr-backend@pegasus-503013.iam.gserviceaccount.com`
- member: `serviceAccount:pegasus-503013.svc.id.goog[pegasusx-ssmr/backend-go]`

## Next
Step 8–9: images + apply Deployments for API/worker.
