#!/bin/bash

# Query Firestore Data via REST API
# This script uses curl to read data from Firestore collections as JSON
# Supports OAuth tokens, billing data, and staging database

set -e  # Exit on any error

# Default values
COLLECTION="oauth_tokens"
OUTPUT_FILE=""
PROJECT_ID=""
DATABASE=""

# Function to show usage
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -c, --collection COLLECTION    Firestore collection to query (default: oauth_tokens)"
    echo "  -p, --project PROJECT_ID       GCP Project ID (required)"
    echo "  -d, --database DATABASE        Database name (required)"
    echo "  -o, --output FILE              Save output to file"
    echo "  -h, --help                     Show this help message"
    echo ""
    echo "Available collections:"
    echo "  oauth_tokens                   OAuth token data (default)"
    echo "  usage_records                  Billing usage records"
    echo ""
    echo "Examples:"
    echo "  $0 -p PROJECT_ID -d DATABASE_NAME"
    echo "  $0 -p PROJECT_ID -d DATABASE_NAME -c usage_records"
    echo "  $0 -p PROJECT_ID -d DATABASE_NAME -c oauth_tokens -o output.json"
    exit 0
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--collection)
            COLLECTION="$2"
            shift 2
            ;;
        -p|--project)
            PROJECT_ID="$2"
            shift 2
            ;;
        -d|--database)
            DATABASE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            ;;
    esac
done

# No environment validation needed anymore

# Validate that PROJECT_ID and DATABASE are set
if [ -z "$PROJECT_ID" ]; then
    echo "Error: Project ID is required. Use -p/--project to specify."
    exit 1
fi

if [ -z "$DATABASE" ]; then
    echo "Error: Database name is required. Use -d/--database to specify."
    exit 1
fi

echo "🚀 Querying Firestore database via REST API..."
echo "Project ID: $PROJECT_ID"
echo "Database: $DATABASE"
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
FIRESTORE_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/$COLLECTION"

echo "📡 Querying Firestore REST API..."
echo "URL: $FIRESTORE_URL"
echo ""

# Make the curl request and handle output
if [ -n "$OUTPUT_FILE" ]; then
    echo "💾 Saving results to: $OUTPUT_FILE"
    curl -H "Authorization: Bearer $ACCESS_TOKEN" \
         -H "Content-Type: application/json" \
         "$FIRESTORE_URL" | jq '.' > "$OUTPUT_FILE"
    
    if [ $? -eq 0 ]; then
        echo "✅ Query completed and saved to $OUTPUT_FILE"
        echo "📊 Document count: $(jq '.documents | length // 0' "$OUTPUT_FILE")"
        
        # Show collection summary
        echo ""
        echo "📊 Collection Summary ($COLLECTION):"
        echo "   Total documents: $(jq '.documents | length // 0' "$OUTPUT_FILE")"
    else
        echo "❌ Failed to save results"
        exit 1
    fi
else
    curl -H "Authorization: Bearer $ACCESS_TOKEN" \
         -H "Content-Type: application/json" \
         "$FIRESTORE_URL" | jq '.'
    echo ""
    echo "✅ Query completed!"
fi

