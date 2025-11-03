#!/bin/bash

# Stop script for Teller Notification Service (Podman version)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Stopping Teller Notification Service...${NC}"

# Parse arguments
VOLUMES_FLAG=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --volumes|-v)
            VOLUMES_FLAG="-v"
            echo -e "${YELLOW}Removing volumes...${NC}"
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --volumes, -v  Remove volumes along with containers"
            echo "  --help, -h     Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Stop all containers
echo -e "${YELLOW}Stopping containers...${NC}"
podman-compose down $VOLUMES_FLAG

# Show remaining containers if any
REMAINING=$(podman ps --filter "label=com.docker.compose.project=telltheworld" -q)
if [ ! -z "$REMAINING" ]; then
    echo -e "${YELLOW}Warning: Some containers are still running${NC}"
    podman ps --filter "label=com.docker.compose.project=telltheworld"
else
    echo -e "${GREEN}All Teller services stopped successfully${NC}"
fi

# Show volume status if not removed
if [ -z "$VOLUMES_FLAG" ]; then
    echo ""
    echo -e "${YELLOW}Data volumes are preserved. Use --volumes to remove them.${NC}"
fi