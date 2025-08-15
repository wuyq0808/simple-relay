# Simple Relay Infrastructure

Terraform configuration for deploying simple-relay to Google Cloud with GitHub Actions.

## Quick Setup

1. **Create GCS bucket for Terraform state:**
   ```bash
   gsutil mb gs://your-project-terraform-state
   ```

2. **Create service account with required permissions** (see below)

3. **Add GitHub secrets** (see below) 

4. **Push to main branch** → Automatic deployment

## GitHub Actions Deployment

### Required GitHub Secrets

Add these secrets to your GitHub repository:

#### **Authentication**
- `GCP_SA_KEY` - Service account JSON key with required permissions
- `GCP_PROJECT_ID` - Your Google Cloud project ID
- `TF_STATE_BUCKET` - GCS bucket name for Terraform state storage

#### **Application Secrets**
- `API_SECRET_KEY` - Anthropic API secret key
- `ALLOWED_CLIENT_SECRET_KEY` - Client secret for OAuth
- `API_BASE_URL` - Anthropic API base URL (https://api.anthropic.com)
- `OFFICIAL_BASE_URL` - Anthropic console URL (https://console.anthropic.com)

### Service Account Setup

Create a service account with these IAM roles:
```bash
gcloud iam service-accounts create terraform-deploy
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:terraform-deploy@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/cloudsql.admin"
# Repeat for other roles: run.admin, secretmanager.admin, compute.networkAdmin, 
# serviceusage.serviceUsageAdmin, resourcemanager.projectIamAdmin, storage.admin
```

Download the key and add as `GCP_SA_KEY` secret.

### Deployment Flow

1. **Pull Request**: Runs `terraform plan` to validate changes
2. **Main Branch Push**: Deploys directly to production
3. **Manual Deploy**: Use workflow dispatch for on-demand deployment

## Architecture

```
Internet → Cloud Run → Cloud SQL Proxy → Cloud SQL MySQL
                ↓
          Secret Manager (credentials)
                ↓
            VPC (optional private networking)
```


## Cost Estimation

| Component | Configuration | Monthly Cost |
|-----------|---------------|--------------|
| Cloud SQL | db-f1-micro, 10GB SSD | ~$7 |
| Cloud Run | Minimal usage | ~$0-5 |
| Secret Manager | 4 secrets | ~$0.24 |
| VPC Connector | If enabled | ~$7 |
| **Total** | | **~$7-19/month** |


