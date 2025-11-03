#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Teller Service Status ===${NC}"
echo ""

# Function to check service health
check_service() {
    local service=$1
    local port=$2
    local protocol=${3:-tcp}

    if docker-compose ps | grep -q "$service.*Up"; then
        echo -e "${GREEN}✓${NC} $service: ${GREEN}Running${NC}"

        if [ ! -z "$port" ]; then
            if nc -z localhost $port 2>/dev/null; then
                echo -e "  └─ Port $port: ${GREEN}Accessible${NC}"
            else
                echo -e "  └─ Port $port: ${RED}Not accessible${NC}"
            fi
        fi
    else
        echo -e "${RED}✗${NC} $service: ${RED}Not running${NC}"
    fi
}

# Check core services
echo -e "${YELLOW}Core Services:${NC}"
check_service "postgres" 5432
check_service "redis" 6379

echo ""
echo -e "${YELLOW}Application Services:${NC}"
check_service "teller-server" 8000
check_service "teller-worker-1"
check_service "teller-worker-2"
check_service "teller-worker-3"

echo ""
echo -e "${YELLOW}Optional Services:${NC}"
check_service "kafka" 9092
check_service "zookeeper" 2181
check_service "pgadmin" 5050

# Database statistics
echo ""
echo -e "${BLUE}=== Database Statistics ===${NC}"
docker-compose exec -T postgres psql -U postgres -d telltheworld -c "
SELECT
    status,
    COUNT(*) as count,
    COUNT(CASE WHEN processing_worker_id IS NOT NULL THEN 1 END) as processing_by_worker
FROM notifications
GROUP BY status
ORDER BY status;
" 2>/dev/null || echo -e "${RED}Could not connect to database${NC}"

# Worker statistics
echo ""
echo -e "${BLUE}=== Worker Statistics ===${NC}"
docker-compose exec -T postgres psql -U postgres -d telltheworld -c "
SELECT
    processing_worker_id,
    COUNT(*) as notifications_processing
FROM notifications
WHERE status = 'processing'
  AND processing_worker_id IS NOT NULL
GROUP BY processing_worker_id
ORDER BY processing_worker_id;
" 2>/dev/null || echo -e "${YELLOW}No active workers or database not accessible${NC}"

# Recent notifications
echo ""
echo -e "${BLUE}=== Recent Notifications ===${NC}"
docker-compose exec -T postgres psql -U postgres -d telltheworld -c "
SELECT
    notification_id,
    title,
    status,
    processing_attempts,
    created_at
FROM notifications
ORDER BY created_at DESC
LIMIT 5;
" 2>/dev/null || echo -e "${YELLOW}No notifications found or database not accessible${NC}"

# Container resource usage
echo ""
echo -e "${BLUE}=== Container Resource Usage ===${NC}"
docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep teller

# Show logs command
echo ""
echo -e "${BLUE}=== Useful Commands ===${NC}"
echo -e "${YELLOW}View server logs:${NC} docker-compose logs -f teller-server"
echo -e "${YELLOW}View worker logs:${NC} docker-compose logs -f teller-worker-1 teller-worker-2"
echo -e "${YELLOW}View database logs:${NC} docker-compose logs -f postgres"
echo -e "${YELLOW}Access database:${NC} docker-compose exec postgres psql -U postgres -d telltheworld"
echo -e "${YELLOW}Access Redis:${NC} docker-compose exec redis redis-cli"