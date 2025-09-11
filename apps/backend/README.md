# Backend Service

The main proxy service that handles authentication, OAuth token management, and request routing.

## Quick Start

```bash
# Run tests
make test                 # Run all tests
make test-integration     # Run integration tests only
make test-unit           # Run unit tests only

# Development
make run                 # Run the service locally
make dev                 # Run with hot reload

# Build
make build               # Build binary
make build-docker        # Build Docker image

# Utilities
make lint                # Run linters
make fmt                 # Format code
make help                # Show all available commands
```

## Project Structure

```
backend/
├── Makefile                      # Service-specific commands
├── cmd/
│   └── main.go                  # Application entry point
├── internal/                    # Private application code
│   ├── services/               # Business logic
│   ├── messages/               # Error messages
│   └── ...
├── e2e_test/                    # End-to-end tests
│   ├── docker-compose.test.yml # Test infrastructure
│   ├── mocks/                  # Mock services
│   └── helpers/                # Test utilities
└── scripts/
    ├── test-integration.sh      # Integration test runner
    └── read-firestore.sh        # Database query script
```

## Testing

### Unit Tests
```bash
make test-unit
```

### Integration Tests
Integration tests run against a Firestore emulator in Docker:

```bash
# Run all integration tests
make test-integration

# Run specific test
./scripts/test-integration.sh TestHappyPath

# Run in watch mode (re-runs on file changes)
make test-integration-watch
```

### Test Coverage
```bash
make test-coverage
# Opens coverage.html in browser
```

## Development

### Prerequisites
- Go 1.20+
- Docker & Docker Compose
- Make

### Environment Variables
Create a `.env` file for local development:

```env
API_SECRET_KEY=your-secret-key
OFFICIAL_BASE_URL=https://api.anthropic.com
BILLING_SERVICE_URL=http://localhost:8081
GCP_PROJECT_ID=simple-relay-468808
FIRESTORE_DATABASE_NAME=simple-relay-db-staging
```

### Running Locally
```bash
# Install dependencies
make deps

# Run the service
make run

# Or with hot reload
make dev
```

## Database Scripts

### Query Firestore Collections
```bash
# Read billing data from staging
./scripts/read-firestore.sh -e staging -c usage_records

# Save production oauth tokens to file
./scripts/read-firestore.sh -c oauth_tokens -o tokens.json
```

Available collections:
- `oauth_tokens` - OAuth token data
- `usage_records` - Billing usage records
- `api_key_bindings` - API key to user mappings
- `users` - User accounts

## Code Quality

### Formatting
```bash
make fmt
```

### Linting
```bash
make lint
```

## Architecture

The backend service acts as a proxy between clients and the Claude API:

1. **Authentication**: Validates API keys against Firestore
2. **OAuth Management**: Manages OAuth tokens for upstream API access
3. **Rate Limiting**: Enforces daily points limits
4. **Billing Integration**: Streams usage data to billing service
5. **Request Proxying**: Forwards authenticated requests to Claude API

## Contributing

1. Write tests for new features
2. Ensure all tests pass: `make test`
3. Format code: `make fmt`
4. Run linters: `make lint`
5. Update documentation as needed