#!/bin/bash

# OAuth Token Management Script for Firestore
# Manages OAuth tokens in the oauth_tokens collection

set -e  # Exit on any error

# Default values
COMMAND=""
EMAIL=""
ACCESS_TOKEN=""
REFRESH_TOKEN=""
ORGANIZATION=""
ACCOUNT_UUID=""
ORG_UUID=""
PROJECT_ID=""
DATABASE=""
EXPIRES_HOURS=8

# Function to show usage
show_usage() {
    echo "🚀 OAuth Token Management"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  add <email> <access_token> <refresh_token> [org_name] [account_uuid] [org_uuid]   Add OAuth token"
    echo "  list                                                    List all OAuth tokens"
    echo "  delete <email>                                          Delete OAuth token by email"
    echo ""
    echo "Options:"
    echo "  -p, --project PROJECT_ID            GCP Project ID (required)"
    echo "  -d, --database DATABASE             Database name (required)"
    echo "  -e, --expires HOURS                 Token expiration hours (default: 8)"
    echo "  -h, --help                          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 add user@example.com sk-ant-oat01-... sk-ant-ort01-... \"User's Org\" account-uuid org-uuid -p PROJECT_ID -d DATABASE_NAME"
    echo "  $0 list -p PROJECT_ID -d DATABASE_NAME"
    echo "  $0 delete user@example.com -p PROJECT_ID -d DATABASE_NAME"
}

# Function to get access token
get_access_token() {
    gcloud auth application-default print-access-token
}

# Function to add OAuth token
add_oauth_token() {
    local email="$1"
    local access_token="$2"
    local refresh_token="$3"
    local org_name="$4"
    local account_uuid="$5"
    local org_uuid="$6"
    
    if [[ -z "$email" || -z "$access_token" || -z "$refresh_token" || -z "$account_uuid" || -z "$org_uuid" ]]; then
        echo "❌ Error: Email, access token, refresh token, account UUID, and organization UUID are required"
        exit 1
    fi
    
    if [[ -z "$org_name" ]]; then
        org_name="${email}s Organization"
    fi
    
    # Keep single quotes as-is - they're valid in JSON strings
    
    echo "📝 Adding OAuth token for: $email"
    
    # Generate timestamps
    local current_time=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local expires_at=$(date -u -d "+${EXPIRES_HOURS} hours" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -v "+${EXPIRES_HOURS}H" +"%Y-%m-%dT%H:%M:%SZ")
    
    # UUIDs are now required parameters
    
    echo "🔑 Getting access token..."
    local gcp_token=$(get_access_token)
    
    echo "📡 Writing to Firestore..."
    
    # Create JSON payload file to avoid parameter issues
    cat > /tmp/oauth_payload.json <<EOF
{
    "fields": {
        "access_token": {
            "stringValue": "${access_token}"
        },
        "refresh_token": {
            "stringValue": "${refresh_token}"
        },
        "account_email": {
            "stringValue": "${email}"
        },
        "organization_name": {
            "stringValue": "${org_name}"
        },
        "scope": {
            "stringValue": "user:inference user:profile"
        },
        "account_uuid": {
            "stringValue": "${account_uuid}"
        },
        "organization_uuid": {
            "stringValue": "${org_uuid}"
        },
        "expires_at": {
            "timestampValue": "${expires_at}"
        },
        "updated_at": {
            "timestampValue": "${current_time}"
        },
        "refresh_started_at": {
            "timestampValue": "${current_time}"
        }
    }
}
EOF

    # Make the API call using file input to avoid parameter size issues
    # Use PATCH with documentId to create/update with specific document ID (account_uuid)
    curl -s -X PATCH \
        "https://firestore.googleapis.com/v1/projects/${PROJECT_ID}/databases/${DATABASE}/documents/oauth_tokens/${account_uuid}" \
        -H "Authorization: Bearer ${gcp_token}" \
        -H "Content-Type: application/json" \
        -d @/tmp/oauth_payload.json > /tmp/oauth_response.json
    
    local curl_exit_code=$?
    
    if [[ $curl_exit_code -ne 0 ]]; then
        echo "❌ Curl failed with exit code: $curl_exit_code"
        exit 1
    fi
    
    local response=$(cat /tmp/oauth_response.json)
    
    if [[ -z "$response" ]]; then
        echo "❌ Empty response from API"
        exit 1
    fi
    
    # Check for errors
    if echo "$response" | grep -q "error"; then
        echo "❌ Error adding OAuth token:"
        echo "$response" | jq -r '.error.message' 2>/dev/null || echo "$response"
        exit 1
    fi
    
    # Check if response contains a document name (indicates success)
    if echo "$response" | grep -q "\"name\""; then
        echo "✅ Successfully created OAuth token document"
    else
        echo "⚠️ Unexpected response format - token may not have been created"
    fi
    
    
    echo "✅ Added OAuth token for '$email'"
    echo "   Organization: $org_name"
    echo "   Expires: $expires_at"
}

# Function to list OAuth tokens
list_oauth_tokens() {
    echo "📖 Listing OAuth tokens..."
    
    local gcp_token=$(get_access_token)
    
    local response=$(curl -s -X GET \
        "https://firestore.googleapis.com/v1/projects/${PROJECT_ID}/databases/${DATABASE}/documents/oauth_tokens" \
        -H "Authorization: Bearer ${gcp_token}")
    
    # Check for errors
    if echo "$response" | grep -q "error"; then
        echo "❌ Error listing OAuth tokens:"
        echo "$response" | jq -r '.error.message' 2>/dev/null || echo "$response"
        exit 1
    fi
    
    # Parse and display tokens
    echo "$response" | jq -r '.documents[]? | 
        "Email: " + .fields.account_email.stringValue + 
        " | Organization: " + .fields.organization_name.stringValue + 
        " | Expires: " + .fields.expires_at.timestampValue + 
        " | Updated: " + .fields.updated_at.timestampValue'
}

# Function to delete OAuth token
delete_oauth_token() {
    local email="$1"
    
    if [[ -z "$email" ]]; then
        echo "❌ Error: Email is required"
        exit 1
    fi
    
    echo "🗑️  Deleting OAuth token for: $email"
    
    local gcp_token=$(get_access_token)
    
    # First, find the document ID by listing and filtering
    local response=$(curl -s -X GET \
        "https://firestore.googleapis.com/v1/projects/${PROJECT_ID}/databases/${DATABASE}/documents/oauth_tokens" \
        -H "Authorization: Bearer ${gcp_token}")
    
    local doc_name=$(echo "$response" | jq -r --arg email "$email" '.documents[]? | select(.fields.account_email.stringValue == $email) | .name')
    
    if [[ -z "$doc_name" ]]; then
        echo "❌ OAuth token not found for email: $email"
        exit 1
    fi
    
    # Delete the document
    local delete_response=$(curl -s -X DELETE \
        "https://firestore.googleapis.com/v1/${doc_name}" \
        -H "Authorization: Bearer ${gcp_token}")
    
    echo "✅ Deleted OAuth token for '$email'"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        add)
            COMMAND="add"
            shift
            # Parse add command arguments
            EMAIL="$1"; shift
            ACCESS_TOKEN="$1"; shift
            REFRESH_TOKEN="$1"; shift
            # Check if next argument is organization (not a flag)
            if [[ $# -gt 0 && ! "$1" =~ ^- ]]; then
                ORGANIZATION="$1"; shift
            fi
            # Check if next argument is account_uuid (not a flag)
            if [[ $# -gt 0 && ! "$1" =~ ^- ]]; then
                ACCOUNT_UUID="$1"; shift
            fi
            # Check if next argument is org_uuid (not a flag)
            if [[ $# -gt 0 && ! "$1" =~ ^- ]]; then
                ORG_UUID="$1"; shift
            fi
            ;;
        list)
            COMMAND="list"
            shift
            ;;
        delete)
            COMMAND="delete"
            shift
            EMAIL="$1"; shift
            ;;
        -p|--project)
            PROJECT_ID="$2"
            shift 2
            ;;
        -d|--database)
            DATABASE="$2"
            shift 2
            ;;
        -e|--expires)
            EXPIRES_HOURS="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Validate required parameters
if [[ -z "$PROJECT_ID" || -z "$DATABASE" ]]; then
    echo "❌ Error: Project ID and Database are required"
    echo ""
    show_usage
    exit 1
fi

if [[ -z "$COMMAND" ]]; then
    echo "❌ Error: Command is required"
    echo ""
    show_usage
    exit 1
fi

# Check if jq is available (for JSON parsing)
if ! command -v jq &> /dev/null; then
    echo "⚠️  jq not found. JSON output may not be formatted nicely."
fi

echo "🚀 OAuth Token Management"
echo "Project ID: $PROJECT_ID"
echo "Database: $DATABASE"
echo ""

# Execute command
case $COMMAND in
    add)
        add_oauth_token "$EMAIL" "$ACCESS_TOKEN" "$REFRESH_TOKEN" "$ORGANIZATION" "$ACCOUNT_UUID" "$ORG_UUID"
        ;;
    list)
        list_oauth_tokens
        ;;
    delete)
        delete_oauth_token "$EMAIL"
        ;;
    *)
        echo "❌ Unknown command: $COMMAND"
        exit 1
        ;;
esac

echo "✅ Operation completed!"