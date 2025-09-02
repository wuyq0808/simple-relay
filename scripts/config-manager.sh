#!/bin/bash

# Configuration Management Script for Firestore
# This script uses curl to read/write configuration data in Firestore as JSON
# Manages app configuration in the app_config collection

set -e  # Exit on any error

# Default values
COMMAND=""
KEY=""
VALUE=""
DESCRIPTION=""
PROJECT_ID=""
DATABASE=""

# Function to show usage
show_usage() {
    echo "Configuration Management Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  write <key> <value> [description]   Write a configuration value"
    echo "  read [key1] [key2] ...              Read configuration values"
    echo ""
    echo "Options:"
    echo "  -p, --project PROJECT_ID            GCP Project ID (required)"
    echo "  -d, --database DATABASE             Database name (required)"
    echo "  -h, --help                          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 write signup_enabled false \"Disable signup\" -p PROJECT_ID -d DATABASE_NAME"
    echo "  $0 read signup_enabled -p PROJECT_ID -d DATABASE_NAME"
    echo "  $0 read -p PROJECT_ID -d DATABASE_NAME"
    exit 0
}

# Function to write configuration
write_config() {
    local key="$1"
    local value="$2"
    local description="$3"
    
    echo "📝 Writing configuration: $key = $value"
    
    # Get the Firestore value type
    local value_type
    value_type=$(get_firestore_type "$value")
    
    # Convert string values to appropriate JSON types
    local json_value
    if [ "$value" = "true" ] || [ "$value" = "false" ]; then
        json_value="$value"
    elif [[ "$value" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
        json_value="$value"
    else
        json_value="\"$value\""
    fi
    
    # Get current timestamp
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
    
    # Build JSON payload
    local json_payload
    if [ -n "$description" ]; then
        json_payload="{
  \"fields\": {
    \"key\": {\"stringValue\": \"$key\"},
    \"value\": {\"$value_type\": $json_value},
    \"description\": {\"stringValue\": \"$description\"},
    \"updated_at\": {\"stringValue\": \"$timestamp\"}
  }
}"
    else
        json_payload="{
  \"fields\": {
    \"key\": {\"stringValue\": \"$key\"},
    \"value\": {\"$value_type\": $json_value},
    \"updated_at\": {\"stringValue\": \"$timestamp\"}
  }
}"
    fi
    
    # Get access token
    echo "🔑 Getting access token..."
    ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
    
    if [ -z "$ACCESS_TOKEN" ]; then
        echo "❌ Failed to get access token"
        echo "Please run: gcloud auth application-default login"
        exit 1
    fi
    
    # Firestore REST API endpoint for document
    FIRESTORE_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/app_config/$key"
    
    echo "📡 Writing to Firestore..."
    
    # Make the PATCH request to write/update the document
    curl -s -X PATCH \
         -H "Authorization: Bearer $ACCESS_TOKEN" \
         -H "Content-Type: application/json" \
         -d "$json_payload" \
         "$FIRESTORE_URL" > /dev/null
    
    if [ $? -eq 0 ]; then
        echo "✅ Set config '$key' = $value"
        if [ -n "$description" ]; then
            echo "   Description: $description"
        fi
    else
        echo "❌ Failed to write configuration"
        exit 1
    fi
}

# Function to get Firestore value type
get_firestore_type() {
    local value="$1"
    if [ "$value" = "true" ] || [ "$value" = "false" ]; then
        echo "booleanValue"
    elif [[ "$value" =~ ^[0-9]+$ ]]; then
        echo "integerValue"
    elif [[ "$value" =~ ^[0-9]+\.[0-9]+$ ]]; then
        echo "doubleValue"
    else
        echo "stringValue"
    fi
}

# Function to read configuration
read_config() {
    local keys=("$@")
    
    echo "📖 Reading configuration..."
    
    # Get access token
    echo "🔑 Getting access token..."
    ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
    
    if [ -z "$ACCESS_TOKEN" ]; then
        echo "❌ Failed to get access token"
        echo "Please run: gcloud auth application-default login"
        exit 1
    fi
    
    if [ ${#keys[@]} -gt 0 ]; then
        # Read specific keys
        echo ""
        echo "Reading specific keys: ${keys[*]}"
        echo ""
        
        for key in "${keys[@]}"; do
            FIRESTORE_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/app_config/$key"
            
            response=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
                           -H "Content-Type: application/json" \
                           "$FIRESTORE_URL")
            
            if echo "$response" | jq -e '.fields' > /dev/null 2>&1; then
                # Document exists, extract value
                value=$(echo "$response" | jq -r '.fields.value | to_entries[0].value')
                description=$(echo "$response" | jq -r '.fields.description.stringValue // empty')
                updated_at=$(echo "$response" | jq -r '.fields.updated_at.stringValue // empty')
                
                echo "$key: $value"
                if [ -n "$description" ] && [ "$description" != "null" ]; then
                    echo "   Description: $description"
                fi
                if [ -n "$updated_at" ] && [ "$updated_at" != "null" ]; then
                    echo "   Updated: $updated_at"
                fi
            else
                echo "$key: [not found]"
            fi
            echo ""
        done
    else
        # Read all configs
        echo ""
        echo "All configuration:"
        echo ""
        
        FIRESTORE_URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/app_config"
        
        response=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
                       -H "Content-Type: application/json" \
                       "$FIRESTORE_URL")
        
        if echo "$response" | jq -e '.documents' > /dev/null 2>&1; then
            echo "$response" | jq -r '.documents[] | 
                (.fields.key.stringValue // (.name | split("/") | last)) as $key |
                (.fields.value | to_entries[0].value) as $value |
                (.fields.description.stringValue // empty) as $description |
                (.fields.updated_at.stringValue // empty) as $updated_at |
                "\($key): \($value)" +
                (if $description != "" and $description != null then "\n   Description: \($description)" else "" end) +
                (if $updated_at != "" and $updated_at != null then "\n   Updated: \($updated_at)" else "" end) +
                "\n"'
        else
            echo "No configurations found"
        fi
    fi
}

# Parse command line arguments
if [ $# -eq 0 ]; then
    show_usage
fi

COMMAND="$1"
shift

case "$COMMAND" in
    write)
        if [ $# -lt 2 ]; then
            echo "Error: write command requires key and value"
            echo "Usage: $0 write <key> <value> [description] -p PROJECT_ID -d DATABASE_NAME"
            exit 1
        fi
        KEY="$1"
        VALUE="$2"
        shift 2
        if [ $# -gt 0 ] && [[ ! "$1" =~ ^- ]]; then
            DESCRIPTION="$1"
            shift
        fi
        ;;
    read)
        # Collect all non-option arguments as keys
        KEYS=()
        while [ $# -gt 0 ] && [[ ! "$1" =~ ^- ]]; do
            KEYS+=("$1")
            shift
        done
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

echo "🚀 Configuration Management"
echo "Project ID: $PROJECT_ID"
echo "Database: $DATABASE"
echo ""

# Execute the command
case "$COMMAND" in
    write)
        write_config "$KEY" "$VALUE" "$DESCRIPTION"
        ;;
    read)
        if [ ${#KEYS[@]} -eq 0 ]; then
            read_config
        else
            read_config "${KEYS[@]}"
        fi
        ;;
esac

echo "✅ Operation completed!"
