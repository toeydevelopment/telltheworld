#!/bin/bash

# Test script for ecommerce service events
# This script tests all four event types: chat message, purchase, payment reminder, and shipping update

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

ECOMMERCE_HOST="localhost:8080"

echo -e "${BLUE}=== E-commerce Service Event Test ===${NC}"
echo ""

# Test 1: Chat Message Event
echo -e "${YELLOW}Test 1: Sending Chat Message Event...${NC}"
curl -X POST http://$ECOMMERCE_HOST/api/v1/events/chat-message \
  -H 'Content-Type: application/json' \
  -d '{
    "buyer_id": "buyer_123",
    "seller_id": "seller_456",
    "message": "Is this item still available?",
    "conversation_id": "conv_789"
  }' | jq . 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Chat message event sent${NC}"
else
    echo -e "${RED}✗ Failed to send chat message event${NC}"
fi

echo ""

# Test 2: Purchase Event
echo -e "${YELLOW}Test 2: Triggering Purchase Event...${NC}"
curl -X POST http://$ECOMMERCE_HOST/api/v1/events/purchase \
  -H 'Content-Type: application/json' \
  -d '{
    "buyer_id": "buyer_123",
    "seller_id": "seller_456",
    "order_id": "ORD-2024-001",
    "amount": 299.99,
    "items": [
      {
        "product_id": "PROD-001",
        "product_name": "Wireless Headphones",
        "quantity": 1,
        "price": 299.99
      }
    ]
  }' | jq . 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Purchase event sent${NC}"
else
    echo -e "${RED}✗ Failed to send purchase event${NC}"
fi

echo ""

# Test 3: Payment Reminder Event
echo -e "${YELLOW}Test 3: Sending Payment Reminder...${NC}"
curl -X POST http://$ECOMMERCE_HOST/api/v1/events/payment-reminder \
  -H 'Content-Type: application/json' \
  -d '{
    "buyer_id": "buyer_123",
    "order_id": "ORD-2024-001",
    "amount_due": 299.99,
    "due_date": "2024-11-10T23:59:59Z"
  }' | jq . 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Payment reminder sent${NC}"
else
    echo -e "${RED}✗ Failed to send payment reminder${NC}"
fi

echo ""

# Test 4: Shipping Update Event
echo -e "${YELLOW}Test 4: Sending Shipping Update...${NC}"
curl -X POST http://$ECOMMERCE_HOST/api/v1/events/shipping-update \
  -H 'Content-Type: application/json' \
  -d '{
    "buyer_id": "buyer_123",
    "order_id": "ORD-2024-001",
    "tracking_number": "TRK-123456789",
    "carrier": "FedEx",
    "status": "shipped"
  }' | jq . 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Shipping update sent${NC}"
else
    echo -e "${RED}✗ Failed to send shipping update${NC}"
fi

echo ""

# Test 5: Check notifications in database
echo -e "${YELLOW}Test 5: Checking database for notifications...${NC}"
NOTIFICATION_COUNT=$(podman exec teller-postgres psql -U postgres -d telltheworld -t -c "SELECT COUNT(*) FROM notifications WHERE created_at > NOW() - INTERVAL '5 minutes'" 2>/dev/null)

if [ ! -z "$NOTIFICATION_COUNT" ] && [ "$NOTIFICATION_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Found $NOTIFICATION_COUNT recent notification(s) in database${NC}"

    echo -e "\nRecent notifications:"
    podman exec teller-postgres psql -U postgres -d telltheworld -c "SELECT notification_id, title, status, created_at FROM notifications WHERE created_at > NOW() - INTERVAL '5 minutes' ORDER BY created_at DESC LIMIT 5" 2>/dev/null
else
    echo -e "${YELLOW}No recent notifications found in database${NC}"
fi

echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "• Chat Message Event: Buyer → Seller notification"
echo -e "• Purchase Event: Order notification to seller"
echo -e "• Payment Reminder: Payment due notification to buyer"
echo -e "• Shipping Update: Tracking info to buyer"

echo ""
echo -e "${GREEN}E-commerce event testing complete!${NC}"
echo ""
echo -e "${YELLOW}Check logs for details:${NC}"
echo "• Ecommerce service: podman logs ecommerce-server"
echo "• Teller service: podman logs teller-server"
echo "• Worker: podman logs teller-worker-1"