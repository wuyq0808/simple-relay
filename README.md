# Simple Relay

A monorepo with React frontend and Golang backend deployed to Google Cloud.

## Structure

- `apps/frontend/` - React + Express frontend
- `apps/backend/` - Golang API backend

## Development

**Backend:**
```bash
cd apps/backend
go run cmd/main.go
```

**Frontend:**
```bash
cd apps/frontend
npm install
npm run build
npm run dev
```

## Deployment

The Golang backend automatically deploys to Google Cloud Run on push to main.

### Setup Required:

1. Create GCP project
2. Enable Cloud Run API
3. Create service account with Cloud Run Admin role
4. Add these GitHub secrets:
   - `GCP_PROJECT_ID` - Your GCP project ID
   - `GCP_SA_KEY` - Service account JSON key

### Manual Deploy:
```bash
cd apps/backend
gcloud run deploy simple-relay-backend --source .
```
