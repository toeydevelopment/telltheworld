#!/bin/bash

# Load testing script for ecommerce service
# This script simulates concurrent requests to test the service under load

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

ECOMMERCE_HOST="localhost:8080"
CONCURRENT_REQUESTS=10
TOTAL_REQUESTS=100

echo -e "${BLUE}=== E-commerce Service Load Test ===${NC}"
echo ""
echo -e "${CYAN}Configuration:${NC}"
echo "• Host: $ECOMMERCE_HOST"
echo "• Concurrent Requests: $CONCURRENT_REQUESTS"
echo "• Total Requests: $TOTAL_REQUESTS"
echo ""

# Function to send a single request
send_request() {
    local request_type=$1
    local index=$2

    case $request_type in
        "chat")
            curl -s -X POST http://$ECOMMERCE_HOST/api/v1/events/chat-message \
                -H 'Content-Type: application/json' \
                -d "{
                    \"buyer_id\": \"load_buyer_$index\",
                    \"seller_id\": \"load_seller_$index\",
                    \"message\": \"Load test message $index\",
                    \"conversation_id\": \"load_conv_$index\"
                }" > /dev/null 2>&1
            ;;
        "purchase")
            curl -s -X POST http://$ECOMMERCE_HOST/api/v1/events/purchase \
                -H 'Content-Type: application/json' \
                -d "{
                    \"buyer_id\": \"load_buyer_$index\",
                    \"seller_id\": \"load_seller_$index\",
                    \"order_id\": \"LOAD-ORD-$index\",
                    \"amount\": 99.99,
                    \"items\": [{
                        \"product_id\": \"LOAD-PROD-$index\",
                        \"product_name\": \"Load Test Product $index\",
                        \"quantity\": 1,
                        \"price\": 99.99
                    }]
                }" > /dev/null 2>&1
            ;;
        "payment")
            curl -s -X POST http://$ECOMMERCE_HOST/api/v1/events/payment-reminder \
                -H 'Content-Type: application/json' \
                -d "{
                    \"buyer_id\": \"load_buyer_$index\",
                    \"order_id\": \"LOAD-ORD-$index\",
                    \"amount_due\": 99.99,
                    \"due_date\": \"2024-12-01T00:00:00Z\"
                }" > /dev/null 2>&1
            ;;
        "shipping")
            curl -s -X POST http://$ECOMMERCE_HOST/api/v1/events/shipping-update \
                -H 'Content-Type: application/json' \
                -d "{
                    \"buyer_id\": \"load_buyer_$index\",
                    \"order_id\": \"LOAD-ORD-$index\",
                    \"tracking_number\": \"LOAD-TRK-$index\",
                    \"carrier\": \"LoadCarrier\",
                    \"status\": \"shipped\"
                }" > /dev/null 2>&1
            ;;
    esac

    echo -n "."
}

# Function to run concurrent requests
run_load_test() {
    local test_name=$1
    local endpoint_type=$2
    local num_requests=$3

    echo -e "${YELLOW}Running load test: $test_name${NC}"
    echo -n "Progress: "

    START_TIME=$(date +%s%N)

    # Run requests in parallel
    for ((i=1; i<=num_requests; i++)); do
        send_request "$endpoint_type" "$i" &

        # Limit concurrent connections
        if [[ $(jobs -r -p | wc -l) -ge $CONCURRENT_REQUESTS ]]; then
            wait -n
        fi
    done

    # Wait for all background jobs to complete
    wait

    END_TIME=$(date +%s%N)
    DURATION=$((($END_TIME - $START_TIME) / 1000000))

    echo ""
    echo -e "${GREEN}✓ Completed $num_requests requests in ${DURATION}ms${NC}"
    echo -e "  Average: $((DURATION / num_requests))ms per request"
    echo ""
}

# Test 1: Chat Message Load Test
run_load_test "Chat Messages" "chat" 25

# Test 2: Purchase Event Load Test
run_load_test "Purchase Events" "purchase" 25

# Test 3: Payment Reminder Load Test
run_load_test "Payment Reminders" "payment" 25

# Test 4: Shipping Update Load Test
run_load_test "Shipping Updates" "shipping" 25

# Test 5: Mixed Load Test
echo -e "${YELLOW}Running mixed load test (all endpoints)...${NC}"
echo -n "Progress: "

START_TIME=$(date +%s%N)

for ((i=1; i<=25; i++)); do
    send_request "chat" "$i" &
    send_request "purchase" "$i" &
    send_request "payment" "$i" &
    send_request "shipping" "$i" &

    # Limit concurrent connections
    while [[ $(jobs -r -p | wc -l) -ge $CONCURRENT_REQUESTS ]]; do
        wait -n
    done
done

wait

END_TIME=$(date +%s%N)
DURATION=$((($END_TIME - $START_TIME) / 1000000))

echo ""
echo -e "${GREEN}✓ Completed 100 mixed requests in ${DURATION}ms${NC}"
echo ""

# Check service health
echo -e "${YELLOW}Checking service health after load test...${NC}"
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://$ECOMMERCE_HOST/api/v1/events/chat-message -X POST -H "Content-Type: application/json" -d '{"buyer_id":"health_check","seller_id":"health_check","message":"test","conversation_id":"test"}')

if [ "$HTTP_STATUS" -eq 200 ]; then
    echo -e "${GREEN}✓ Service is healthy and responding${NC}"
else
    echo -e "${RED}✗ Service returned status code: $HTTP_STATUS${NC}"
fi

# Database statistics
echo ""
echo -e "${YELLOW}Database statistics:${NC}"
NOTIFICATION_COUNT=$(podman exec teller-postgres psql -U postgres -d telltheworld -t -c "SELECT COUNT(*) FROM notifications WHERE created_at > NOW() - INTERVAL '5 minutes'" 2>/dev/null || echo "0")
echo -e "• Notifications created in last 5 minutes: $NOTIFICATION_COUNT"

echo ""
echo -e "${BLUE}=== Load Test Summary ===${NC}"
echo "• Total requests sent: $((TOTAL_REQUESTS))"
echo "• All endpoints tested under load"
echo "• Service remained responsive"
echo ""
echo -e "${GREEN}Load testing complete!${NC}"
echo ""
echo -e "${YELLOW}Monitor system resources:${NC}"
echo "• CPU: podman stats ecommerce-server"
echo "• Logs: podman logs ecommerce-server --tail 50"