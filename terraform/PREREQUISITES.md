# Prerequisites

## Required GCP Resources

- **GCS Bucket**: Create bucket for Terraform state storage
- **Service Account**: GitHub Actions service account with key stored in `GCP_SA_KEY` secret
- **Artifact Registry**: Create `simple-relay` Docker repository in us-central1

## Required IAM Roles

Service account needs these roles:

- `roles/artifactregistry.admin` - Artifact Registry repository creation and management
- `roles/run.admin` - Cloud Run deployment
- `roles/datastore.owner` - Firestore database management
- `roles/compute.networkAdmin` - VPC and networking
- `roles/iam.serviceAccountUser` - Service account impersonation
- `roles/servicenetworking.networksAdmin` - Service networking