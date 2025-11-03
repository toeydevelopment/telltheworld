#!/bin/bash

# Test Data Generator for Teller Notification Service
# This script generates various types of test notifications for load testing

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
API_BASE_URL="${API_URL:-http://localhost:8000}"
DEFAULT_COUNT=10
DEFAULT_DELAY=0.1
DEFAULT_TYPE="mixed"

# Parse command line arguments
COUNT=$DEFAULT_COUNT
DELAY=$DEFAULT_DELAY
TYPE=$DEFAULT_TYPE
CONCURRENT=false

show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Generate test notifications for the Teller service.

OPTIONS:
    -n, --count NUMBER      Number of notifications to generate (default: $DEFAULT_COUNT)
    -d, --delay SECONDS     Delay between notifications in seconds (default: $DEFAULT_DELAY)
    -t, --type TYPE         Type of notifications to generate (default: $DEFAULT_TYPE)
                           Options: single, bulk, email, push, mixed, scheduled, large
    -c, --concurrent        Send notifications concurrently
    -h, --help             Show this help message

EXAMPLES:
    # Generate 100 mixed notifications
    $0 -n 100 -t mixed

    # Generate 50 email notifications with 1 second delay
    $0 -n 50 -t email -d 1

    # Generate 1000 concurrent notifications
    $0 -n 1000 -c

    # Load test with 5000 notifications
    $0 -n 5000 -t mixed -c

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--count)
            COUNT="$2"
            shift 2
            ;;
        -d|--delay)
            DELAY="$2"
            shift 2
            ;;
        -t|--type)
            TYPE="$2"
            shift 2
            ;;
        -c|--concurrent)
            CONCURRENT=true
            shift
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

# Counters
SUCCESS_COUNT=0
FAILED_COUNT=0
START_TIME=$(date +%s)

# Function to generate random string
random_string() {
    local length=${1:-10}
    cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w $length | head -n 1
}

# Function to generate random email
generate_email() {
    echo "user_$(random_string 8)@example.com"
}

# Function to generate random device token
generate_device_token() {
    echo "device_$(random_string 32)"
}

# Function to generate random phone number
generate_phone() {
    echo "+1$(( RANDOM % 900 + 100 ))$(( RANDOM % 900 + 100 ))$(( RANDOM % 10000 ))"
}

# Function to generate lorem ipsum text
generate_lorem() {
    local words=${1:-10}
    local lorem="Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua"
    echo "$lorem" | cut -d' ' -f1-$words
}

# Function to send single notification
send_single_notification() {
    local index=$1
    local email=$(generate_email)
    local device=$(generate_device_token)
    local title="Test Notification #$index"
    local body="$(generate_lorem 15) - Generated at $(date '+%Y-%m-%d %H:%M:%S')"
    local ref_id="test_ref_$(random_string 10)"

    local response=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/send" \
        -H 'Content-Type: application/json' \
        -d '{
            "notification": {
                "title": "'"$title"'",
                "body": "'"$body"'",
                "ref_id": "'"$ref_id"'"
            },
            "destinations": [
                {"channel": 1, "destination": "'"$email"'"},
                {"channel": 2, "destination": "'"$device"'"}
            ]
        }' 2>/dev/null)

    if echo "$response" | grep -q "notificationId"; then
        return 0
    else
        return 1
    fi
}

# Function to send email-only notification
send_email_notification() {
    local index=$1
    local email=$(generate_email)
    local title="Email Test #$index"
    local body="This is an email-only notification. $(generate_lorem 20)"

    local response=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/send" \
        -H 'Content-Type: application/json' \
        -d '{
            "notification": {
                "title": "'"$title"'",
                "body": "'"$body"'",
                "ref_id": "email_test_'$index'"
            },
            "destinations": [
                {"channel": 1, "destination": "'"$email"'"}
            ]
        }' 2>/dev/null)

    if echo "$response" | grep -q "notificationId"; then
        return 0
    else
        return 1
    fi
}

# Function to send push-only notification
send_push_notification() {
    local index=$1
    local device=$(generate_device_token)
    local title="Push Test #$index"
    local body="Push notification: $(generate_lorem 10)"

    local response=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/send" \
        -H 'Content-Type: application/json' \
        -d '{
            "notification": {
                "title": "'"$title"'",
                "body": "'"$body"'",
                "ref_id": "push_test_'$index'"
            },
            "destinations": [
                {"channel": 2, "destination": "'"$device"'"}
            ]
        }' 2>/dev/null)

    if echo "$response" | grep -q "notificationId"; then
        return 0
    else
        return 1
    fi
}

# Function to send scheduled notification
send_scheduled_notification() {
    local index=$1
    local email=$(generate_email)
    local schedule_minutes=$((RANDOM % 60 + 1))
    local schedule_time=$(date -u -d "+$schedule_minutes minutes" '+%Y-%m-%dT%H:%M:%SZ')
    local title="Scheduled Test #$index"
    local body="This notification is scheduled for $schedule_minutes minutes from now"

    local response=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/send" \
        -H 'Content-Type: application/json' \
        -d '{
            "notification": {
                "title": "'"$title"'",
                "body": "'"$body"'",
                "ref_id": "scheduled_test_'$index'",
                "schedule_at": "'"$schedule_time"'"
            },
            "destinations": [
                {"channel": 1, "destination": "'"$email"'"}
            ]
        }' 2>/dev/null)

    if echo "$response" | grep -q "notificationId"; then
        return 0
    else
        return 1
    fi
}

# Function to send large notification with many destinations
send_large_notification() {
    local index=$1
    local destinations=""

    # Generate 10-20 destinations
    local dest_count=$((RANDOM % 11 + 10))
    for ((j=1; j<=dest_count; j++)); do
        if [ $j -gt 1 ]; then
            destinations="$destinations,"
        fi
        if [ $((j % 2)) -eq 0 ]; then
            destinations="$destinations{\"channel\": 1, \"destination\": \"$(generate_email)\"}"
        else
            destinations="$destinations{\"channel\": 2, \"destination\": \"$(generate_device_token)\"}"
        fi
    done

    local title="Large Batch Test #$index"
    local body="$(generate_lorem 50)"

    local response=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/send" \
        -H 'Content-Type: application/json' \
        -d '{
            "notification": {
                "title": "'"$title"'",
                "body": "'"$body"'",
                "ref_id": "large_test_'$index'"
            },
            "destinations": ['"$destinations"']
        }' 2>/dev/null)

    if echo "$response" | grep -q "notificationId"; then
        return 0
    else
        return 1
    fi
}

# Function to send bulk notifications
send_bulk_notifications() {
    local batch_size=5
    local requests=""

    for ((i=1; i<=batch_size; i++)); do
        if [ $i -gt 1 ]; then
            requests="$requests,"
        fi
        requests="$requests"'{
            "notification": {
                "title": "Bulk Item '$i'",
                "body": "Bulk notification body '$i'",
                "ref_id": "bulk_'$(random_string 10)'"
            },
            "destinations": [
                {"channel": 1, "destination": "'$(generate_email)'"},
                {"channel": 2, "destination": "'$(generate_device_token)'"}
            ]
        }'
    done

    local response=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/bulk-send" \
        -H 'Content-Type: application/json' \
        -d '{
            "requests": ['"$requests"']
        }' 2>/dev/null)

    if echo "$response" | grep -q "notificationIds"; then
        return 0
    else
        return 1
    fi
}

# Function to send mixed type notification
send_mixed_notification() {
    local index=$1
    local type_choice=$((RANDOM % 5))

    case $type_choice in
        0) send_single_notification $index ;;
        1) send_email_notification $index ;;
        2) send_push_notification $index ;;
        3) send_scheduled_notification $index ;;
        4) send_large_notification $index ;;
    esac
}

# Function to send notification based on type
send_notification() {
    local index=$1

    case $TYPE in
        single)
            send_single_notification $index
            ;;
        email)
            send_email_notification $index
            ;;
        push)
            send_push_notification $index
            ;;
        scheduled)
            send_scheduled_notification $index
            ;;
        large)
            send_large_notification $index
            ;;
        bulk)
            send_bulk_notifications
            ;;
        mixed)
            send_mixed_notification $index
            ;;
        *)
            echo -e "${RED}Unknown type: $TYPE${NC}"
            exit 1
            ;;
    esac
}

# Main execution
echo -e "${BLUE}=== Teller Notification Test Data Generator ===${NC}"
echo -e "${YELLOW}Configuration:${NC}"
echo -e "  API URL: $API_BASE_URL"
echo -e "  Count: $COUNT"
echo -e "  Type: $TYPE"
echo -e "  Delay: $DELAY seconds"
echo -e "  Concurrent: $CONCURRENT"
echo ""

# Check if API is available
echo -e "${YELLOW}Checking API availability...${NC}"
if ! curl -s -f "$API_BASE_URL/healthz" > /dev/null 2>&1; then
    echo -e "${RED}API is not available at $API_BASE_URL${NC}"
    echo -e "${YELLOW}Make sure the service is running${NC}"
    exit 1
fi
echo -e "${GREEN}API is available${NC}"
echo ""

echo -e "${YELLOW}Generating $COUNT notifications...${NC}"

if [ "$CONCURRENT" = true ]; then
    # Concurrent execution
    echo -e "${YELLOW}Sending notifications concurrently...${NC}"

    for ((i=1; i<=COUNT; i++)); do
        {
            if send_notification $i; then
                echo -n "."
            else
                echo -n "x"
            fi
        } &

        # Limit concurrent processes
        if [ $((i % 100)) -eq 0 ]; then
            wait
            echo " [$i/$COUNT]"
        fi
    done
    wait
    echo ""

    # Can't track individual success/failure in concurrent mode
    SUCCESS_COUNT=$COUNT
    FAILED_COUNT=0
else
    # Sequential execution
    for ((i=1; i<=COUNT; i++)); do
        printf "\r${YELLOW}Progress: $i/$COUNT${NC} "

        if send_notification $i; then
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
            printf "${GREEN}✓${NC}"
        else
            FAILED_COUNT=$((FAILED_COUNT + 1))
            printf "${RED}✗${NC}"
        fi

        if [ $i -lt $COUNT ] && [ "$DELAY" != "0" ]; then
            sleep $DELAY
        fi
    done
    echo ""
fi

# Calculate statistics
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
RATE=$(echo "scale=2; $COUNT / $DURATION" | bc 2>/dev/null || echo "N/A")

# Print summary
echo ""
echo -e "${BLUE}=== Generation Complete ===${NC}"
echo -e "${GREEN}Successfully generated: $SUCCESS_COUNT${NC}"
if [ $FAILED_COUNT -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED_COUNT${NC}"
fi
echo -e "Duration: ${DURATION} seconds"
echo -e "Rate: ${RATE} notifications/second"
echo ""

# Database statistics
echo -e "${BLUE}=== Quick Database Check ===${NC}"
if command -v psql > /dev/null 2>&1; then
    PGPASSWORD="${POSTGRES_PASSWORD:-password}" psql -h localhost -p 5432 -U postgres -d telltheworld -t -c "
        SELECT status, COUNT(*) as count
        FROM notifications
        WHERE created_at > NOW() - INTERVAL '5 minutes'
        GROUP BY status
        ORDER BY status
    " 2>/dev/null | while read line; do
        echo "  $line"
    done
else
    echo -e "${YELLOW}psql not found, skipping database check${NC}"
fi

echo ""
echo -e "${GREEN}Test data generation complete!${NC}"
echo -e "${YELLOW}You can monitor the processing using:${NC}"
echo -e "  ./scripts/docker-status.sh"
echo -e "  docker-compose logs -f teller-worker-1"

exit 0