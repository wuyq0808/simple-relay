#!/bin/bash

# Migrate daily_cost_limits collection to daily_points_limits collection
# Usage: ./migrate-cost-limits-to-points.sh -p PROJECT_ID -d DATABASE_NAME

set -e

# Default values
PROJECT_ID=""
DATABASE_NAME=""
DRY_RUN=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -p|--project)
      PROJECT_ID="$2"
      shift 2
      ;;
    -d|--database)
      DATABASE_NAME="$2"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    *)
      echo "Unknown option $1"
      echo "Usage: $0 -p PROJECT_ID -d DATABASE_NAME [--dry-run]"
      exit 1
      ;;
  esac
done

# Validate required parameters
if [[ -z "$PROJECT_ID" || -z "$DATABASE_NAME" ]]; then
  echo "Error: PROJECT_ID and DATABASE_NAME are required"
  echo "Usage: $0 -p PROJECT_ID -d DATABASE_NAME [--dry-run]"
  exit 1
fi

echo "🚀 Migrating daily_cost_limits to daily_points_limits"
echo "Project ID: $PROJECT_ID"
echo "Database: $DATABASE_NAME"
if [[ "$DRY_RUN" = true ]]; then
  echo "DRY RUN MODE - No changes will be made"
fi

# Get access token
echo "🔑 Getting access token..."
ACCESS_TOKEN=$(gcloud auth print-access-token)

# Read all documents from daily_cost_limits
echo "📖 Reading documents from daily_cost_limits..."
OLD_COLLECTION_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE_NAME/documents/daily_cost_limits"

RESPONSE=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" "$OLD_COLLECTION_URL")

# Check if collection exists
if [[ $(echo "$RESPONSE" | jq -r '.error.code // "null"') != "null" ]]; then
  echo "❌ Error reading collection: $(echo "$RESPONSE" | jq -r '.error.message')"
  exit 1
fi

# Get document count
DOC_COUNT=$(echo "$RESPONSE" | jq -r '.documents // [] | length')
echo "📊 Found $DOC_COUNT documents to migrate"

if [[ $DOC_COUNT -eq 0 ]]; then
  echo "✅ No documents to migrate"
  exit 0
fi

# Process each document
echo "🔄 Processing documents..."
NEW_COLLECTION_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE_NAME/documents/daily_points_limits"

echo "$RESPONSE" | jq -c '.documents[]' | while read -r doc; do
  # Extract document ID and data
  DOC_NAME=$(echo "$doc" | jq -r '.name')
  DOC_ID=$(basename "$DOC_NAME")
  
  # Extract current fields
  USER_ID=$(echo "$doc" | jq -r '.fields.userId.stringValue')
  COST_LIMIT=$(echo "$doc" | jq -r '.fields.costLimit.doubleValue')
  UPDATE_TIME=$(echo "$doc" | jq -r '.fields.updateTime.stringValue')
  
  # Convert cost limit to points limit (cost * 1000)
  POINTS_LIMIT=$(echo "$COST_LIMIT * 1000" | bc)
  POINTS_LIMIT_INT=${POINTS_LIMIT%.*} # Remove decimal part
  
  echo "  📝 Migrating $DOC_ID: $COST_LIMIT -> $POINTS_LIMIT_INT points"
  
  if [[ "$DRY_RUN" = false ]]; then
    # Create new document in daily_points_limits
    NEW_DOC=$(cat <<EOF
{
  "fields": {
    "userId": {"stringValue": "$USER_ID"},
    "pointsLimit": {"integerValue": "$POINTS_LIMIT_INT"},
    "updateTime": {"stringValue": "$UPDATE_TIME"}
  }
}
EOF
)
    
    RESULT=$(curl -s -X PATCH \
      -H "Authorization: Bearer $ACCESS_TOKEN" \
      -H "Content-Type: application/json" \
      -d "$NEW_DOC" \
      "$NEW_COLLECTION_URL/$DOC_ID")
    
    if [[ $(echo "$RESULT" | jq -r '.error.code // "null"') != "null" ]]; then
      echo "    ❌ Error: $(echo "$RESULT" | jq -r '.error.message')"
    else
      echo "    ✅ Migrated successfully"
    fi
  fi
done

if [[ "$DRY_RUN" = false ]]; then
  echo ""
  echo "🎉 Migration completed!"
  echo ""
  echo "⚠️  IMPORTANT: After verifying the migration worked correctly,"
  echo "   you should manually delete the old daily_cost_limits collection"
  echo "   and update your application code to use daily_points_limits"
else
  echo ""
  echo "🔍 Dry run completed - no changes made"
fi