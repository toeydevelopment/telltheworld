#!/bin/bash

# Start script for Teller Notification Service
# Default event broker: NATS with JetStream
# Use --redis flag to switch to Redis pubsub
# Use --kafka flag to add Kafka support

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color
alias docker-compose="docker-compose"
echo -e "${GREEN}Starting Teller Notification Service...${NC}"
echo -e "${YELLOW}Using NATS as default event broker${NC}"

# Check if .env exists, if not copy from .env.example
if [ ! -f .env ]; then
    echo -e "${YELLOW}Creating .env from .env.example...${NC}"
    cp .env.example .env
    echo -e "${GREEN}Please update .env with your configuration${NC}"
fi

# Parse arguments
PROFILE=""
BUILD_FLAG=""
SCALE_WORKERS=2
EVENT_BROKER="nats"

while [[ $# -gt 0 ]]; do
    case $1 in
        --redis)
            PROFILE="$PROFILE --profile redis"
            EVENT_BROKER="redis"
            echo -e "${YELLOW}Using Redis as event broker...${NC}"
            shift
            ;;
        --kafka)
            PROFILE="--profile kafka"
            EVENT_BROKER="kafka"
            echo -e "${YELLOW}Starting with Kafka support...${NC}"
            shift
            ;;
        --tools)
            PROFILE="$PROFILE --profile tools"
            echo -e "${YELLOW}Starting with development tools (pgAdmin)...${NC}"
            shift
            ;;
        --scale)
            PROFILE="$PROFILE --profile scale"
            SCALE_WORKERS=3
            echo -e "${YELLOW}Starting with additional workers...${NC}"
            shift
            ;;
        --build)
            BUILD_FLAG="--build"
            echo -e "${YELLOW}Building images...${NC}"
            shift
            ;;
        --workers)
            SCALE_WORKERS="$2"
            echo -e "${YELLOW}Scaling to $SCALE_WORKERS workers...${NC}"
            shift
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo -e "${YELLOW}Available options:${NC}"
            echo "  --redis      Use Redis as event broker (default: NATS)"
            echo "  --kafka      Use Kafka as event broker"
            echo "  --tools      Include development tools (pgAdmin)"
            echo "  --scale      Start with 3 workers instead of 2"
            echo "  --workers N  Start with N workers"
            echo "  --build      Rebuild Docker images"
            exit 1
            ;;
    esac
done

# Start services based on selected event broker
echo -e "${GREEN}Starting core services...${NC}"
if [ "$EVENT_BROKER" == "redis" ]; then
    docker-compose $PROFILE up -d $BUILD_FLAG postgres redis
elif [ "$EVENT_BROKER" == "kafka" ]; then
    docker-compose $PROFILE up -d $BUILD_FLAG postgres zookeeper kafka
else
    # Default to NATS
    docker-compose $PROFILE up -d $BUILD_FLAG postgres nats
fi

# Wait for PostgreSQL to be ready
echo -e "${YELLOW}Waiting for PostgreSQL to be ready...${NC}"
until docker-compose exec -T postgres pg_isready -U postgres -d telltheworld > /dev/null 2>&1; do
    echo -n "."
    sleep 1
done
echo ""

# Wait for event broker to be ready
if [ "$EVENT_BROKER" == "nats" ]; then
    echo -e "${YELLOW}Waiting for NATS to be ready...${NC}"
    until curl -s http://localhost:8222/healthz > /dev/null 2>&1; do
        echo -n "."
        sleep 1
    done
    echo ""
elif [ "$EVENT_BROKER" == "redis" ]; then
    echo -e "${YELLOW}Waiting for Redis to be ready...${NC}"
    until docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; do
        echo -n "."
        sleep 1
    done
    echo ""
elif [ "$EVENT_BROKER" == "kafka" ]; then
    echo -e "${YELLOW}Waiting for Kafka to be ready...${NC}"
    sleep 10  # Kafka takes longer to start
    echo ""
fi

# Start server
echo -e "${GREEN}Starting notification server...${NC}"
docker-compose $PROFILE up -d $BUILD_FLAG teller-server

# Wait for server to be ready
echo -e "${YELLOW}Waiting for server to be ready...${NC}"
sleep 5

# Start workers
echo -e "${GREEN}Starting notification workers...${NC}"
docker-compose $PROFILE up -d $BUILD_FLAG teller-worker-1 teller-worker-2

if [ "$SCALE_WORKERS" -gt 2 ]; then
    docker-compose $PROFILE up -d teller-worker-3
fi

# Start ecommerce service
echo -e "${GREEN}Starting ecommerce service...${NC}"
docker-compose $PROFILE up -d $BUILD_FLAG ecommerce-server

# Show status
echo ""
echo -e "${GREEN}=== Service Status ===${NC}"
docker-compose ps

echo ""
echo -e "${GREEN}=== Service URLs ===${NC}"
echo -e "Notification API (HTTP): ${YELLOW}http://localhost:8000${NC}"
echo -e "Notification API (gRPC): ${YELLOW}localhost:9000${NC}"
echo -e "E-commerce API (HTTP): ${YELLOW}http://localhost:8080${NC}"
echo -e "E-commerce API (gRPC): ${YELLOW}localhost:9090${NC}"
echo -e "PostgreSQL: ${YELLOW}localhost:5432${NC}"

# Show event broker URLs based on what's running
if [ "$EVENT_BROKER" == "nats" ]; then
    echo -e "NATS (Event Broker): ${YELLOW}localhost:4222${NC}"
    echo -e "NATS Monitoring: ${YELLOW}http://localhost:8222${NC}"
elif [ "$EVENT_BROKER" == "redis" ]; then
    echo -e "Redis (Event Broker): ${YELLOW}localhost:6379${NC}"
elif [ "$EVENT_BROKER" == "kafka" ]; then
    echo -e "Kafka (Event Broker): ${YELLOW}localhost:9092${NC}"
fi

if [[ "$PROFILE" == *"tools"* ]]; then
    echo -e "pgAdmin: ${YELLOW}http://localhost:5050${NC}"
fi

echo ""
echo -e "${GREEN}=== Quick Test ===${NC}"
echo "Test notification send:"
echo -e "${YELLOW}curl -X POST http://localhost:8000/api/v1/notifications/send \\
  -H 'Content-Type: application/json' \\
  -d '{
    \"notification\": {
      \"title\": \"Test Notification\",
      \"body\": \"This is a test message\"
    },
    \"destinations\": [
      {\"channel\": 1, \"destination\": \"test@example.com\"},
      {\"channel\": 2, \"destination\": \"device_token_123\"}
    ]
  }'${NC}"

echo ""
echo -e "${GREEN}To view logs:${NC} docker-compose logs -f [service-name]"
echo -e "${GREEN}To stop all services:${NC} ./scripts/docker-down.sh"
