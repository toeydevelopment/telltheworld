#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Stopping Teller Notification Service...${NC}"

# Parse arguments
CLEAN_VOLUMES=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN_VOLUMES="-v"
            echo -e "${RED}Will remove volumes (data will be lost!)${NC}"
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Stop all services
docker-compose --profile kafka --profile tools --profile scale down $CLEAN_VOLUMES

if [ "$CLEAN_VOLUMES" == "-v" ]; then
    echo -e "${GREEN}All services stopped and volumes removed${NC}"
else
    echo -e "${GREEN}All services stopped (volumes preserved)${NC}"
    echo -e "${YELLOW}To remove volumes, run: ./scripts/docker-down.sh --clean${NC}"
fi