#!/bin/bash

# Daily Points Limit Management Script for Firestore
# This script uses curl to read/write daily points limit data in Firestore as JSON
# Manages points limits in the daily_points_limits collection

set -e  # Exit on any error

# Default values
COMMAND=""
USER_ID=""
POINTS_LIMIT=""
PROJECT_ID=""
DATABASE=""

# Function to show usage
show_usage() {
    echo "Daily Points Limit Management Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  set USER_EMAIL POINTS_LIMIT    Set points limit for user"
    echo "  get USER_EMAIL                 Get points limit for user"
    echo "  list                          List all points limits"
    echo ""
    echo "Options:"
    echo "  -p, --project PROJECT_ID      GCP Project ID"
    echo "  -d, --database DATABASE       Firestore Database ID"
    echo ""
    echo "Examples:"
    echo "  $0 set user@example.com 1000 -p my-project -d my-database"
    echo "  $0 get user@example.com -p my-project -d my-database"
    echo "  $0 list -p my-project -d my-database"
}

# Parse command line arguments
COMMAND="$1"
shift

case "$COMMAND" in
    set)
        USER_ID="$1"
        POINTS_LIMIT="$2"
        shift 2
        ;;
    get)
        USER_ID="$1"
        shift
        ;;
    list)
        ;;
    *)
        echo "Error: Invalid command '$COMMAND'"
        show_usage
        exit 1
        ;;
esac

# Parse remaining options
while [[ $# -gt 0 ]]; do
  case $1 in
    -p|--project)
      PROJECT_ID="$2"
      shift 2
      ;;
    -d|--database)
      DATABASE="$2"
      shift 2
      ;;
    *)
      echo "Unknown option $1"
      show_usage
      exit 1
      ;;
  esac
done

# Validate required parameters
if [[ -z "$PROJECT_ID" || -z "$DATABASE" ]]; then
  echo "Error: PROJECT_ID and DATABASE are required"
  show_usage
  exit 1
fi

if [[ "$COMMAND" == "set" && (-z "$USER_ID" || -z "$POINTS_LIMIT") ]]; then
  echo "Error: USER_EMAIL and POINTS_LIMIT are required for set command"
  show_usage
  exit 1
fi

if [[ "$COMMAND" == "get" && -z "$USER_ID" ]]; then
  echo "Error: USER_EMAIL is required for get command"
  show_usage
  exit 1
fi

echo "🚀 Daily Points Limit Management"
echo "Project ID: $PROJECT_ID"
echo "Database: $DATABASE"
echo ""

# Get access token
echo "🔑 Getting access token..."
ACCESS_TOKEN=$(gcloud auth print-access-token)

# Base URL for Firestore REST API
COLLECTION_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/daily_points_limits"

case "$COMMAND" in
    set)
        echo "📝 Setting daily points limit: $USER_ID = $POINTS_LIMIT points"
        
        # Create document data
        DOC_DATA=$(cat <<EOF
{
  "fields": {
    "userId": {"stringValue": "$USER_ID"},
    "pointsLimit": {"integerValue": "$POINTS_LIMIT"},
    "updateTime": {"stringValue": "$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")"}
  }
}
EOF
)
        
        # Write to Firestore
        echo "📡 Writing to Firestore..."
        RESULT=$(curl -s -X PATCH \
          -H "Authorization: Bearer $ACCESS_TOKEN" \
          -H "Content-Type: application/json" \
          -d "$DOC_DATA" \
          "$COLLECTION_URL/$USER_ID")
        
        if [[ $(echo "$RESULT" | jq -r '.error.code // "null"') != "null" ]]; then
          echo "❌ Error: $(echo "$RESULT" | jq -r '.error.message')"
          exit 1
        fi
        
        echo "✅ Set daily points limit for '$USER_ID' = $POINTS_LIMIT points"
        UPDATE_TIME=$(echo "$RESULT" | jq -r '.fields.updateTime.stringValue')
        echo "   Updated: $UPDATE_TIME"
        ;;
        
    get)
        echo "📖 Getting daily points limit for: $USER_ID"
        
        RESULT=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
          "$COLLECTION_URL/$USER_ID")
        
        if [[ $(echo "$RESULT" | jq -r '.error.code // "null"') != "null" ]]; then
          if [[ $(echo "$RESULT" | jq -r '.error.code') == "NOT_FOUND" ]]; then
            echo ""
            echo "No points limit found for user: $USER_ID"
          else
            echo "❌ Error: $(echo "$RESULT" | jq -r '.error.message')"
            exit 1
          fi
        else
          POINTS_LIMIT=$(echo "$RESULT" | jq -r '.fields.pointsLimit.integerValue')
          UPDATE_TIME=$(echo "$RESULT" | jq -r '.fields.updateTime.stringValue')
          echo ""
          echo "📊 Points limit for $USER_ID:"
          echo "   Points Limit: $POINTS_LIMIT"
          echo "   Updated: $UPDATE_TIME"
        fi
        ;;
        
    list)
        echo "📋 Listing all daily points limits..."
        
        RESULT=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
          "$COLLECTION_URL")
        
        if [[ $(echo "$RESULT" | jq -r '.error.code // "null"') != "null" ]]; then
          echo "❌ Error: $(echo "$RESULT" | jq -r '.error.message')"
          exit 1
        fi
        
        DOC_COUNT=$(echo "$RESULT" | jq -r '.documents // [] | length')
        echo ""
        echo "📊 Found $DOC_COUNT points limits:"
        echo ""
        
        if [[ $DOC_COUNT -gt 0 ]]; then
          printf "%-30s %-15s %-25s\n" "USER_EMAIL" "POINTS_LIMIT" "UPDATE_TIME"
          printf "%-30s %-15s %-25s\n" "$(printf "%*s" 30 "" | tr " " "-")" "$(printf "%*s" 15 "" | tr " " "-")" "$(printf "%*s" 25 "" | tr " " "-")"
          
          echo "$RESULT" | jq -r '.documents[] | [
            .fields.userId.stringValue,
            .fields.pointsLimit.integerValue,
            .fields.updateTime.stringValue
          ] | @tsv' | while IFS=$'\t' read -r user_id points_limit update_time; do
            printf "%-30s %-15s %-25s\n" "$user_id" "$points_limit" "$update_time"
          done
        fi
        ;;
esac

echo ""
echo "✅ Operation completed!"