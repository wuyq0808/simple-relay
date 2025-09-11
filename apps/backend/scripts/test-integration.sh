#!/bin/bash

# Backend Integration Test Runner
# Runs integration tests with Firestore emulator

set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BACKEND_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"
E2E_DIR="$BACKEND_DIR/e2e_test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting backend integration tests...${NC}"

# Change to e2e test directory
cd "$E2E_DIR"

# No watch mode - always run once

# Start Firestore emulator
echo -e "${YELLOW}📦 Starting Firestore emulator...${NC}"
docker-compose -f docker-compose.test.yml up -d

# Function to cleanup on exit
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up...${NC}"
    cd "$E2E_DIR"
    docker-compose -f docker-compose.test.yml down
}

# Register cleanup function
trap cleanup EXIT

# Wait for Firestore emulator to be ready
echo -e "${YELLOW}⏳ Waiting for Firestore emulator to be ready...${NC}"
for i in {1..30}; do
    if curl -s http://localhost:8080 > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Firestore emulator is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}❌ Firestore emulator failed to start${NC}"
        docker-compose -f docker-compose.test.yml logs
        exit 1
    fi
    sleep 1
done

# Set environment variables
export FIRESTORE_EMULATOR_HOST=localhost:8080
export GCP_PROJECT_ID=test-project

# Run tests
echo -e "${BLUE}🧪 Running E2E integration tests...${NC}"
cd "$BACKEND_DIR"

# Check if we should run specific tests
if [ -n "$1" ]; then
    echo "Running specific test: $1"
    go test -v ./e2e_test/... -run "$1" -timeout 30s
else
    # Run all E2E tests by default
    go test -v ./e2e_test/... -run "TestE2E" -timeout 30s
fi

TEST_RESULT=$?

if [ $TEST_RESULT -eq 0 ]; then
    echo -e "${GREEN}✅ All backend integration tests passed!${NC}"
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit $TEST_RESULT
fi