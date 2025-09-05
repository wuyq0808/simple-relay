#!/bin/bash

# Daily Cost Limit Management Script for Firestore
# This script uses curl to read/write daily cost limit data in Firestore as JSON
# Manages cost limits in the daily_cost_limits collection

set -e  # Exit on any error

# Default values
COMMAND=""
USER_ID=""
COST_LIMIT=""
PROJECT_ID=""
DATABASE=""

# Function to show usage
show_usage() {
    echo "Daily Cost Limit Management Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  set <user_id> <cost_limit>          Set a daily cost limit for a user"
    echo "  get <user_id>                       Get daily cost limit for a user"
    echo "  list                                List all daily cost limits"
    echo ""
    echo "Options:"
    echo "  -p, --project PROJECT_ID            GCP Project ID (required)"
    echo "  -d, --database DATABASE             Database name (required)"
    echo "  -h, --help                          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 set user@example.com 100.50 -p PROJECT_ID -d DATABASE_NAME"
    echo "  $0 get user@example.com -p PROJECT_ID -d DATABASE_NAME"
    echo "  $0 list -p PROJECT_ID -d DATABASE_NAME"
    exit 0
}

# Function to set cost limit
set_cost_limit() {
    local user_id="$1"
    local cost_limit="$2"
    
    echo "📝 Setting daily cost limit: $user_id = \$$cost_limit"
    
    # Get current timestamp
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
    
    # Build JSON payload
    local json_payload
    json_payload="{
  \"fields\": {
    \"userId\": {\"stringValue\": \"$user_id\"},
    \"costLimit\": {\"doubleValue\": $cost_limit},
    \"updateTime\": {\"stringValue\": \"$timestamp\"}
  }
}"
    
    # Get access token
    echo "🔑 Getting access token..."
    ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
    
    if [ -z "$ACCESS_TOKEN" ]; then
        echo "❌ Failed to get access token"
        echo "Please run: gcloud auth application-default login"
        exit 1
    fi
    
    # Firestore REST API endpoint for document
    FIRESTORE_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/daily_cost_limits/$user_id"
    
    echo "📡 Writing to Firestore..."
    
    # Make the PATCH request to write/update the document
    curl -s -X PATCH \
         -H "Authorization: Bearer $ACCESS_TOKEN" \
         -H "Content-Type: application/json" \
         -d "$json_payload" \
         "$FIRESTORE_URL" > /dev/null
    
    if [ $? -eq 0 ]; then
        echo "✅ Set daily cost limit for '$user_id' = \$$cost_limit"
        echo "   Updated: $timestamp"
    else
        echo "❌ Failed to set cost limit"
        exit 1
    fi
}

# Function to get cost limit
get_cost_limit() {
    local user_id="$1"
    
    echo "📖 Getting daily cost limit for: $user_id"
    
    # Get access token
    echo "🔑 Getting access token..."
    ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
    
    if [ -z "$ACCESS_TOKEN" ]; then
        echo "❌ Failed to get access token"
        echo "Please run: gcloud auth application-default login"
        exit 1
    fi
    
    FIRESTORE_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/daily_cost_limits/$user_id"
    
    response=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
                   -H "Content-Type: application/json" \
                   "$FIRESTORE_URL")
    
    echo ""
    if echo "$response" | jq -e '.fields' > /dev/null 2>&1; then
        # Document exists, extract values
        cost_limit=$(echo "$response" | jq -r '.fields.costLimit.doubleValue')
        update_time=$(echo "$response" | jq -r '.fields.updateTime.stringValue')
        
        echo "User: $user_id"
        echo "Daily Cost Limit: \$$cost_limit"
        echo "Last Updated: $update_time"
    else
        echo "No cost limit found for user: $user_id"
    fi
    echo ""
}

# Function to list all cost limits
list_cost_limits() {
    echo "📖 Listing all daily cost limits..."
    
    # Get access token
    echo "🔑 Getting access token..."
    ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
    
    if [ -z "$ACCESS_TOKEN" ]; then
        echo "❌ Failed to get access token"
        echo "Please run: gcloud auth application-default login"
        exit 1
    fi
    
    FIRESTORE_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/daily_cost_limits"
    
    response=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
                   -H "Content-Type: application/json" \
                   "$FIRESTORE_URL")
    
    echo ""
    echo "All daily cost limits:"
    echo ""
    
    if echo "$response" | jq -e '.documents' > /dev/null 2>&1; then
        echo "$response" | jq -r '.documents[] | 
            .fields.userId.stringValue as $userId |
            .fields.costLimit.doubleValue as $costLimit |
            .fields.updateTime.stringValue as $updateTime |
            "User: \($userId)\nDaily Cost Limit: $\($costLimit)\nLast Updated: \($updateTime)\n"'
    else
        echo "No cost limits found"
    fi
}

# Parse command line arguments
if [ $# -eq 0 ]; then
    show_usage
fi

COMMAND="$1"
shift

case "$COMMAND" in
    set)
        if [ $# -lt 2 ]; then
            echo "Error: set command requires user_id and cost_limit"
            echo "Usage: $0 set <user_id> <cost_limit> -p PROJECT_ID -d DATABASE_NAME"
            exit 1
        fi
        USER_ID="$1"
        COST_LIMIT="$2"
        shift 2
        ;;
    get)
        if [ $# -lt 1 ]; then
            echo "Error: get command requires user_id"
            echo "Usage: $0 get <user_id> -p PROJECT_ID -d DATABASE_NAME"
            exit 1
        fi
        USER_ID="$1"
        shift 1
        ;;
    list)
        # No additional arguments needed for list
        ;;
    -h|--help)
        show_usage
        ;;
    *)
        echo "Unknown command: $COMMAND"
        show_usage
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
        -h|--help)
            show_usage
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            ;;
    esac
done

# Validate that PROJECT_ID and DATABASE are set
if [ -z "$PROJECT_ID" ]; then
    echo "Error: Project ID is required. Use -p/--project to specify."
    exit 1
fi

if [ -z "$DATABASE" ]; then
    echo "Error: Database name is required. Use -d/--database to specify."
    exit 1
fi

echo "🚀 Daily Cost Limit Management"
echo "Project ID: $PROJECT_ID"
echo "Database: $DATABASE"
echo ""

# Execute the command
case "$COMMAND" in
    set)
        set_cost_limit "$USER_ID" "$COST_LIMIT"
        ;;
    get)
        get_cost_limit "$USER_ID"
        ;;
    list)
        list_cost_limits
        ;;
esac

echo "✅ Operation completed!"