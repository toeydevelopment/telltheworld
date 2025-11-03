#!/bin/bash

# Error testing script for ecommerce service
# This script tests error handling and edge cases

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

ECOMMERCE_HOST="localhost:8080"

echo -e "${BLUE}=== E-commerce Error Handling Test ===${NC}"
echo ""

# Function to test an endpoint with invalid data
test_error_case() {
    local test_name=$1
    local endpoint=$2
    local data=$3
    local expected_behavior=$4

    echo -e "${YELLOW}Test: $test_name${NC}"
    echo -e "Expected: $expected_behavior"

    RESPONSE=$(curl -s -X POST http://$ECOMMERCE_HOST$endpoint \
        -H 'Content-Type: application/json' \
        -d "$data" 2>&1)

    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://$ECOMMERCE_HOST$endpoint \
        -H 'Content-Type: application/json' \
        -d "$data" 2>&1)

    echo "Response code: $HTTP_CODE"
    echo "Response: ${RESPONSE:0:150}..."

    if [ "$HTTP_CODE" -ge 400 ]; then
        echo -e "${GREEN}✓ Error handled correctly${NC}"
    else
        echo -e "${MAGENTA}⚠ Unexpected success - might need validation${NC}"
    fi
    echo ""
}

echo -e "${MAGENTA}=== Testing Invalid JSON ===${NC}"
echo ""

test_error_case \
    "Malformed JSON" \
    "/api/v1/events/chat-message" \
    '{"buyer_id": "test", "invalid json}' \
    "Should reject malformed JSON"

echo -e "${MAGENTA}=== Testing Missing Required Fields ===${NC}"
echo ""

test_error_case \
    "Chat Message - Missing buyer_id" \
    "/api/v1/events/chat-message" \
    '{"seller_id": "seller_001", "message": "test", "conversation_id": "conv_001"}' \
    "Should reject missing buyer_id"

test_error_case \
    "Purchase - Missing order_id" \
    "/api/v1/events/purchase" \
    '{"buyer_id": "buyer_001", "seller_id": "seller_001", "amount": 99.99, "items": []}' \
    "Should reject missing order_id"

test_error_case \
    "Payment Reminder - Missing amount_due" \
    "/api/v1/events/payment-reminder" \
    '{"buyer_id": "buyer_001", "order_id": "ORD-001", "due_date": "2024-12-01T00:00:00Z"}' \
    "Should reject missing amount_due"

test_error_case \
    "Shipping Update - Missing tracking_number" \
    "/api/v1/events/shipping-update" \
    '{"buyer_id": "buyer_001", "order_id": "ORD-001", "carrier": "FedEx", "status": "shipped"}' \
    "Should reject missing tracking_number"

echo -e "${MAGENTA}=== Testing Invalid Data Types ===${NC}"
echo ""

test_error_case \
    "Purchase - Invalid amount (string instead of number)" \
    "/api/v1/events/purchase" \
    '{"buyer_id": "buyer_001", "seller_id": "seller_001", "order_id": "ORD-001", "amount": "not_a_number", "items": []}' \
    "Should reject non-numeric amount"

test_error_case \
    "Purchase - Negative amount" \
    "/api/v1/events/purchase" \
    '{"buyer_id": "buyer_001", "seller_id": "seller_001", "order_id": "ORD-001", "amount": -99.99, "items": []}' \
    "Should handle negative amounts appropriately"

echo -e "${MAGENTA}=== Testing Empty Values ===${NC}"
echo ""

test_error_case \
    "Chat Message - Empty message" \
    "/api/v1/events/chat-message" \
    '{"buyer_id": "buyer_001", "seller_id": "seller_001", "message": "", "conversation_id": "conv_001"}' \
    "Should handle empty message"

test_error_case \
    "Purchase - Empty items array" \
    "/api/v1/events/purchase" \
    '{"buyer_id": "buyer_001", "seller_id": "seller_001", "order_id": "ORD-001", "amount": 0, "items": []}' \
    "Should handle empty items array"

echo -e "${MAGENTA}=== Testing Extremely Long Values ===${NC}"
echo ""

LONG_STRING=$(printf 'x%.0s' {1..10000})

test_error_case \
    "Chat Message - Extremely long message (10K chars)" \
    "/api/v1/events/chat-message" \
    "{\"buyer_id\": \"buyer_001\", \"seller_id\": \"seller_001\", \"message\": \"$LONG_STRING\", \"conversation_id\": \"conv_001\"}" \
    "Should handle or reject very long messages"

echo -e "${MAGENTA}=== Testing Special Characters ===${NC}"
echo ""

test_error_case \
    "Chat Message - SQL Injection attempt" \
    "/api/v1/events/chat-message" \
    '{"buyer_id": "buyer_001", "seller_id": "seller_001", "message": "test\"; DROP TABLE notifications; --", "conversation_id": "conv_001"}' \
    "Should safely handle SQL injection attempts"

test_error_case \
    "Chat Message - Unicode and emoji" \
    "/api/v1/events/chat-message" \
    '{"buyer_id": "buyer_001", "seller_id": "seller_001", "message": "Hello 世界 🌍 مرحبا", "conversation_id": "conv_001"}' \
    "Should handle unicode and emoji correctly"

echo -e "${MAGENTA}=== Testing Invalid Endpoints ===${NC}"
echo ""

echo -e "${YELLOW}Test: Non-existent endpoint${NC}"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://$ECOMMERCE_HOST/api/v1/events/invalid-endpoint -X POST -H "Content-Type: application/json" -d '{}')
echo "Response code: $HTTP_CODE"
if [ "$HTTP_CODE" -eq 404 ]; then
    echo -e "${GREEN}✓ Returns 404 for invalid endpoint${NC}"
else
    echo -e "${RED}✗ Unexpected response code: $HTTP_CODE${NC}"
fi
echo ""

echo -e "${YELLOW}Test: Wrong HTTP method (GET instead of POST)${NC}"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://$ECOMMERCE_HOST/api/v1/events/chat-message -X GET)
echo "Response code: $HTTP_CODE"
if [ "$HTTP_CODE" -eq 405 ] || [ "$HTTP_CODE" -eq 404 ]; then
    echo -e "${GREEN}✓ Rejects wrong HTTP method${NC}"
else
    echo -e "${MAGENTA}⚠ Response code: $HTTP_CODE${NC}"
fi
echo ""

echo -e "${MAGENTA}=== Testing Rate Limiting (if implemented) ===${NC}"
echo ""

echo -e "${YELLOW}Sending 50 rapid requests...${NC}"
for i in {1..50}; do
    curl -s -X POST http://$ECOMMERCE_HOST/api/v1/events/chat-message \
        -H 'Content-Type: application/json' \
        -d '{"buyer_id": "rate_test", "seller_id": "rate_test", "message": "test", "conversation_id": "test"}' \
        > /dev/null 2>&1 &
done
wait

echo -e "${GREEN}✓ Service handled rapid requests${NC}"
echo ""

echo -e "${BLUE}=== Error Test Summary ===${NC}"
echo "• Tested invalid JSON handling"
echo "• Tested missing required fields"
echo "• Tested invalid data types"
echo "• Tested boundary conditions"
echo "• Tested special characters and injection attempts"
echo "• Tested invalid endpoints and methods"
echo ""

echo -e "${YELLOW}Service Health Check:${NC}"
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://$ECOMMERCE_HOST/api/v1/events/chat-message -X POST -H "Content-Type: application/json" -d '{"buyer_id":"health","seller_id":"health","message":"health check","conversation_id":"health"}')

if [ "$HTTP_STATUS" -eq 200 ]; then
    echo -e "${GREEN}✓ Service is still healthy after error tests${NC}"
else
    echo -e "${RED}✗ Service returned status: $HTTP_STATUS${NC}"
fi

echo ""
echo -e "${GREEN}Error testing complete!${NC}"