#!/bin/bash

# Script to analyze upstream account usage from minute aggregates
# Shows minute-level granular data for monitoring and debugging

set -e

# Default values
PROJECT_ID="simple-relay-468808"
DATABASE="simple-relay-db-production"
HOURS_BACK=1

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse command line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -p|--project) PROJECT_ID="$2"; shift ;;
        -d|--database) DATABASE="$2"; shift ;;
        --hours) HOURS_BACK="$2"; shift ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -p, --project PROJECT_ID    GCP project ID (default: simple-relay-468808)"
            echo "  -d, --database DATABASE     Database name (default: simple-relay-db-production)"
            echo "  --hours HOURS               Number of hours back to query (default: 1)"
            echo "  -h, --help                 Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                          # Show last 1 hour"
            echo "  $0 --hours 6                # Show last 6 hours"
            echo "  $0 --hours 24               # Show last 24 hours"
            echo "  $0 -d simple-relay-db-staging --hours 2  # Check staging last 2 hours"
            exit 0
            ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Calculate date range
get_datetime_hours_ago() {
    local hours_ago="$1"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date -j -v-${hours_ago}H "+%Y-%m-%dT%H:%M"
    else
        date -d "${hours_ago} hours ago" "+%Y-%m-%dT%H:%M"
    fi
}

get_current_datetime() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date "+%Y-%m-%dT%H:%M"
    else
        date "+%Y-%m-%dT%H:%M"
    fi
}

START_DATETIME=$(get_datetime_hours_ago $HOURS_BACK)
END_DATETIME=$(get_current_datetime)

echo -e "${BLUE}📊 Upstream Account Minute Usage Summary (Last $HOURS_BACK hours)${NC}"
echo "================================"
echo "Project: $PROJECT_ID"
echo "Database: $DATABASE"
echo "Time Range: $START_DATETIME to $END_DATETIME"
echo ""

# Function to get OAuth token
get_oauth_token() {
    gcloud auth application-default print-access-token
}

echo "🔑 Getting access token..."
ACCESS_TOKEN=$(get_oauth_token)

echo "📡 Analyzing upstream account minute usage for last $HOURS_BACK hours..."

# Query Firestore for minute aggregates in the time range
URL="https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents/upstream_account_minute_aggregates"

# Fetch all minute aggregate documents
response=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" "$URL")

# Parse JSON and filter by time range, then aggregate by date-hour for display
echo "$response" | jq -r --arg start_time "$START_DATETIME" --arg end_time "$END_DATETIME" '
.documents[]? | 
select(.fields.minute.timestampValue) |
{
  name: .name,
  minute: .fields.minute.timestampValue,
  upstream_account_uuid: .fields.upstream_account_uuid.stringValue,
  total_requests: (.fields.total_requests.integerValue // "0" | tonumber),
  total_cost: (.fields.total_cost.doubleValue // 0),
  total_points: (.fields.total_points.doubleValue // .fields.total_points.integerValue // 0),
  total_input_tokens: (.fields.total_input_tokens.integerValue // "0" | tonumber),
  total_output_tokens: (.fields.total_output_tokens.integerValue // "0" | tonumber),
  total_cache_read_tokens: (.fields.total_cache_read_tokens.integerValue // "0" | tonumber),
  total_cache_write_tokens: (.fields.total_cache_write_tokens.integerValue // "0" | tonumber)
} |
select(.minute >= $start_time and .minute <= $end_time) |
[
  .minute[:16], # YYYY-MM-DDTHH:MM
  .upstream_account_uuid[:16] + "...",
  .total_requests,
  .total_points,
  ("$" + (.total_cost | tostring)),
  .total_input_tokens,
  .total_output_tokens,
  .total_cache_read_tokens,
  .total_cache_write_tokens
] | @tsv' | \
sort | \
awk -F'\t' '
BEGIN {
    print "MINUTE               UPSTREAM_ACCOUNT      REQS   POINTS   COST      INPUT    OUTPUT   C_READ   C_WRITE"
    print "==================== ================= ======== ====== ========== ======== ======== ======== ========"
    total_requests = 0
    total_points = 0
    total_cost = 0
    total_input = 0
    total_output = 0
    total_cache_read = 0
    total_cache_write = 0
}
{
    printf "%-20s %-17s %8s %6s %10s %8s %8s %8s %8s\n", $1, $2, $3, $4, $5, $6, $7, $8, $9
    total_requests += $3
    total_points += $4
    # Extract cost value without $
    cost_val = $5
    gsub(/\$/, "", cost_val)
    total_cost += cost_val
    total_input += $6
    total_output += $7
    total_cache_read += $8
    total_cache_write += $9
}
END {
    print "==================== ================= ======== ====== ========== ======== ======== ======== ========"
    printf "%-20s %-17s %8d %6.0f %10s %8d %8d %8d %8d\n", "TOTAL", "", total_requests, total_points, ("$" total_cost), total_input, total_output, total_cache_read, total_cache_write
}'

echo ""
echo "================================"
echo -e "${BLUE}📈 Summary (Last $HOURS_BACK hours)${NC}"

# Calculate totals from the filtered data
TOTALS=$(echo "$response" | jq -r --arg start_time "$START_DATETIME" --arg end_time "$END_DATETIME" '
.documents[]? | 
select(.fields.minute.timestampValue) |
{
  minute: .fields.minute.timestampValue,
  total_requests: (.fields.total_requests.integerValue // "0" | tonumber),
  total_cost: (.fields.total_cost.doubleValue // 0),
  total_points: (.fields.total_points.doubleValue // .fields.total_points.integerValue // 0),
  total_input_tokens: (.fields.total_input_tokens.integerValue // "0" | tonumber),
  total_output_tokens: (.fields.total_output_tokens.integerValue // "0" | tonumber)
} |
select(.minute >= $start_time and .minute <= $end_time) |
[.total_requests, .total_cost, .total_points, .total_input_tokens, .total_output_tokens] | @tsv' | \
awk -F'\t' '
{
    total_requests += $1
    total_cost += $2
    total_points += $3
    total_input += $4
    total_output += $5
}
END {
    printf "%d\t%.2f\t%.0f\t%d\t%d\n", total_requests, total_cost, total_points, total_input, total_output
}')

if [ -n "$TOTALS" ]; then
    IFS=$'\t' read -r total_requests total_cost total_points total_input total_output <<< "$TOTALS"
    echo "Total Requests: ${total_requests:-0}"
    echo "Total Points: ${total_points:-0}"
    echo "Total Cost: \$${total_cost:-0.00}"
    echo "Total Input Tokens: ${total_input:-0}"
    echo "Total Output Tokens: ${total_output:-0}"
else
    echo "No data found for the specified time range"
fi

echo ""
echo "✅ Analysis complete!"