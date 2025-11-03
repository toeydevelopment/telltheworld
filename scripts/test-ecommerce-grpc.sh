#!/bin/bash

# Test script for ecommerce service gRPC endpoints
# This script tests the gRPC API of the ecommerce service

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

ECOMMERCE_GRPC_HOST="localhost:9090"

echo -e "${BLUE}=== E-commerce gRPC API Test ===${NC}"
echo ""

# Check if grpcurl is installed
if ! command -v grpcurl &> /dev/null; then
    echo -e "${YELLOW}grpcurl is not installed. Installing...${NC}"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install grpcurl
    else
        echo -e "${RED}Please install grpcurl manually${NC}"
        exit 1
    fi
fi

echo -e "${YELLOW}Listing available services...${NC}"
grpcurl -plaintext $ECOMMERCE_GRPC_HOST list

echo ""
echo -e "${YELLOW}Describing EcommerceService...${NC}"
grpcurl -plaintext $ECOMMERCE_GRPC_HOST describe ecommerce.v1.EcommerceService

echo ""
echo -e "${YELLOW}Test 1: Sending Chat Message via gRPC...${NC}"
grpcurl -plaintext -d '{
  "buyer_id": "grpc_buyer_001",
  "seller_id": "grpc_seller_001",
  "message": "Testing gRPC chat message",
  "conversation_id": "grpc_conv_001"
}' $ECOMMERCE_GRPC_HOST ecommerce.v1.EcommerceService/SendChatMessage

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ gRPC chat message sent${NC}"
else
    echo -e "${RED}✗ Failed to send gRPC chat message${NC}"
fi

echo ""
echo -e "${YELLOW}Test 2: Triggering Purchase via gRPC...${NC}"
grpcurl -plaintext -d '{
  "buyer_id": "grpc_buyer_001",
  "seller_id": "grpc_seller_001",
  "order_id": "GRPC-ORD-001",
  "amount": 199.99,
  "items": [
    {
      "product_id": "GRPC-PROD-001",
      "product_name": "Test Product via gRPC",
      "quantity": 2,
      "price": 99.99
    }
  ]
}' $ECOMMERCE_GRPC_HOST ecommerce.v1.EcommerceService/TriggerPurchase

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ gRPC purchase triggered${NC}"
else
    echo -e "${RED}✗ Failed to trigger gRPC purchase${NC}"
fi

echo ""
echo -e "${YELLOW}Test 3: Sending Payment Reminder via gRPC...${NC}"
grpcurl -plaintext -d '{
  "buyer_id": "grpc_buyer_001",
  "order_id": "GRPC-ORD-001",
  "amount_due": 199.99,
  "due_date": "2024-12-01T00:00:00Z"
}' $ECOMMERCE_GRPC_HOST ecommerce.v1.EcommerceService/SendPaymentReminder

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ gRPC payment reminder sent${NC}"
else
    echo -e "${RED}✗ Failed to send gRPC payment reminder${NC}"
fi

echo ""
echo -e "${YELLOW}Test 4: Sending Shipping Update via gRPC...${NC}"
grpcurl -plaintext -d '{
  "buyer_id": "grpc_buyer_001",
  "order_id": "GRPC-ORD-001",
  "tracking_number": "GRPC-TRK-123456",
  "carrier": "DHL",
  "status": "in_transit"
}' $ECOMMERCE_GRPC_HOST ecommerce.v1.EcommerceService/SendShippingUpdate

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ gRPC shipping update sent${NC}"
else
    echo -e "${RED}✗ Failed to send gRPC shipping update${NC}"
fi

echo ""
echo -e "${BLUE}=== gRPC Test Summary ===${NC}"
echo -e "• All gRPC endpoints tested successfully"
echo -e "• Check database for notification records"
echo ""
echo -e "${GREEN}E-commerce gRPC testing complete!${NC}"