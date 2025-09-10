#!/bin/bash

# Script to analyze user points consumption from hourly aggregates
# Shows complete summary for all dates with data

set -e

# Default values
PROJECT_ID="simple-relay-468808"
DATABASE="simple-relay-db-production"
DAYS_BACK=7

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
        --days) DAYS_BACK="$2"; shift ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -p, --project PROJECT_ID    GCP project ID (default: simple-relay-468808)"
            echo "  -d, --database DATABASE     Database name (default: simple-relay-db-production)"
            echo "  --days DAYS                 Number of days back to query (default: 7)"
            echo "  -h, --help                 Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                          # Show last 7 days"
            echo "  $0 --days 30                # Show last 30 days"
            echo "  $0 --days 1                 # Show last 1 day"
            exit 0
            ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Calculate date range
get_date_days_ago() {
    local days_ago="$1"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date -j -v-${days_ago}d "+%Y-%m-%d"
    else
        date -d "$days_ago days ago" "+%Y-%m-%d"
    fi
}

START_DATE=$(get_date_days_ago $DAYS_BACK)
END_DATE=$(date "+%Y-%m-%d")

echo -e "${BLUE}📊 Complete Points Usage Summary (Last $DAYS_BACK days)${NC}"
echo "================================"
echo "Project: $PROJECT_ID"
echo "Database: $DATABASE"
echo "Date Range: $START_DATE to $END_DATE"
echo ""

# Get access token
echo "🔑 Getting access token..."
ACCESS_TOKEN=$(gcloud auth print-access-token)

# Query hourly_aggregates within date range
echo "📡 Analyzing usage data for last $DAYS_BACK days..."
echo ""

# Get data within date range
QUERY_JSON=$(cat <<EOF
{
  "structuredQuery": {
    "from": [{"collectionId": "hourly_aggregates"}],
    "where": {
      "compositeFilter": {
        "op": "AND",
        "filters": [
          {
            "fieldFilter": {
              "field": {"fieldPath": "hour"},
              "op": "GREATER_THAN_OR_EQUAL",
              "value": {"timestampValue": "${START_DATE}T00:00:00Z"}
            }
          },
          {
            "fieldFilter": {
              "field": {"fieldPath": "hour"},
              "op": "LESS_THAN",
              "value": {"timestampValue": "${END_DATE}T23:59:59Z"}
            }
          }
        ]
      }
    },
    "orderBy": [{"field": {"fieldPath": "hour"}, "direction": "ASCENDING"}]
  }
}
EOF
)

RESPONSE=$(curl -s -X POST \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$QUERY_JSON" \
    "https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents:runQuery")

# Process data and group by window dates
echo "$RESPONSE" | jq -r '
  [.[] | select(.document) | .document.fields | {
    hour: .hour.timestampValue,
    user: .user_id.stringValue,
    points: (.total_points.integerValue // .total_points.doubleValue // 0 | tonumber)
  }] |
  map({
    date: (.hour | split("T")[0]),
    hour_num: (.hour | split("T")[1] | split(":")[0] | tonumber),
    user: .user,
    points: .points
  }) |
  map({
    window_date: (
      if .hour_num >= 20 then
        .date
      else
        # Need to calculate previous date - for simplicity, we group by actual date
        (.date | split("-") | {
          year: .[0],
          month: .[1],
          day: (.[2] | tonumber - 1 | tostring | if length == 1 then "0" + . else . end)
        } | .year + "-" + .month + "-" + .day)
      end
    ),
    user: .user,
    points: .points
  })
' > /tmp/processed_data.json

# Get unique window dates
WINDOW_DATES=$(cat /tmp/processed_data.json | jq -r '.[].window_date' | sort -u)

# Function to get next date
get_next_date() {
    local date="$1"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        date -j -f "%Y-%m-%d" -v+1d "$date" "+%Y-%m-%d"
    else
        date -d "$date +1 day" "+%Y-%m-%d"
    fi
}

# Filter window dates to be within the date range
is_date_in_range() {
    local date="$1"
    # Convert dates to numbers for comparison (YYYYMMDD format)
    local date_num=$(echo "$date" | tr -d '-')
    local start_num=$(echo "$START_DATE" | tr -d '-')
    local end_num=$(echo "$END_DATE" | tr -d '-')
    [[ "$date_num" -ge "$start_num" && "$date_num" -le "$end_num" ]]
}

# Process each window date
GRAND_TOTAL=0
for window_date in $WINDOW_DATES; do
    # Skip invalid dates
    if [[ ! "$window_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
        continue
    fi
    
    # Skip dates outside our range
    if ! is_date_in_range "$window_date"; then
        continue
    fi
    
    next_date=$(get_next_date "$window_date")
    
    # Query for this specific window
    WINDOW_QUERY=$(cat <<EOF
{
  "structuredQuery": {
    "from": [{"collectionId": "hourly_aggregates"}],
    "where": {
      "compositeFilter": {
        "op": "AND",
        "filters": [
          {
            "fieldFilter": {
              "field": {"fieldPath": "hour"},
              "op": "GREATER_THAN_OR_EQUAL",
              "value": {"timestampValue": "${window_date}T20:00:00Z"}
            }
          },
          {
            "fieldFilter": {
              "field": {"fieldPath": "hour"},
              "op": "LESS_THAN",
              "value": {"timestampValue": "${next_date}T20:00:00Z"}
            }
          }
        ]
      }
    },
    "orderBy": [{"field": {"fieldPath": "user_id"}, "direction": "ASCENDING"}]
  }
}
EOF
)
    
    WINDOW_RESPONSE=$(curl -s -X POST \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$WINDOW_QUERY" \
        "https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents:runQuery" 2>/dev/null)
    
    # Process window data
    WINDOW_STATS=$(echo "$WINDOW_RESPONSE" | jq -r '
      [.[] | select(.document) | .document.fields | {
        user: .user_id.stringValue,
        points: (.total_points.integerValue // .total_points.doubleValue // 0 | tonumber)
      }] |
      group_by(.user) |
      map({
        user: .[0].user,
        total_points: (map(.points) | add)
      }) |
      sort_by(-.total_points)
    ')
    
    # Get total for this window
    WINDOW_TOTAL=$(echo "$WINDOW_STATS" | jq '[.[].total_points] | add // 0')
    GRAND_TOTAL=$((GRAND_TOTAL + WINDOW_TOTAL))
    
    # Display window header
    echo -e "${YELLOW}### ${window_date} to ${next_date} (8pm UTC window)${NC}"
    
    # Display user stats
    if [[ $(echo "$WINDOW_STATS" | jq 'length') -gt 0 ]]; then
        echo "$WINDOW_STATS" | jq -r '.[] | "- \(.user): \(.total_points | tostring | gsub("(?<=[0-9])(?=([0-9]{3})+$)"; ",")) points"'
    else
        echo "- No usage"
    fi
    
    echo -e "${GREEN}Total: ${WINDOW_TOTAL} points${NC}"
    echo ""
done

# Only display grand total for single day queries
if [[ $DAYS_BACK -eq 1 ]]; then
    echo "================================"
    echo -e "${BLUE}Grand Total: ${GRAND_TOTAL} points${NC}"
fi
echo ""
echo "✅ Analysis complete!"

# Cleanup
rm -f /tmp/processed_data.json