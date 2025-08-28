# Frontend Deployment

Frontend deploys as part of the main deployment workflow alongside backend and billing services.

## Required GitHub Secrets

Frontend shares the main deployment workflow, so all main deployment secrets are required:

**Authentication:**
- `GCP_SA_KEY` - Service account JSON key
- `GCP_PROJECT_ID` - Google Cloud project ID

**Application Secrets:**
- `API_SECRET_KEY` - Anthropic API secret key
- `ALLOWED_CLIENT_SECRET_KEY` - Client secret for OAuth
- `API_BASE_URL` - Anthropic API base URL
- `OFFICIAL_BASE_URL` - Anthropic console URL

**Frontend-specific:**
- `RESEND_API_KEY_STAGING` - Staging Resend API key  
- `RESEND_API_KEY_PRODUCTION` - Production Resend API key
- `RESEND_FROM_EMAIL` - noreply@aifastlane.net

## Deployment

- **Push to main** → production deployment (includes frontend)
- **Manual staging** → staging deployment via workflow_dispatch
- **URLs**: `simple-relay-frontend-staging` and `simple-relay-frontend-production` on Cloud Run