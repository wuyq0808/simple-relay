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

