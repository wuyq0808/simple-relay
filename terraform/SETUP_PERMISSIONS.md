# Required Service Account Permissions

The service account used by GitHub Actions (stored in `GCP_SA_KEY` secret) needs the following IAM roles:

## Required Roles

```bash
# Grant all required roles to your service account
PROJECT_ID="your-project-id"
SERVICE_ACCOUNT_EMAIL="your-service-account@${PROJECT_ID}.iam.gserviceaccount.com"

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/artifactregistry.repoAdmin"

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/datastore.owner"

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/compute.networkAdmin"

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/servicenetworking.networksAdmin"
```

## Roles Explanation

- **`roles/artifactregistry.repoAdmin`** - Create and manage Artifact Registry repositories
- **`roles/run.admin`** - Deploy and manage Cloud Run services
- **`roles/datastore.owner`** - Create and manage Firestore databases
- **`roles/compute.networkAdmin`** - Create VPC and networking resources
- **`roles/iam.serviceAccountUser`** - Act as service accounts for Cloud Run
- **`roles/servicenetworking.networksAdmin`** - Manage service networking connections

## One-Command Setup

Run this to grant all permissions at once:

```bash
PROJECT_ID="your-project-id"
SERVICE_ACCOUNT_EMAIL="your-service-account@${PROJECT_ID}.iam.gserviceaccount.com"

for role in \
  "roles/artifactregistry.repoAdmin" \
  "roles/run.admin" \
  "roles/datastore.owner" \
  "roles/compute.networkAdmin" \
  "roles/iam.serviceAccountUser" \
  "roles/servicenetworking.networksAdmin"; do
  
  gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
    --role="${role}"
done
```

After granting these permissions, the GitHub Actions workflow will be able to create all necessary infrastructure.