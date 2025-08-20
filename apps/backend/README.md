# Backend Services

## Database Query Script

### Usage
```bash
./scripts/read-firestore.sh [options]
```

### Options
- `-c, --collection` - Collection to query (default: oauth_tokens)
- `-e, --environment` - Environment: production, staging (default: staging)
- `-o, --output` - Save output to file
- `-h, --help` - Show help

### Examples
```bash
# Read billing data from staging
./scripts/read-firestore.sh -e staging -c usage_records

# Save production oauth tokens to file
./scripts/read-firestore.sh -c oauth_tokens -o tokens.json

# Read billing data and save to file
./scripts/read-firestore.sh -c usage_records -o billing-data.json
```

### Collections
- `oauth_tokens` - OAuth token data
- `usage_records` - Billing usage records

### Setup
Ensure `.env` file contains:
```
GCP_PROJECT_ID=simple-relay-468808
FIRESTORE_DATABASE_NAME=simple-relay-db-staging
```