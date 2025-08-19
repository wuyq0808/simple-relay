#!/bin/bash

# Read Firestore Data via REST API
# This script uses curl to read OAuth tokens from Firestore as JSON

set -e  # Exit on any error

# Load environment variables from .env file
if [ -f .env ]; then
    export $(cat .env | grep -v '#' | awk '/=/ {print $1}')
fi

# Check if GCP_PROJECT_ID is set
if [ -z "$GCP_PROJECT_ID" ]; then
    echo "Error: GCP_PROJECT_ID environment variable is not set"
    echo "Please set it in your .env file or export it directly"
    exit 1
fi

# Configuration
PROJECT_ID="$GCP_PROJECT_ID"
COLLECTION="oauth_tokens"

echo "🚀 Reading Firestore data via REST API..."
echo "Project ID: $PROJECT_ID"
echo "Collection: $COLLECTION"

# Get access token using gcloud
echo "🔑 Getting access token..."
ACCESS_TOKEN=$(gcloud auth application-default print-access-token)

if [ -z "$ACCESS_TOKEN" ]; then
    echo "❌ Failed to get access token"
    echo "Please run: gcloud auth application-default login"
    exit 1
fi

# Firestore REST API endpoint
FIRESTORE_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/(default)/documents/$COLLECTION"

echo "📡 Querying Firestore REST API..."
echo "URL: $FIRESTORE_URL"
echo ""

# Make the curl request
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "Content-Type: application/json" \
     "$FIRESTORE_URL" | jq '.'

echo ""
echo "✅ Query completed!"
echo ""
echo "💡 To save to file:"
echo "   ./scripts/read-firestore.sh > firestore-data.json"
echo ""
echo "💡 To get a specific document:"
echo "   curl -H \"Authorization: Bearer \$ACCESS_TOKEN\" \\"
echo "        \"$FIRESTORE_URL/DOCUMENT_ID\" | jq '.'"