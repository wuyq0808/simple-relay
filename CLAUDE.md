# Claude Code Configuration

## Database Information

### Project ID
```
simple-relay-468808
```

### Databases
- **Staging**: `simple-relay-db-staging`
- **Production**: `simple-relay-db-production`

### Collections
- `users` - User accounts and authentication data
- `api_key_bindings` - API key to user email bindings
- `oauth_tokens` - OAuth token data
- `usage_records` - Billing usage records  
- `user_token_bindings` - User token binding system
- `app_config` - Application configuration settings

### Script Usage
```bash
# Read from staging
./scripts/read-firestore.sh -p simple-relay-468808 -d simple-relay-db-staging -c COLLECTION_NAME

# Read from production
./scripts/read-firestore.sh -p simple-relay-468808 -d simple-relay-db-production -c COLLECTION_NAME

# Manage app configuration
./scripts/config-manager.sh -p simple-relay-468808 -d DATABASE_NAME -c get -k CONFIG_KEY
./scripts/config-manager.sh -p simple-relay-468808 -d DATABASE_NAME -c set -k CONFIG_KEY -v VALUE
```

## Development Server Rules
- ALWAYS run development servers (npm run dev, yarn dev, etc.) in background using run_in_background: true
- Never run development servers without the background flag as they will timeout and block execution

## Deployment Instructions

### Deploy to Staging
```bash
# Trigger staging deployment from any branch
gh workflow run "Staging" --ref BRANCH_NAME

# Example:
gh workflow run "Staging" --ref feature/frontend-signup
```

### NEVER Deploy to Production
- DO NOT trigger production deployments automatically
- Production deployments require manual approval and testing
- Only deploy to staging for testing purposes

### Service URLs
- **Frontend Production**: https://simple-relay-frontend-production-573916960175.us-central1.run.app
- **Frontend Staging**: https://simple-relay-frontend-staging-573916960175.us-central1.run.app
- **Backend Production**: https://simple-relay-production-573916960175.us-central1.run.app
- **Backend Staging**: https://simple-relay-staging-573916960175.us-central1.run.app
- **Billing Production**: https://simple-relay-billing-production-573916960175.us-central1.run.app
- **Billing Staging**: https://simple-relay-billing-staging-573916960175.us-central1.run.app

### Required Environment Variables (GitHub Secrets)
- `RESEND_API_KEY_STAGING` / `RESEND_API_KEY_PRODUCTION` - Email service API keys
- `RESEND_FROM_EMAIL` - noreply@aifastlane.net
- `GCP_SA_KEY` - Service account JSON key