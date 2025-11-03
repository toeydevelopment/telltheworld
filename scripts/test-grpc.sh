#!/bin/bash

# Test script for gRPC API
# This script tests the notification service via gRPC

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

GRPC_HOST="localhost:9000"
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="telltheworld"
DB_USER="postgres"
DB_PASSWORD="${POSTGRES_PASSWORD:-password}"

echo -e "${BLUE}=== gRPC Notification Service Test ===${NC}"
echo ""

# Function to execute SQL query via podman
execute_sql() {
    podman exec teller-postgres psql -U postgres -d telltheworld -t -c "$1" 2>/dev/null
}

# Check if grpcurl is installed
if ! command -v grpcurl > /dev/null 2>&1; then
    echo -e "${YELLOW}grpcurl is not installed. Installing...${NC}"
    # Try to install grpcurl based on the platform
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if command -v brew > /dev/null 2>&1; then
            brew install grpcurl
        else
            echo -e "${RED}Please install grpcurl manually: https://github.com/fullstorydev/grpcurl${NC}"
            exit 1
        fi
    else
        echo -e "${RED}Please install grpcurl: https://github.com/fullstorydev/grpcurl${NC}"
        exit 1
    fi
fi

# Test 1: Check gRPC server connectivity
echo -e "${YELLOW}Test 1: Checking gRPC server connectivity...${NC}"
if nc -zv localhost 9000 2>&1 | grep -q succeeded; then
    echo -e "${GREEN}✓ gRPC server is listening on port 9000${NC}"
else
    echo -e "${RED}✗ gRPC server is not accessible${NC}"
    exit 1
fi

# Test 2: List available services (using reflection if enabled)
echo -e "\n${YELLOW}Test 2: Attempting to list gRPC services...${NC}"
grpcurl -plaintext $GRPC_HOST list 2>/dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ gRPC reflection is enabled${NC}"
else
    echo -e "${YELLOW}Note: gRPC reflection might be disabled, using known service${NC}"
fi

# Test 3: Send a test notification via gRPC
echo -e "\n${YELLOW}Test 3: Sending test notification via gRPC...${NC}"

# Create JSON payload for gRPC request
GRPC_REQUEST='{
  "notification": {
    "title": "gRPC Test Notification",
    "body": "This is a test via gRPC",
    "ref_id": "grpc-test-001"
  },
  "destinations": [
    {
      "channel": 1,
      "destination": "grpc@test.com"
    },
    {
      "channel": 2,
      "destination": "device_grpc_123"
    }
  ]
}'

echo "Request payload:"
echo "$GRPC_REQUEST" | jq . 2>/dev/null || echo "$GRPC_REQUEST"

# Try to send the notification
RESPONSE=$(echo "$GRPC_REQUEST" | grpcurl -plaintext -d @ $GRPC_HOST teller.v1.NotificationService/SendToUser 2>&1)

if echo "$RESPONSE" | grep -q "notificationId"; then
    echo -e "${GREEN}✓ Successfully sent notification via gRPC${NC}"
    NOTIFICATION_ID=$(echo "$RESPONSE" | jq -r '.notificationId' 2>/dev/null)
    echo "Notification ID: $NOTIFICATION_ID"
else
    echo -e "${RED}✗ Failed to send notification${NC}"
    echo "Response: $RESPONSE"
fi

# Test 4: Check database for notifications
echo -e "\n${YELLOW}Test 4: Checking database for notifications...${NC}"
COUNT=$(execute_sql "SELECT COUNT(*) FROM notifications")
if [ "$COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Found $COUNT notification(s) in database${NC}"

    # Show recent notifications
    echo -e "\nRecent notifications:"
    execute_sql "SELECT notification_id, title, status, created_at FROM notifications ORDER BY created_at DESC LIMIT 3" | column -t
else
    echo -e "${YELLOW}No notifications found in database${NC}"
fi

# Test 5: Test bulk send via gRPC
echo -e "\n${YELLOW}Test 5: Testing bulk send via gRPC...${NC}"

BULK_REQUEST='{
  "requests": [
    {
      "notification": {
        "title": "Bulk Test 1",
        "body": "First bulk notification",
        "ref_id": "bulk-test-001"
      },
      "destinations": [
        {"channel": 1, "destination": "bulk1@test.com"}
      ]
    },
    {
      "notification": {
        "title": "Bulk Test 2",
        "body": "Second bulk notification",
        "ref_id": "bulk-test-002"
      },
      "destinations": [
        {"channel": 2, "destination": "device_bulk_456"}
      ]
    }
  ]
}'

BULK_RESPONSE=$(echo "$BULK_REQUEST" | grpcurl -plaintext -d @ $GRPC_HOST teller.v1.NotificationService/BulkSendToUser 2>&1)

if echo "$BULK_RESPONSE" | grep -q "notificationId"; then
    echo -e "${GREEN}✓ Successfully sent bulk notifications via gRPC${NC}"
    echo "$BULK_RESPONSE" | jq . 2>/dev/null || echo "$BULK_RESPONSE"
else
    echo -e "${RED}✗ Failed to send bulk notifications${NC}"
    echo "Response: $BULK_RESPONSE"
fi

# Test 6: Check worker processing
echo -e "\n${YELLOW}Test 6: Checking worker processing...${NC}"
PROCESSING=$(execute_sql "SELECT COUNT(*) FROM notifications WHERE status IN ('processing', 'completed')")
if [ "$PROCESSING" -gt 0 ]; then
    echo -e "${GREEN}✓ Workers are processing notifications ($PROCESSING processed/processing)${NC}"
else
    echo -e "${YELLOW}No notifications have been processed yet${NC}"
fi

# Test 7: Check NATS messages
echo -e "\n${YELLOW}Test 7: Checking NATS JetStream...${NC}"
NATS_STATUS=$(curl -s http://localhost:8222/jsz | jq '.streams[0].state.messages' 2>/dev/null)
if [ ! -z "$NATS_STATUS" ]; then
    echo -e "${GREEN}✓ NATS JetStream has $NATS_STATUS messages${NC}"
else
    echo -e "${YELLOW}Could not check NATS status${NC}"
fi

# Summary
echo -e "\n${BLUE}=== Test Summary ===${NC}"
FINAL_COUNT=$(execute_sql "SELECT COUNT(*) FROM notifications")
echo "Total notifications in database: $FINAL_COUNT"

STATUS_SUMMARY=$(execute_sql "SELECT status, COUNT(*) FROM notifications GROUP BY status")
if [ ! -z "$STATUS_SUMMARY" ]; then
    echo -e "\nNotification status breakdown:"
    echo "$STATUS_SUMMARY" | column -t
fi

echo -e "\n${GREEN}gRPC testing complete!${NC}"
echo ""
echo -e "${YELLOW}Useful commands:${NC}"
echo "• List gRPC services: grpcurl -plaintext localhost:9000 list"
echo "• Check database: podman exec teller-postgres psql -U postgres -d telltheworld"
echo "• View logs: podman logs teller-server"
echo "• Monitor NATS: curl http://localhost:8222/jsz | jq"