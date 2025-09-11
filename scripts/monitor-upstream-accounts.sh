#!/bin/bash

# Script to analyze upstream account usage from hourly aggregates
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
            echo "  $0 --days 1                 # Show today's usage"
            echo "  $0 -d simple-relay-db-staging --days 1  # Check staging today"
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

echo -e "${BLUE}📊 Upstream Account Usage Summary (Last $DAYS_BACK days)${NC}"
echo "================================"
echo "Project: $PROJECT_ID"
echo "Database: $DATABASE"
echo "Date Range: $START_DATE to $END_DATE"
echo ""

# Get access token
echo "🔑 Getting access token..."
ACCESS_TOKEN=$(gcloud auth print-access-token)

# Query upstream_account_hourly_aggregates within date range
echo "📡 Analyzing upstream account usage for last $DAYS_BACK days..."
echo ""

# Get data within date range
QUERY_JSON=$(cat <<EOF
{
  "structuredQuery": {
    "from": [{"collectionId": "upstream_account_hourly_aggregates"}],
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

# Check if we have any data
if [[ $(echo "$RESPONSE" | jq '[.[] | select(.document)] | length') -eq 0 ]]; then
    echo -e "${YELLOW}No upstream account usage data found for the specified date range.${NC}"
    echo ""
    exit 0
fi

# Process data and group by date
echo "$RESPONSE" | jq -r '
  [.[] | select(.document) | .document.fields | {
    hour: .hour.timestampValue,
    account: .upstream_account_uuid.stringValue,
    requests: (.total_requests.integerValue // 0 | tonumber),
    points: ((.total_points.integerValue // .total_points.doubleValue // 0 | tonumber) / 1000000 | round),
    cost: (.total_cost.doubleValue // 0),
    input_tokens: (.total_input_tokens.integerValue // 0 | tonumber),
    output_tokens: (.total_output_tokens.integerValue // 0 | tonumber)
  }] |
  map({
    date: (.hour | split("T")[0]),
    hour_num: (.hour | split("T")[1] | split(":")[0]),
    account: .account,
    requests: .requests,
    points: .points,
    cost: .cost,
    input_tokens: .input_tokens,
    output_tokens: .output_tokens
  })
' > /tmp/upstream_processed_data.json

# Get unique dates
DATES=$(cat /tmp/upstream_processed_data.json | jq -r '.[].date' | sort -u)

# Process each date
GRAND_TOTAL_REQUESTS=0
GRAND_TOTAL_POINTS=0
GRAND_TOTAL_COST=0
GRAND_TOTAL_INPUT=0
GRAND_TOTAL_OUTPUT=0

for date in $DATES; do
    # Skip invalid dates
    if [[ ! "$date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
        continue
    fi
    
    # Display date header
    echo -e "${YELLOW}### $date${NC}"
    
    # Query for this specific date
    DATE_QUERY=$(cat <<EOF
{
  "structuredQuery": {
    "from": [{"collectionId": "upstream_account_hourly_aggregates"}],
    "where": {
      "compositeFilter": {
        "op": "AND",
        "filters": [
          {
            "fieldFilter": {
              "field": {"fieldPath": "hour"},
              "op": "GREATER_THAN_OR_EQUAL",
              "value": {"timestampValue": "${date}T00:00:00Z"}
            }
          },
          {
            "fieldFilter": {
              "field": {"fieldPath": "hour"},
              "op": "LESS_THAN",
              "value": {"timestampValue": "${date}T23:59:59Z"}
            }
          }
        ]
      }
    },
    "orderBy": [{"field": {"fieldPath": "upstream_account_uuid"}, "direction": "ASCENDING"}]
  }
}
EOF
)
    
    DATE_RESPONSE=$(curl -s -X POST \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$DATE_QUERY" \
        "https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents:runQuery" 2>/dev/null)
    
    # Process date data
    DATE_STATS=$(echo "$DATE_RESPONSE" | jq -r '
      [.[] | select(.document) | .document.fields | {
        account: .upstream_account_uuid.stringValue,
        requests: (.total_requests.integerValue // 0 | tonumber),
        points: ((.total_points.integerValue // .total_points.doubleValue // 0 | tonumber) / 1000000 | round),
        cost: (.total_cost.doubleValue // 0),
        input_tokens: (.total_input_tokens.integerValue // 0 | tonumber),
        output_tokens: (.total_output_tokens.integerValue // 0 | tonumber)
      }] |
      group_by(.account) |
      map({
        account: .[0].account,
        total_requests: (map(.requests) | add),
        total_points: (map(.points) | add),
        total_cost: (map(.cost) | add),
        total_input_tokens: (map(.input_tokens) | add),
        total_output_tokens: (map(.output_tokens) | add)
      }) |
      sort_by(-.total_cost)
    ')
    
    # Get totals for this date
    DATE_TOTAL_REQUESTS=$(echo "$DATE_STATS" | jq '[.[].total_requests] | add // 0')
    DATE_TOTAL_POINTS=$(echo "$DATE_STATS" | jq '[.[].total_points] | add // 0')
    DATE_TOTAL_COST=$(echo "$DATE_STATS" | jq '[.[].total_cost] | add // 0')
    DATE_TOTAL_INPUT=$(echo "$DATE_STATS" | jq '[.[].total_input_tokens] | add // 0')
    DATE_TOTAL_OUTPUT=$(echo "$DATE_STATS" | jq '[.[].total_output_tokens] | add // 0')
    
    GRAND_TOTAL_REQUESTS=$((GRAND_TOTAL_REQUESTS + DATE_TOTAL_REQUESTS))
    GRAND_TOTAL_POINTS=$((GRAND_TOTAL_POINTS + DATE_TOTAL_POINTS))
    GRAND_TOTAL_COST=$(echo "$GRAND_TOTAL_COST + $DATE_TOTAL_COST" | bc)
    GRAND_TOTAL_INPUT=$((GRAND_TOTAL_INPUT + DATE_TOTAL_INPUT))
    GRAND_TOTAL_OUTPUT=$((GRAND_TOTAL_OUTPUT + DATE_TOTAL_OUTPUT))
    
    # Display account stats
    if [[ $(echo "$DATE_STATS" | jq 'length') -gt 0 ]]; then
        echo "$DATE_STATS" | jq -r '.[] | 
          "- \(.account | .[0:16])...: " +
          "\(.total_requests | tostring | gsub("(?<=[0-9])(?=([0-9]{3})+$)"; ",")) reqs, " +
          "\(.total_points | tostring | gsub("(?<=[0-9])(?=([0-9]{3})+$)"; ",")) pts, " +
          "$\(.total_cost | . * 100 | round / 100), " +
          "\(.total_input_tokens | tostring | gsub("(?<=[0-9])(?=([0-9]{3})+$)"; ",")) in, " +
          "\(.total_output_tokens | tostring | gsub("(?<=[0-9])(?=([0-9]{3})+$)"; ",")) out"'
    else
        echo "- No usage"
    fi
    
    echo -e "${GREEN}Daily Total: $DATE_TOTAL_REQUESTS requests, $DATE_TOTAL_POINTS points, \$$(echo "$DATE_TOTAL_COST" | xargs printf "%.2f")${NC}"
    echo ""
done

# Display grand totals
echo "================================"
echo -e "${BLUE}📈 Summary (Last $DAYS_BACK days)${NC}"
echo -e "Total Requests: $(echo $GRAND_TOTAL_REQUESTS | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta')"
echo -e "Total Points: $(echo $GRAND_TOTAL_POINTS | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta')"
echo -e "Total Cost: \$$(echo "$GRAND_TOTAL_COST" | xargs printf "%.2f")"
echo -e "Total Input Tokens: $(echo $GRAND_TOTAL_INPUT | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta')"
echo -e "Total Output Tokens: $(echo $GRAND_TOTAL_OUTPUT | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta')"

# Calculate daily averages if more than 1 day
if [[ $DAYS_BACK -gt 1 ]]; then
    AVG_REQUESTS=$((GRAND_TOTAL_REQUESTS / DAYS_BACK))
    AVG_POINTS=$((GRAND_TOTAL_POINTS / DAYS_BACK))
    AVG_COST=$(echo "scale=2; $GRAND_TOTAL_COST / $DAYS_BACK" | bc)
    
    echo ""
    echo -e "${BLUE}📊 Daily Averages${NC}"
    echo -e "Avg Requests/Day: $(echo $AVG_REQUESTS | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta')"
    echo -e "Avg Points/Day: $(echo $AVG_POINTS | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta')"
    echo -e "Avg Cost/Day: \$$(echo "$AVG_COST" | xargs printf "%.2f")"
fi

# Check for rate-limited accounts
echo ""
echo -e "${YELLOW}🔍 Checking for Rate-Limited Accounts${NC}"

OAUTH_QUERY=$(cat <<EOF
{
  "structuredQuery": {
    "from": [{"collectionId": "oauth_tokens"}],
    "where": {
      "fieldFilter": {
        "field": {"fieldPath": "is_rate_limited"},
        "op": "EQUAL",
        "value": {"booleanValue": true}
      }
    }
  }
}
EOF
)

OAUTH_RESPONSE=$(curl -s -X POST \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$OAUTH_QUERY" \
    "https://firestore.googleapis.com/v1/projects/$PROJECT_ID/databases/$DATABASE/documents:runQuery" 2>/dev/null)

RATE_LIMITED=$(echo "$OAUTH_RESPONSE" | jq -r '
  [.[] | select(.document) | .document.fields | {
    account: (.account_uuid.stringValue // .document.name | split("/") | last),
    org: .organization_name.stringValue,
    limited_at: .rate_limited_at.timestampValue
  }]
')

if [[ $(echo "$RATE_LIMITED" | jq 'length') -gt 0 ]]; then
    echo -e "${RED}⚠️  Rate-limited accounts found:${NC}"
    echo "$RATE_LIMITED" | jq -r '.[] | "- \(.account | .[0:16])... (\(.org)): Limited at \(.limited_at)"'
else
    echo -e "${GREEN}✅ No rate-limited accounts${NC}"
fi

echo ""
echo "✅ Analysis complete!"

# Cleanup
rm -f /tmp/upstream_processed_data.json