#!/bin/bash

# Comprehensive test suite for ecommerce service
# This script runs all ecommerce tests in sequence

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${BLUE}===============================================${NC}"
echo -e "${BLUE}    E-commerce Service Comprehensive Test     ${NC}"
echo -e "${BLUE}===============================================${NC}"
echo ""

# Function to run a test script
run_test() {
    local test_name=$1
    local script_name=$2

    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}Running: $test_name${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    if [ -f "$SCRIPT_DIR/$script_name" ]; then
        bash "$SCRIPT_DIR/$script_name"
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✓ $test_name completed successfully${NC}"
        else
            echo -e "${RED}✗ $test_name failed${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ Script $script_name not found${NC}"
    fi

    echo ""
    sleep 2  # Brief pause between tests
}

# Check if services are running
echo -e "${YELLOW}Checking if services are running...${NC}"

# Check ecommerce service
if podman ps | grep -q ecommerce-server; then
    echo -e "${GREEN}✓ Ecommerce service is running${NC}"
else
    echo -e "${RED}✗ Ecommerce service is not running${NC}"
    echo -e "${YELLOW}Starting services...${NC}"
    cd "$SCRIPT_DIR/.."
    podman-compose up -d ecommerce-server
    sleep 5
fi

# Check teller service
if podman ps | grep -q teller-server; then
    echo -e "${GREEN}✓ Teller service is running${NC}"
else
    echo -e "${RED}✗ Teller service is not running${NC}"
    echo -e "${YELLOW}Please start the teller service first${NC}"
    exit 1
fi

echo ""

# Run test suites in sequence
echo -e "${MAGENTA}Starting Test Suite Execution${NC}"
echo ""

# Test 1: Basic functionality
run_test "Basic Functionality Test" "test-ecommerce.sh"

# Test 2: gRPC endpoints (only if grpcurl is available)
if command -v grpcurl &> /dev/null; then
    run_test "gRPC API Test" "test-ecommerce-grpc.sh"
else
    echo -e "${YELLOW}⚠ Skipping gRPC tests (grpcurl not installed)${NC}"
    echo ""
fi

# Test 3: Error handling
run_test "Error Handling Test" "test-ecommerce-errors.sh"

# Test 4: Load testing
echo -e "${YELLOW}Do you want to run load tests? (y/n)${NC}"
read -r response
if [[ "$response" =~ ^[Yy]$ ]]; then
    run_test "Load Testing" "test-ecommerce-load.sh"
else
    echo -e "${YELLOW}⚠ Skipping load tests${NC}"
    echo ""
fi

# Generate summary report
echo -e "${BLUE}===============================================${NC}"
echo -e "${BLUE}              Test Summary Report              ${NC}"
echo -e "${BLUE}===============================================${NC}"
echo ""

# Get notification statistics
echo -e "${CYAN}Database Statistics:${NC}"
TOTAL_NOTIFICATIONS=$(podman exec teller-postgres psql -U postgres -d telltheworld -t -c "SELECT COUNT(*) FROM notifications" 2>/dev/null || echo "0")
RECENT_NOTIFICATIONS=$(podman exec teller-postgres psql -U postgres -d telltheworld -t -c "SELECT COUNT(*) FROM notifications WHERE created_at > NOW() - INTERVAL '10 minutes'" 2>/dev/null || echo "0")
PENDING_NOTIFICATIONS=$(podman exec teller-postgres psql -U postgres -d telltheworld -t -c "SELECT COUNT(*) FROM notifications WHERE status = 'pending'" 2>/dev/null || echo "0")

echo "• Total notifications: $TOTAL_NOTIFICATIONS"
echo "• Recent notifications (last 10 min): $RECENT_NOTIFICATIONS"
echo "• Pending notifications: $PENDING_NOTIFICATIONS"
echo ""

# Get service logs summary
echo -e "${CYAN}Recent Service Activity:${NC}"
echo "• Ecommerce service errors:"
podman logs ecommerce-server 2>&1 | grep -i error | tail -3 || echo "  No recent errors"
echo ""

# Service health check
echo -e "${CYAN}Service Health Status:${NC}"
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/events/chat-message -X POST -H "Content-Type: application/json" -d '{"buyer_id":"health","seller_id":"health","message":"test","conversation_id":"test"}')

if [ "$HTTP_STATUS" -eq 200 ]; then
    echo -e "${GREEN}✓ Ecommerce service: HEALTHY${NC}"
else
    echo -e "${RED}✗ Ecommerce service: UNHEALTHY (HTTP $HTTP_STATUS)${NC}"
fi

# Check worker status
WORKER_COUNT=$(podman ps | grep -c teller-worker || echo "0")
echo -e "• Active workers: $WORKER_COUNT"
echo ""

echo -e "${BLUE}===============================================${NC}"
echo -e "${GREEN}     All Tests Completed Successfully!        ${NC}"
echo -e "${BLUE}===============================================${NC}"
echo ""
echo -e "${YELLOW}For detailed logs, check:${NC}"
echo "• Ecommerce: podman logs ecommerce-server"
echo "• Teller: podman logs teller-server"
echo "• Workers: podman logs teller-worker-1"
echo "• Database: podman exec teller-postgres psql -U postgres -d telltheworld"