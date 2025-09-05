#!/opt/homebrew/bin/bash

# Check if we have bash 4+ for associative arrays  
if [[ ${BASH_VERSION%%.*} -lt 4 ]]; then
    echo "❌ This script requires bash 4.0+ for associative arrays"
    echo "On macOS, install with: brew install bash"
    echo "Current version: $BASH_VERSION"
    exit 1
fi

# Pure shell billing consistency verification script
# Usage: ./verify-billing-consistency.sh -p PROJECT_ID -d DATABASE_NAME [-u USER_EMAIL] [-h HOUR] [-v]

set -e

# Default values
PROJECT_ID=""
DATABASE_NAME=""
USER_EMAIL=""
HOUR=""
VERBOSE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print usage
usage() {
    echo "Usage: $0 -p PROJECT_ID -d DATABASE_NAME [-u USER_EMAIL] [-h HOUR] [-v]"
    echo ""
    echo "Options:"
    echo "  -p PROJECT_ID    GCP project ID"
    echo "  -d DATABASE_NAME Firestore database name"  
    echo "  -u USER_EMAIL    Filter by specific user email (optional)"
    echo "  -h HOUR          Filter by specific hour YYYY-MM-DDTHH (optional)"
    echo "  -v               Verbose output"
    echo ""
    echo "Examples:"
    echo "  $0 -p simple-relay-468808 -d simple-relay-db-staging"
    echo "  $0 -p simple-relay-468808 -d simple-relay-db-staging -u user@example.com"
    echo "  $0 -p simple-relay-468808 -d simple-relay-db-staging -h 2025-09-05T01 -v"
    exit 1
}

# Parse command line arguments
while getopts "p:d:u:h:v" opt; do
    case $opt in
        p) PROJECT_ID="$OPTARG" ;;
        d) DATABASE_NAME="$OPTARG" ;;
        u) USER_EMAIL="$OPTARG" ;;
        h) HOUR="$OPTARG" ;;
        v) VERBOSE=true ;;
        *) usage ;;
    esac
done

# Check required parameters
if [[ -z "$PROJECT_ID" || -z "$DATABASE_NAME" ]]; then
    echo -e "${RED}Error: PROJECT_ID and DATABASE_NAME are required${NC}"
    usage
fi

echo -e "${BLUE}🔍 Verifying billing consistency...${NC}"
echo "Project: $PROJECT_ID"
echo "Database: $DATABASE_NAME"
[[ -n "$USER_EMAIL" ]] && echo "User filter: $USER_EMAIL"
[[ -n "$HOUR" ]] && echo "Hour filter: $HOUR"
echo ""

# Function to get Firestore access token
get_access_token() {
    gcloud auth print-access-token 2>/dev/null
}

# Function to query Firestore REST API
query_firestore() {
    local collection="$1"
    local access_token="$2"
    
    local url="https://firestore.googleapis.com/v1/projects/${PROJECT_ID}/databases/${DATABASE_NAME}/documents/${collection}"
    
    curl -s -H "Authorization: Bearer ${access_token}" "${url}" 2>/dev/null
}

# Function to extract field value from JSON using jq
get_field_value() {
    local json="$1"
    local field_path="$2"
    
    # Handle different Firestore field types
    echo "$json" | jq -r "
        if .fields.\"$field_path\".stringValue then .fields.\"$field_path\".stringValue
        elif .fields.\"$field_path\".integerValue then .fields.\"$field_path\".integerValue
        elif .fields.\"$field_path\".doubleValue then .fields.\"$field_path\".doubleValue
        elif .fields.\"$field_path\".timestampValue then .fields.\"$field_path\".timestampValue
        else null
        end" 2>/dev/null | sed 's/null//'
}

# Function to extract hour from timestamp
extract_hour() {
    local timestamp="$1"
    echo "$timestamp" | sed -n 's/^\([0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9]\{2\}\).*/\1/p'
}

echo -e "${YELLOW}📡 Getting access token...${NC}"
ACCESS_TOKEN=$(get_access_token)
if [[ -z "$ACCESS_TOKEN" ]]; then
    echo -e "${RED}❌ Failed to get access token. Please run: gcloud auth login${NC}"
    exit 1
fi

echo -e "${YELLOW}📡 Fetching usage records...${NC}"
USAGE_DATA=$(query_firestore "usage_records" "$ACCESS_TOKEN")
if [[ $? -ne 0 ]] || [[ -z "$USAGE_DATA" ]]; then
    echo -e "${RED}❌ Failed to fetch usage records${NC}"
    exit 1
fi

echo -e "${YELLOW}📡 Fetching hourly aggregates...${NC}"
HOURLY_DATA=$(query_firestore "hourly_aggregates" "$ACCESS_TOKEN")
if [[ $? -ne 0 ]] || [[ -z "$HOURLY_DATA" ]]; then
    echo -e "${RED}❌ Failed to fetch hourly aggregates${NC}"
    exit 1
fi

echo -e "${YELLOW}🧮 Analyzing consistency...${NC}"

# Create temporary files for analysis
TEMP_DIR=$(mktemp -d)
USAGE_ANALYSIS="$TEMP_DIR/usage_analysis.txt"
HOURLY_ANALYSIS="$TEMP_DIR/hourly_analysis.txt"
COMPARISON="$TEMP_DIR/comparison.txt"

# Cleanup function
cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

echo ""
echo "📊 Analyzing usage records..."

# Parse usage records and group by user+hour
declare -A usage_stats
declare -A usage_records_count
total_usage_docs=0
processed_usage=0

while read -r doc; do
    [[ -z "$doc" || "$doc" == "null" ]] && continue
    
    total_usage_docs=$((total_usage_docs + 1))
    
    user_id=$(echo "$doc" | jq -r '.fields.user_id.stringValue // empty' 2>/dev/null)
    timestamp=$(echo "$doc" | jq -r '.fields.timestamp.timestampValue // empty' 2>/dev/null)
    model=$(echo "$doc" | jq -r '.fields.model.stringValue // empty' 2>/dev/null)
    input_tokens=$(echo "$doc" | jq -r '.fields.input_tokens.integerValue // 0' 2>/dev/null)
    output_tokens=$(echo "$doc" | jq -r '.fields.output_tokens.integerValue // 0' 2>/dev/null)
    total_cost=$(echo "$doc" | jq -r '.fields.total_cost.doubleValue // 0' 2>/dev/null)
    
    [[ -z "$user_id" || -z "$timestamp" ]] && continue
    
    # Apply user filter
    [[ -n "$USER_EMAIL" && "$user_id" != "$USER_EMAIL" ]] && continue
    
    # Extract hour
    hour_key=$(extract_hour "$timestamp")
    [[ -z "$hour_key" ]] && continue
    
    # Apply hour filter
    [[ -n "$HOUR" && "$hour_key" != "$HOUR" ]] && continue
    
    # Create aggregate key
    agg_key="${user_id}_${hour_key}"
    
    processed_usage=$((processed_usage + 1))
    
    # Initialize if first time seeing this key
    if [[ -z "${usage_stats[$agg_key]}" ]]; then
        usage_stats[$agg_key]="0:0:0:0.0"  # requests:input:output:cost
        usage_records_count[$agg_key]=0
    fi
    
    # Parse current values
    IFS=':' read -r current_req current_input current_output current_cost <<< "${usage_stats[$agg_key]}"
    
    # Accumulate values
    new_req=$((current_req + 1))
    new_input=$((current_input + input_tokens))
    new_output=$((current_output + output_tokens))
    new_cost=$(echo "$current_cost + $total_cost" | bc -l 2>/dev/null | xargs printf "%.6f")
    
    usage_stats[$agg_key]="${new_req}:${new_input}:${new_output}:${new_cost}"
    usage_records_count[$agg_key]=$((${usage_records_count[$agg_key]} + 1))
    
done < <(echo "$USAGE_DATA" | jq -c '.documents[]?' 2>/dev/null)

echo "   Total usage documents: $total_usage_docs"
echo "   Processed (after filters): $processed_usage"
echo "   Grouped into: ${#usage_stats[@]} hourly buckets"

echo ""
echo "📊 Analyzing hourly aggregates..."

# Parse hourly aggregates
declare -A hourly_stats
total_hourly_docs=0
processed_hourly=0

while read -r doc; do
    [[ -z "$doc" || "$doc" == "null" ]] && continue
    
    total_hourly_docs=$((total_hourly_docs + 1))
    
    user_id=$(echo "$doc" | jq -r '.fields.user_id.stringValue // empty' 2>/dev/null)
    hour_timestamp=$(echo "$doc" | jq -r '.fields.hour.timestampValue // empty' 2>/dev/null)
    total_requests=$(echo "$doc" | jq -r '.fields.total_requests.integerValue // 0' 2>/dev/null)
    total_input_tokens=$(echo "$doc" | jq -r '.fields.total_input_tokens.integerValue // 0' 2>/dev/null)
    total_output_tokens=$(echo "$doc" | jq -r '.fields.total_output_tokens.integerValue // 0' 2>/dev/null)
    total_cost=$(echo "$doc" | jq -r '.fields.total_cost.doubleValue // 0' 2>/dev/null)
    
    [[ -z "$user_id" || -z "$hour_timestamp" ]] && continue
    
    # Apply user filter
    [[ -n "$USER_EMAIL" && "$user_id" != "$USER_EMAIL" ]] && continue
    
    # Extract hour
    hour_key=$(extract_hour "$hour_timestamp")
    [[ -z "$hour_key" ]] && continue
    
    # Apply hour filter
    [[ -n "$HOUR" && "$hour_key" != "$HOUR" ]] && continue
    
    # Create aggregate key
    agg_key="${user_id}_${hour_key}"
    
    processed_hourly=$((processed_hourly + 1))
    
    hourly_stats[$agg_key]="${total_requests}:${total_input_tokens}:${total_output_tokens}:${total_cost}"
    
done < <(echo "$HOURLY_DATA" | jq -c '.documents[]?' 2>/dev/null)

echo "   Total hourly aggregate documents: $total_hourly_docs"
echo "   Processed (after filters): $processed_hourly"

echo ""
echo "================================================================================"
echo "📊 BILLING CONSISTENCY REPORT"
echo "================================================================================"

# Compare data
consistent_count=0
inconsistent_count=0
missing_count=0
extra_count=0

# Get all unique keys
all_keys=($(printf '%s\n' "${!usage_stats[@]}" "${!hourly_stats[@]}" | sort -u))

for agg_key in "${all_keys[@]}"; do
    usage_data="${usage_stats[$agg_key]}"
    hourly_data="${hourly_stats[$agg_key]}"
    
    echo ""
    echo "🔍 $agg_key"
    echo "------------------------------------------------------------"
    
    if [[ -z "$usage_data" ]]; then
        echo "❌ EXTRA AGGREGATE: Found in hourly_aggregates but no matching usage_records"
        extra_count=$((extra_count + 1))
        if [[ "$VERBOSE" == true ]]; then
            IFS=':' read -r h_req h_input h_output h_cost <<< "$hourly_data"
            echo "   Stored: $h_req req, $h_input in, $h_output out, \$$h_cost"
        fi
        continue
    fi
    
    if [[ -z "$hourly_data" ]]; then
        echo "❌ MISSING AGGREGATE: Found usage_records but no hourly_aggregates"
        missing_count=$((missing_count + 1))
        if [[ "$VERBOSE" == true ]]; then
            IFS=':' read -r u_req u_input u_output u_cost <<< "$usage_data"
            echo "   Expected: $u_req req, $u_input in, $u_output out, \$$u_cost"
            echo "   Based on ${usage_records_count[$agg_key]} raw records"
        fi
        continue
    fi
    
    # Compare the data
    IFS=':' read -r u_req u_input u_output u_cost <<< "$usage_data"
    IFS=':' read -r h_req h_input h_output h_cost <<< "$hourly_data"
    
    issues=()
    
    [[ "$u_req" != "$h_req" ]] && issues+=("requests: $u_req ≠ $h_req")
    [[ "$u_input" != "$h_input" ]] && issues+=("input_tokens: $u_input ≠ $h_input")
    [[ "$u_output" != "$h_output" ]] && issues+=("output_tokens: $u_output ≠ $h_output")
    
    # Compare costs with tolerance
    cost_diff=$(echo "if ($u_cost - $h_cost < 0) $h_cost - $u_cost else $u_cost - $h_cost" | bc -l 2>/dev/null | xargs printf "%.6f")
    cost_threshold="0.000001"
    if (( $(echo "$cost_diff > $cost_threshold" | bc -l 2>/dev/null) )); then
        issues+=("total_cost: \$$u_cost ≠ \$$h_cost (diff: \$$cost_diff)")
    fi
    
    if [[ ${#issues[@]} -gt 0 ]]; then
        echo "❌ INCONSISTENT (${#issues[@]} issues)"
        for issue in "${issues[@]}"; do
            echo "   • $issue"
        done
        inconsistent_count=$((inconsistent_count + 1))
        
        if [[ "$VERBOSE" == true ]]; then
            echo "   📋 Raw records: ${usage_records_count[$agg_key]}"
        fi
    else
        echo "✅ CONSISTENT"
        consistent_count=$((consistent_count + 1))
        
        if [[ "$VERBOSE" == true ]]; then
            echo "   Total: $h_req req, $h_input in, $h_output out, \$$h_cost"
            echo "   Raw records: ${usage_records_count[$agg_key]}"
        fi
    fi
done

echo ""
echo "================================================================================"
echo "📈 FINAL SUMMARY"
echo "================================================================================"
echo "✅ Consistent aggregates: $consistent_count"
echo "❌ Inconsistent aggregates: $inconsistent_count"
echo "❌ Missing aggregates: $missing_count"
echo "❌ Extra aggregates: $extra_count"
echo "📊 Total aggregates analyzed: ${#all_keys[@]}"

total_issues=$((inconsistent_count + missing_count + extra_count))

echo ""
if [[ $total_issues -eq 0 ]]; then
    echo -e "${GREEN}🎉 ALL BILLING DATA IS CONSISTENT! 🎉${NC}"
    echo -e "${GREEN}✅ Hourly aggregation is working perfectly${NC}"
    exit_code=0
else
    echo -e "${RED}⚠️  Found $total_issues consistency issues${NC}"
    if [[ $missing_count -gt 0 ]]; then
        echo -e "${YELLOW}💡 Missing aggregates may be from before hourly aggregation was implemented${NC}"
    fi
    exit_code=1
fi

exit $exit_code