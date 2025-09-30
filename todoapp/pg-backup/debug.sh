#!/bin/bash

PROJECT_ID="dwk-gke-473404"
CLUSTER_NAME="dwk-cluster"  # Replace with your cluster name
REGION="europe-north1-b"              # Replace with your region
GSA_NAME="pg-backup-sa"
KSA_NAME="pg-backup"
NAMESPACE="project"
BUCKET_NAME="dwk-gke-backups"  # Replace with your bucket name

# 1. Enable Workload Identity on cluster (if not already enabled)
echo "Enabling Workload Identity on cluster..."
gcloud container clusters update ${CLUSTER_NAME} \
  --region=${REGION} \
  --workload-pool=${PROJECT_ID}.svc.id.goog

# 2. Create GCP Service Account
echo "Creating GCP Service Account..."
gcloud iam service-accounts create ${GSA_NAME} \
  --display-name="PostgreSQL Backup Service Account" \
  --project=${PROJECT_ID}

# 3. Grant GCS permissions
echo "Granting GCS permissions..."
gsutil iam ch serviceAccount:${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com:objectAdmin \
  gs://${BUCKET_NAME}

# 4. Create IAM binding
echo "Creating Workload Identity binding..."
gcloud iam service-accounts add-iam-policy-binding \
  ${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${PROJECT_ID}.svc.id.goog[${NAMESPACE}/${KSA_NAME}]" \
  --project=${PROJECT_ID}

# 5. Annotate Kubernetes ServiceAccount (if already exists)
echo "Annotating Kubernetes ServiceAccount..."
kubectl annotate serviceaccount ${KSA_NAME} \
  -n ${NAMESPACE} \
  iam.gke.io/gcp-service-account=${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com \
  --overwrite

echo "Setup complete! Please restart your pods for changes to take effect."
echo "kubectl delete pod -l app=pg-backup"