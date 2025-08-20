#!/bin/bash

# Query Firestore Data via REST API
# This script uses curl to read data from Firestore collections as JSON
# Supports OAuth tokens, billing data, and staging database

set -e  # Exit on any error

# Default values
COLLECTION="oauth_tokens"
ENVIRONMENT="staging"
OUTPUT_FILE=""

# Function to show usage
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -c, --collection COLLECTION    Firestore collection to query (default: oauth_tokens)"
    echo "  -e, --environment ENV          Environment: production, staging (default: staging)"
    echo "  -o, --output FILE              Save output to file"
    echo "  -h, --help                     Show this help message"
    echo ""
    echo "Available collections:"
    echo "  oauth_tokens                   OAuth token data (default)"
    echo "  usage_records                  Billing usage records"
    echo ""
    echo "Examples:"
    echo "  $0                                           # Read oauth_tokens from production"
    echo "  $0 -c usage_records                         # Read billing data from production"  
    echo "  $0 -e staging -c oauth_tokens               # Read oauth_tokens from staging"
    echo "  $0 -c usage_records -o billing-data.json   # Save billing data to file"
    exit 0
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--collection)
            COLLECTION="$2"
            shift 2
            ;;
        -e|--environment)
            ENVIRONMENT="$2"
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

# Load environment variables from .env file
if [ -f .env ]; then
    export $(cat .env | grep -v '#' | awk '/=/ {print $1}')
fi

# Set project ID and database based on environment
case $ENVIRONMENT in
    "production")
        PROJECT_ID="$GCP_PROJECT_ID"
        DATABASE="$FIRESTORE_DATABASE_NAME"
        ;;
    "staging")
        PROJECT_ID="${GCP_PROJECT_ID_STAGING:-$GCP_PROJECT_ID}"
        DATABASE="$FIRESTORE_DATABASE_NAME"
        ;;
    *)
        echo "Error: Invalid environment '$ENVIRONMENT'. Use 'production' or 'staging'"
        exit 1
        ;;
esac

# Check if PROJECT_ID and DATABASE are set
if [ -z "$PROJECT_ID" ]; then
    echo "Error: GCP_PROJECT_ID environment variable is not set for $ENVIRONMENT environment"
    echo "Please set it in your .env file or export it directly"
    if [ "$ENVIRONMENT" = "staging" ]; then
        echo "For staging, you can set GCP_PROJECT_ID_STAGING or it will fall back to GCP_PROJECT_ID"
    fi
    exit 1
fi

if [ -z "$DATABASE" ]; then
    echo "Error: Database name is not set for $ENVIRONMENT environment"
    echo "Please set FIRESTORE_DATABASE_NAME in your .env file"
    exit 1
fi

echo "🚀 Querying Firestore database via REST API..."
echo "Environment: $ENVIRONMENT"
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
        
        # Show collection-specific summary
        case $COLLECTION in
            "usage_records")
                echo ""
                echo "📈 Billing Data Summary:"
                echo "   Total records: $(jq '.documents | length // 0' "$OUTPUT_FILE")"
                if [ "$(jq '.documents | length // 0' "$OUTPUT_FILE")" -gt 0 ]; then
                    echo "   Models used: $(jq -r '.documents[].fields.model.stringValue // empty' "$OUTPUT_FILE" | sort -u | tr '\n' ' ')"
                    echo "   Date range: $(jq -r '.documents[].fields.timestamp.timestampValue // empty' "$OUTPUT_FILE" | sort | head -1) to $(jq -r '.documents[].fields.timestamp.timestampValue // empty' "$OUTPUT_FILE" | sort | tail -1)"
                fi
                ;;
            "oauth_tokens")
                echo ""
                echo "🔐 OAuth Tokens Summary:"
                echo "   Total tokens: $(jq '.documents | length // 0' "$OUTPUT_FILE")"
                ;;
        esac
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

echo ""
echo "💡 Usage examples:"
echo "   # Query billing data:"
echo "   $0 -c usage_records -o billing-data.json"
echo ""
echo "   # Query staging oauth tokens:"
echo "   $0 -e staging -c oauth_tokens"
echo ""
echo "   # Get specific document:"
echo "   curl -H \"Authorization: Bearer \$ACCESS_TOKEN\" \\"
echo "        \"$FIRESTORE_URL/DOCUMENT_ID\" | jq '.'"