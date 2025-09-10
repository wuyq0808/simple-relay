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
- `hourly_aggregates` - Hourly aggregated billing data
- `upstream_account_hourly_aggregates` - Hourly aggregated billing data by OAuth account UUID
- `user_token_bindings` - User token binding system
- `app_config` - Application configuration settings
- `daily_points_limits` - Daily points limits per user (userId, pointsLimit, updateTime)

### Script Usage
```bash
# Read from staging
./scripts/read-firestore.sh -p simple-relay-468808 -d simple-relay-db-staging -c COLLECTION_NAME

# Read from production
./scripts/read-firestore.sh -p simple-relay-468808 -d simple-relay-db-production -c COLLECTION_NAME

# Manage app configuration
./scripts/config-manager.sh -p simple-relay-468808 -d DATABASE_NAME -c get -k CONFIG_KEY
./scripts/config-manager.sh -p simple-relay-468808 -d DATABASE_NAME -c set -k CONFIG_KEY -v VALUE

# Grant API access to users
./scripts/grant-api-access.sh -e USER_EMAIL -p simple-relay-468808 -d simple-relay-db-staging
./scripts/grant-api-access.sh -e USER_EMAIL -p simple-relay-468808 -d simple-relay-db-production

# Revoke API access from users
./scripts/grant-api-access.sh -e USER_EMAIL -p simple-relay-468808 -d DATABASE_NAME -r

# List users with pending access requests
./scripts/grant-api-access.sh -l -p simple-relay-468808 -d simple-relay-db-staging
./scripts/grant-api-access.sh -l -p simple-relay-468808 -d simple-relay-db-production

# Verify billing consistency between usage_records and hourly_aggregates
./scripts/verify-billing-consistency.sh -p simple-relay-468808 -d simple-relay-db-staging
./scripts/verify-billing-consistency.sh -p simple-relay-468808 -d simple-relay-db-staging -u USER_EMAIL -h 2025-09-05T01 -v

# Manage daily points limits  
./scripts/manage-points-limits.sh set USER_EMAIL POINTS_LIMIT -p simple-relay-468808 -d simple-relay-db-staging
./scripts/manage-points-limits.sh get USER_EMAIL -p simple-relay-468808 -d simple-relay-db-staging
./scripts/manage-points-limits.sh list -p simple-relay-468808 -d simple-relay-db-staging

# Manage configuration settings
./scripts/config-manager.sh read CONFIG_KEY -p simple-relay-468808 -d DATABASE_NAME
./scripts/config-manager.sh write CONFIG_KEY VALUE "Description" -p simple-relay-468808 -d DATABASE_NAME
./scripts/config-manager.sh read -p simple-relay-468808 -d DATABASE_NAME  # Read all configs

# Manage OAuth tokens
./scripts/manage-oauth-tokens.sh add USER_EMAIL ACCESS_TOKEN REFRESH_TOKEN "Org Name" -p simple-relay-468808 -d DATABASE_NAME
./scripts/manage-oauth-tokens.sh list -p simple-relay-468808 -d DATABASE_NAME
./scripts/manage-oauth-tokens.sh delete USER_EMAIL -p simple-relay-468808 -d DATABASE_NAME

# Monitor upstream OAuth account usage
./scripts/monitor-upstream-accounts.sh --days 7  # Show last 7 days (default)
./scripts/monitor-upstream-accounts.sh --days 30  # Show last 30 days
./scripts/monitor-upstream-accounts.sh --days 1  # Show today's usage
./scripts/monitor-upstream-accounts.sh -d simple-relay-db-staging --days 1  # Check staging today
./scripts/monitor-upstream-accounts.sh -p simple-relay-468808 -d simple-relay-db-production --days 7  # Production usage
```

## User Registration Limits
```bash
# Manage max user limit (current: 1000 for both staging/production)
./scripts/config-manager.sh read max_registered_users -p simple-relay-468808 -d DATABASE_NAME
./scripts/config-manager.sh write max_registered_users 500 "Max users" -p simple-relay-468808 -d DATABASE_NAME
./scripts/config-manager.sh write max_registered_users 0 "Unlimited" -p simple-relay-468808 -d DATABASE_NAME  # Disable limit
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