# GCP Support case — SSD + IP quota (copy/paste)

## Links (open these)

- Support home: https://console.cloud.google.com/support?project=pegasus-503013  
- Cases: https://console.cloud.google.com/support/cases?project=pegasus-503013  
- Create case: https://console.cloud.google.com/support/new?project=pegasus-503013  
- Quotas UI: https://console.cloud.google.com/iam-admin/quotas?project=pegasus-503013  

## Case fields (recommended)

| Field | Value |
|--------|--------|
| **Project** | `pegasus-503013` |
| **Category** | Technical / Compute Engine / Quotas & limits |
| **Priority** | P3 (or P2 if staging is blocked) |
| **Subject** | Request SSD-TOTAL-GB 500 and IN-USE-ADDRESSES 8 in asia-south1 after paid upgrade |

## Subject

```
Quota increase: SSD-TOTAL-GB 500 + IN-USE-ADDRESSES 8 (asia-south1) — GKE Autopilot staging blocked
```

## Description (paste body)

```
Hello Google Cloud Support,

We upgraded from Free Trial to a full paid billing account. Remaining free-trial credits were retained (expected). We need quota increases to run GKE Autopilot staging.

Project ID: pegasus-503013
Project number: 1002695564567
Organization ID: 667485536839
Billing account: 01BFC8-0FA416-0BBA18 (My Billing Account, open, paid/upgraded)
Region: asia-south1
Primary contact: blackfoxenterprise3697@gmail.com

=== Requested quotas ===
1) Metric: Persistent Disk SSD (GB) / SSD-TOTAL-GB-per-project-region
   Region: asia-south1
   Current limit: 250 GB
   Current usage: ~200 GB (2 × ~100 GB Autopilot node boot disks)
   Requested limit: 500 GB (minimum; 1000 GB preferred if possible)

2) Metric: In-use IP addresses / IN-USE-ADDRESSES-per-project-region
   Region: asia-south1
   Current limit: 4
   Current usage: 2
   Requested limit: 8

=== Business / technical justification ===
We run GKE Autopilot cluster "pegasusx-staging-gke" for a production-bound multi-service backend (Spanner, Memorystore Redis, Managed Kafka already provisioned).

Autopilot nodes use ~100 GB boot disks each. With SSD limit 250 GB we can only fit 2 nodes. Those nodes are filled by system/DaemonSet pods (~94% CPU), so application Deployments stay Pending. Scale-up fails with "GCE quota exceeded".

We need at least one additional node (preferably capacity for API + worker) → SSD ≥ 500 GB and a few more in-use addresses for nodes.

=== Quota preferences already filed (Cloud Quotas API) ===
- SSD preference ID: 1f50635b-6017-4887-af2e-fa1eeb02dff8
  preferredValue=500, grantedValue=250
  state: "We cannot grant the preferred quota '500' ... at this moment."

- IP preference ID: cd556684-055e-41b8-8576-b8041448f24e
  preferredValue=8, grantedValue=4
  state: "We cannot grant the preferred quota '8' ... at this moment."

Console self-serve previously said we are not eligible for increase above 250 based on usage history; we have since upgraded to a paid account and request manual review/approval.

Please grant the increases above so staging pods can schedule.

Thank you.
```

## If you only have Billing Support (no technical plan)

1. Still open Support → choose **Billing** if Technical is greyed out.  
2. Or: https://cloud.google.com/support/docs/billing-support  
3. Ask them to **escalate a quota increase** for Compute Engine SSD in asia-south1, or point you to how to buy Standard Support.  
4. Optional: Standard Support (~$29/month) unlocks technical cases for quotas.

## After they grant

```bash
export PATH="/opt/homebrew/bin:/opt/homebrew/share/google-cloud-sdk/bin:$PATH"
export USE_GKE_GCLOUD_AUTH_PLUGIN=True
gcloud compute regions describe asia-south1 --project=pegasus-503013 \
  --format='json' | python3 -c 'import json,sys;d=json.load(sys.stdin);
[print(q["metric"],q["usage"],q["limit"]) for q in d["quotas"] if q["metric"] in ("SSD_TOTAL_GB","IN_USE_ADDRESSES")]'
kubectl scale deploy/backend-go deploy/backend-go-worker -n pegasusx-staging --replicas=1
kubectl get pods -n pegasusx-staging -w
```
