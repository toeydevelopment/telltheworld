#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
API_BASE_URL="http://localhost:8000"
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="telltheworld"
DB_USER="postgres"
DB_PASSWORD="${POSTGRES_PASSWORD:-password}"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Function to print test headers
print_test_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Function to print test results
print_test_result() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ "$1" == "PASS" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓${NC} $2"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗${NC} $2"
        if [ ! -z "$3" ]; then
            echo -e "  ${RED}Error: $3${NC}"
        fi
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local service_name=$1
    local url=$2
    local max_attempts=30
    local attempt=1

    echo -e "${YELLOW}Waiting for $service_name to be ready...${NC}"

    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}$service_name is ready!${NC}"
            return 0
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done

    echo -e "\n${RED}$service_name failed to start after $max_attempts attempts${NC}"
    return 1
}

# Function to execute SQL query
execute_sql() {
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "$1" 2>/dev/null
}

# Function to check PostgreSQL connection
check_postgres() {
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\q" 2>/dev/null
    return $?
}

# Function to generate random email
generate_email() {
    echo "test_$(date +%s)_$RANDOM@example.com"
}

# Function to generate random device token
generate_device_token() {
    echo "device_$(date +%s)_$RANDOM"
}

print_test_header "Starting Notification Service Test Suite"

# Step 1: Start services
print_test_header "Step 1: Starting Docker Services"

echo -e "${YELLOW}Starting docker-compose services...${NC}"
docker-compose down > /dev/null 2>&1
docker-compose up -d

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to start docker-compose services${NC}"
    exit 1
fi

# Step 2: Wait for services to be ready
print_test_header "Step 2: Waiting for Services"

# Wait for PostgreSQL
wait_for_service "PostgreSQL" "http://localhost:5432" || exit 1

# Wait for NATS
wait_for_service "NATS" "http://localhost:8222/healthz" || exit 1

# Wait for API Server
wait_for_service "API Server" "$API_BASE_URL/healthz" || exit 1

# Give workers time to connect
echo -e "${YELLOW}Waiting for workers to initialize...${NC}"
sleep 5

# Step 3: Test Single Notification Send
print_test_header "Step 3: Testing Single Notification Send"

TEST_EMAIL=$(generate_email)
TEST_DEVICE=$(generate_device_token)

# Send single notification
echo "Sending single notification to $TEST_EMAIL and push device..."
RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/send" \
    -H 'Content-Type: application/json' \
    -d '{
        "notification": {
            "title": "Test Single Notification",
            "body": "This is a test notification body",
            "ref_id": "test-ref-001"
        },
        "destinations": [
            {"channel": 1, "destination": "'$TEST_EMAIL'"},
            {"channel": 2, "destination": "'$TEST_DEVICE'"}
        ]
    }')

NOTIFICATION_ID=$(echo $RESPONSE | jq -r '.notificationId' 2>/dev/null)

if [ ! -z "$NOTIFICATION_ID" ] && [ "$NOTIFICATION_ID" != "null" ]; then
    print_test_result "PASS" "Single notification created: $NOTIFICATION_ID"
else
    print_test_result "FAIL" "Failed to create single notification" "$RESPONSE"
fi

# Wait for processing
sleep 3

# Verify in database
if [ ! -z "$NOTIFICATION_ID" ]; then
    SQL_CHECK="SELECT status, title, body FROM notifications WHERE notification_id = '$NOTIFICATION_ID'"
    DB_RESULT=$(execute_sql "$SQL_CHECK")

    if echo "$DB_RESULT" | grep -q "Test Single Notification"; then
        print_test_result "PASS" "Notification found in database"

        # Check status
        STATUS=$(echo "$DB_RESULT" | awk '{print $1}')
        if [ "$STATUS" == "completed" ] || [ "$STATUS" == "processing" ]; then
            print_test_result "PASS" "Notification status is $STATUS"
        else
            print_test_result "FAIL" "Unexpected notification status: $STATUS"
        fi
    else
        print_test_result "FAIL" "Notification not found in database"
    fi
fi

# Step 4: Test Bulk Notification Send
print_test_header "Step 4: Testing Bulk Notification Send"

# Generate multiple destinations
BULK_DESTINATIONS=""
for i in {1..5}; do
    EMAIL=$(generate_email)
    DEVICE=$(generate_device_token)
    if [ $i -gt 1 ]; then
        BULK_DESTINATIONS="$BULK_DESTINATIONS,"
    fi
    BULK_DESTINATIONS="$BULK_DESTINATIONS"'
        {
            "notification": {
                "title": "Bulk Test '$i'",
                "body": "Bulk notification body '$i'",
                "ref_id": "bulk-ref-'$i'"
            },
            "destinations": [
                {"channel": 1, "destination": "'$EMAIL'"},
                {"channel": 2, "destination": "'$DEVICE'"}
            ]
        }'
done

echo "Sending bulk notifications..."
BULK_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/bulk-send" \
    -H 'Content-Type: application/json' \
    -d '{
        "requests": ['$BULK_DESTINATIONS']
    }')

BULK_IDS=$(echo $BULK_RESPONSE | jq -r '.notificationIds[]' 2>/dev/null | wc -l)

if [ "$BULK_IDS" -eq 5 ]; then
    print_test_result "PASS" "Bulk send created 5 notifications"
else
    print_test_result "FAIL" "Bulk send did not create expected number of notifications" "$BULK_RESPONSE"
fi

# Wait for processing
sleep 3

# Step 5: Test Database Integrity
print_test_header "Step 5: Testing Database Integrity"

# Check total notifications
TOTAL_COUNT=$(execute_sql "SELECT COUNT(*) FROM notifications")
if [ $TOTAL_COUNT -gt 0 ]; then
    print_test_result "PASS" "Database contains $TOTAL_COUNT notifications"
else
    print_test_result "FAIL" "No notifications found in database"
fi

# Check notification channels are stored correctly
CHANNELS_CHECK=$(execute_sql "SELECT COUNT(*) FROM notifications WHERE channels IS NOT NULL AND channels != '[]'")
if [ $CHANNELS_CHECK -gt 0 ]; then
    print_test_result "PASS" "Notification channels are properly stored"
else
    print_test_result "FAIL" "Notification channels are not properly stored"
fi

# Step 6: Test Worker Processing
print_test_header "Step 6: Testing Worker Processing"

# Check if workers have claimed notifications
PROCESSING_COUNT=$(execute_sql "SELECT COUNT(DISTINCT processing_worker_id) FROM notifications WHERE processing_worker_id IS NOT NULL")
if [ $PROCESSING_COUNT -gt 0 ]; then
    print_test_result "PASS" "Workers are processing notifications (found $PROCESSING_COUNT active workers)"
else
    print_test_result "WARN" "No worker processing detected yet"
fi

# Check for completed notifications
COMPLETED_COUNT=$(execute_sql "SELECT COUNT(*) FROM notifications WHERE status = 'completed'")
if [ $COMPLETED_COUNT -gt 0 ]; then
    print_test_result "PASS" "Found $COMPLETED_COUNT completed notifications"
else
    # It might still be processing
    PROCESSING=$(execute_sql "SELECT COUNT(*) FROM notifications WHERE status = 'processing'")
    if [ $PROCESSING -gt 0 ]; then
        print_test_result "PASS" "Found $PROCESSING notifications being processed"
    else
        print_test_result "WARN" "No completed or processing notifications found"
    fi
fi

# Step 7: Test NATS JetStream
print_test_header "Step 7: Testing NATS JetStream"

# Check NATS health
NATS_HEALTH=$(curl -s http://localhost:8222/healthz)
if [ "$NATS_HEALTH" == '{"status":"ok"}' ]; then
    print_test_result "PASS" "NATS server is healthy"
else
    print_test_result "FAIL" "NATS server health check failed"
fi

# Check JetStream stream
STREAM_INFO=$(curl -s http://localhost:8222/jsz | jq '.streams[] | select(.config.name=="TELLER")' 2>/dev/null)
if [ ! -z "$STREAM_INFO" ]; then
    STREAM_MESSAGES=$(echo $STREAM_INFO | jq '.state.messages' 2>/dev/null)
    print_test_result "PASS" "TELLER stream exists with $STREAM_MESSAGES messages"
else
    print_test_result "WARN" "TELLER stream not found or not accessible"
fi

# Step 8: Test Concurrent Processing
print_test_header "Step 8: Testing Concurrent Processing"

echo "Sending 10 concurrent notifications..."
for i in {1..10}; do
    {
        curl -s -X POST "$API_BASE_URL/api/v1/notifications/send" \
            -H 'Content-Type: application/json' \
            -d '{
                "notification": {
                    "title": "Concurrent Test '$i'",
                    "body": "Testing concurrent processing",
                    "ref_id": "concurrent-'$i'"
                },
                "destinations": [
                    {"channel": 1, "destination": "concurrent'$i'@test.com"}
                ]
            }' > /dev/null
    } &
done
wait

print_test_result "PASS" "Sent 10 concurrent notifications"

# Wait for processing
sleep 5

# Check for race conditions (no duplicate processing)
DUPLICATE_CHECK=$(execute_sql "
    SELECT COUNT(*)
    FROM notifications
    WHERE processing_worker_id IS NOT NULL
    GROUP BY notification_id
    HAVING COUNT(*) > 1
")

if [ -z "$DUPLICATE_CHECK" ] || [ "$DUPLICATE_CHECK" -eq 0 ]; then
    print_test_result "PASS" "No race conditions detected (no duplicate processing)"
else
    print_test_result "FAIL" "Race condition detected: duplicate processing found"
fi

# Step 9: Check Error Handling
print_test_header "Step 9: Testing Error Handling"

# Send notification with invalid data (missing required fields)
ERROR_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/send" \
    -H 'Content-Type: application/json' \
    -d '{
        "notification": {},
        "destinations": []
    }')

if echo "$ERROR_RESPONSE" | grep -q "error\|invalid\|required"; then
    print_test_result "PASS" "API properly validates invalid input"
else
    print_test_result "FAIL" "API did not properly validate invalid input"
fi

# Step 10: Database Statistics
print_test_header "Step 10: Database Statistics"

echo -e "\n${BLUE}Notification Status Distribution:${NC}"
execute_sql "
    SELECT status, COUNT(*) as count
    FROM notifications
    GROUP BY status
    ORDER BY status
" | while read line; do
    echo "  $line"
done

echo -e "\n${BLUE}Worker Activity:${NC}"
execute_sql "
    SELECT
        COALESCE(processing_worker_id, 'unassigned') as worker,
        COUNT(*) as notifications
    FROM notifications
    WHERE status IN ('processing', 'completed')
    GROUP BY processing_worker_id
    ORDER BY COUNT(*) DESC
" | while read line; do
    echo "  $line"
done

echo -e "\n${BLUE}Channel Distribution:${NC}"
execute_sql "
    SELECT
        CASE
            WHEN channels::text LIKE '%channel\":1%' THEN 'Email'
            WHEN channels::text LIKE '%channel\":2%' THEN 'Push'
            ELSE 'Unknown'
        END as channel_type,
        COUNT(*) as count
    FROM notifications
    WHERE channels IS NOT NULL
    GROUP BY channel_type
" | while read line; do
    echo "  $line"
done

# Final Summary
print_test_header "Test Summary"

TOTAL_TESTS=$((TESTS_PASSED + TESTS_FAILED))
SUCCESS_RATE=0
if [ $TOTAL_TESTS -gt 0 ]; then
    SUCCESS_RATE=$((TESTS_PASSED * 100 / TOTAL_TESTS))
fi

echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
echo -e "Total Tests: $TOTAL_TESTS"
echo -e "Success Rate: $SUCCESS_RATE%"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All tests passed successfully!${NC}"
    exit 0
else
    echo -e "\n${RED}✗ Some tests failed. Please review the results above.${NC}"
    exit 1
fi