#!/bin/bash

# Test script adapted for Podman/Podman-Compose
# Uses podman-compose instead of docker-compose

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

# Function to execute SQL query
execute_sql() {
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "$1" 2>/dev/null
}

# Function to generate random email
generate_email() {
    echo "test_$(date +%s)_$RANDOM@example.com"
}

print_test_header "Podman-Based Notification Service Test Suite"

# Step 1: Check Container Status
print_test_header "Step 1: Container Status Check"

# Check if podman-compose is available
if command -v podman-compose > /dev/null 2>&1; then
    print_test_result "PASS" "podman-compose is installed"
else
    print_test_result "FAIL" "podman-compose is not installed"
    exit 1
fi

# Check container status
CONTAINER_STATUS=$(podman-compose ps --format json 2>/dev/null)
if [ $? -eq 0 ]; then
    print_test_result "PASS" "Successfully retrieved container status"

    # Check individual containers
    for service in postgres nats teller-server; do
        if podman ps --format "{{.Names}}" | grep -q "$service"; then
            print_test_result "PASS" "$service container is running"
        else
            print_test_result "FAIL" "$service container is not running"
        fi
    done
else
    print_test_result "FAIL" "Failed to get container status"
fi

# Step 2: Check PostgreSQL Connection
print_test_header "Step 2: Database Connectivity"

if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\q" 2>/dev/null; then
    print_test_result "PASS" "PostgreSQL is accessible"

    # Check if database exists
    DB_EXISTS=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -t -c "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" 2>/dev/null)
    if [ ! -z "$DB_EXISTS" ]; then
        print_test_result "PASS" "Database '$DB_NAME' exists"
    else
        print_test_result "FAIL" "Database '$DB_NAME' does not exist"
    fi
else
    print_test_result "FAIL" "Cannot connect to PostgreSQL"
fi

# Step 3: Check NATS Health
print_test_header "Step 3: NATS Service Check"

NATS_HEALTH=$(curl -s http://localhost:8222/healthz 2>/dev/null)
if [ "$NATS_HEALTH" == '{"status":"ok"}' ]; then
    print_test_result "PASS" "NATS server is healthy"
else
    print_test_result "FAIL" "NATS health check failed"
fi

# Step 4: Test API Availability
print_test_header "Step 4: API Server Check"

# Wait a moment for server to stabilize
sleep 2

# Check if API is responding
API_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/healthz 2>/dev/null)
if [ "$API_RESPONSE" == "200" ]; then
    print_test_result "PASS" "API server is responding"
else
    print_test_result "FAIL" "API server is not responding (HTTP $API_RESPONSE)"

    # Check server logs for errors
    echo -e "${YELLOW}Checking server logs for errors...${NC}"
    podman logs teller-server --tail 10 2>&1 | grep -i error | head -5
fi

# Step 5: Simple API Test (if server is running)
if [ "$API_RESPONSE" == "200" ]; then
    print_test_header "Step 5: API Functionality Test"

    TEST_EMAIL=$(generate_email)

    # Send test notification
    SEND_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/notifications/send" \
        -H 'Content-Type: application/json' \
        -d '{
            "notification": {
                "title": "Podman Test",
                "body": "Testing with Podman",
                "ref_id": "podman-test-001"
            },
            "destinations": [
                {"channel": 1, "destination": "'$TEST_EMAIL'"}
            ]
        }' 2>/dev/null)

    if echo "$SEND_RESPONSE" | grep -q "notificationId"; then
        NOTIFICATION_ID=$(echo $SEND_RESPONSE | jq -r '.notificationId' 2>/dev/null)
        print_test_result "PASS" "Created notification: $NOTIFICATION_ID"
    else
        print_test_result "FAIL" "Failed to create notification"
        echo "Response: $SEND_RESPONSE"
    fi
fi

# Step 6: Container Resource Usage
print_test_header "Step 6: Container Resources"

echo -e "${YELLOW}Container resource usage:${NC}"
podman stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep teller

# Summary
print_test_header "Test Summary"

SUCCESS_RATE=0
if [ $TOTAL_TESTS -gt 0 ]; then
    SUCCESS_RATE=$((TESTS_PASSED * 100 / TOTAL_TESTS))
fi

echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
echo -e "Total Tests: $TOTAL_TESTS"
echo -e "Success Rate: $SUCCESS_RATE%"

# Provide debugging help if tests failed
if [ $TESTS_FAILED -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}=== Debugging Help ===${NC}"
    echo "1. Check container status: podman-compose ps"
    echo "2. View server logs: podman logs teller-server"
    echo "3. View worker logs: podman logs teller-worker-1"
    echo "4. Check NATS: curl http://localhost:8222/varz"
    echo "5. Restart services: podman-compose restart"
fi

exit $((TESTS_FAILED > 0 ? 1 : 0))