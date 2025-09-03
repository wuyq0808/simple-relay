#!/bin/bash

# Grant API Access Script for Firestore
# This script enables API access for a specific user by setting api_enabled to true
# Updates user documents in the users collection

set -e  # Exit on any error

# Default values
EMAIL=""
PROJECT_ID=""
DATABASE=""
REVOKE=false

# Function to show usage
show_usage() {
    echo "Grant API Access Script"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -e, --email EMAIL              User email address (required)"
    echo "  -p, --project PROJECT_ID       GCP Project ID (required)"
    echo "  -d, --database DATABASE        Database name (required)"
    echo "  -r, --revoke                   Revoke API access instead of granting"
    echo "  -h, --help                     Show this help message"
    echo ""
    echo "Examples:"
    echo "  # Grant API access"
    echo "  $0 -e user@example.com -p simple-relay-468808 -d simple-relay-db-staging"
    echo ""
    echo "  # Revoke API access"
    echo "  $0 -e user@example.com -p simple-relay-468808 -d simple-relay-db-staging -r"
    echo ""
    echo "  # Grant API access in production"
    echo "  $0 -e user@example.com -p simple-relay-468808 -d simple-relay-db-production"
    exit 0
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--email)
            EMAIL="$2"
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
        -r|--revoke)
            REVOKE=true
            shift
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

# Validate required parameters
if [ -z "$EMAIL" ]; then
    echo "❌ Error: User email is required. Use -e/--email to specify."
    exit 1
fi

if [ -z "$PROJECT_ID" ]; then
    echo "❌ Error: Project ID is required. Use -p/--project to specify."
    exit 1
fi

if [ -z "$DATABASE" ]; then
    echo "❌ Error: Database name is required. Use -d/--database to specify."
    exit 1
fi

# Set action based on revoke flag
if [ "$REVOKE" = true ]; then
    ACTION="Revoking"
    API_ENABLED="false"
else
    ACTION="Granting"
    API_ENABLED="true"
fi

echo "🚀 $ACTION API access via Firestore REST API..."
echo "Project ID: $PROJECT_ID"
echo "Database: $DATABASE"
echo "User Email: $EMAIL"
echo "API Enabled: $API_ENABLED"
echo ""

# Get access token using gcloud
echo "🔑 Getting access token..."
ACCESS_TOKEN=$(gcloud auth application-default print-access-token)

if [ -z "$ACCESS_TOKEN" ]; then
    echo "❌ Failed to get access token"
    echo "Please run: gcloud auth application-default login"
    exit 1
fi

# Check if user exists first
echo "🔍 Checking if user exists..."
FIRESTORE_GET_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/users/$EMAIL"

USER_EXISTS=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
                   -H "Content-Type: application/json" \
                   "$FIRESTORE_GET_URL" | jq -r '.name // empty')

if [ -z "$USER_EXISTS" ]; then
    echo "❌ Error: User $EMAIL not found in database"
    echo "Please ensure the user has an account first"
    exit 1
fi

echo "✅ User found in database"

# Update the user document to set api_enabled field
echo "📝 Updating user API access..."
FIRESTORE_PATCH_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/users/$EMAIL?updateMask.fieldPaths=api_enabled"

RESPONSE=$(curl -s -X PATCH \
  "$FIRESTORE_PATCH_URL" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"fields\": {
      \"api_enabled\": {
        \"booleanValue\": $API_ENABLED
      }
    }
  }")

# Check if the update was successful
if echo "$RESPONSE" | jq -e '.updateTime' > /dev/null; then
    UPDATE_TIME=$(echo "$RESPONSE" | jq -r '.updateTime')
    echo "✅ API access successfully updated!"
    echo "📅 Update time: $UPDATE_TIME"
    
    # Show current user status
    echo ""
    echo "📊 User Status:"
    echo "   Email: $EMAIL"
    echo "   API Enabled: $API_ENABLED"
    
    if [ "$REVOKE" = true ]; then
        echo ""
        echo "⚠️  API access has been revoked for $EMAIL"
        echo "   The user can no longer create new API keys"
        echo "   Existing API keys will continue to work"
    else
        echo ""
        echo "🎉 API access has been granted to $EMAIL"
        echo "   The user can now create and use API keys"
    fi
else
    echo "❌ Failed to update user API access"
    echo "Response: $RESPONSE"
    exit 1
fi

echo ""
echo "✅ Operation completed successfully!"