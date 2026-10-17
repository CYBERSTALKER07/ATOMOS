#!/bin/bash
set -e

echo "Waiting for pegasusx-staging-gke to be deleted..."
while gcloud container clusters list --region=asia-south1 --project=pegasusx-prod-uz | grep -q pegasusx-staging-gke; do
  echo "Still deleting..."
  sleep 10
done
echo "Deleted."

echo "Creating new standard cluster..."
gcloud container clusters create pegasusx-staging-gke \
  --zone=asia-south1-a \
  --project=pegasusx-prod-uz \
  --machine-type=e2-medium \
  --num-nodes=2 \
  --disk-type=pd-standard \
  --disk-size=50 \
  --workload-pool=pegasusx-prod-uz.svc.id.goog \
  --enable-autoscaling \
  --min-nodes=1 \
  --max-nodes=3 \
  --quiet

echo "Cluster creation completed."
