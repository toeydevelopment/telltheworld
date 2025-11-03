# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TellTheWorld - A microservices monorepo built with Go and Kratos framework. Contains multiple services:
- **teller**: Distributed notification service for multi-channel delivery (email, push, SMS, webhook)
- **ecommerce**: E-commerce service (in development)

The project uses a workspace structure (go.work) with shared packages for common functionality.

## Critical Architecture Decisions

### Event Broker: NATS (Default)
- **Primary**: NATS with JetStream for reliable message delivery
- **Alternative**: Redis pub/sub (use `--redis` flag), Kafka (use `--kafka` flag)
- **Consumer naming**: NATS consumers require alphanumeric names - dots/dashes are replaced with underscores in `internal/pubsub/nats_pubsub.go`

### Container Runtime: Podman
- The project uses **Podman** instead of Docker
- Use `podman-compose` for orchestration
- Scripts in `/scripts/` have both Docker and Podman versions

### Database Schema
- Uses GORM with PostgreSQL
- **Important**: AUTO_MIGRATE is disabled due to constraint conflicts
- Database initialized via `deploy/postgres/init.sql`
- Unique constraint: `uni_notifications_notification_id` must exist

## Key Architecture

### Project Structure
- **`apps/`**: Individual microservice applications (teller, ecommerce)
- **`apis/`**: Protocol Buffer definitions and generated code for service APIs
- **`internal/`**: Shared internal packages (wkratos utilities for Kratos framework)
- **`pkg/`**: Shared packages that can be imported by external projects
  - `wpubsub/`: Abstraction for pub/sub systems (NATS, Memory)
  - `wgorm/`: GORM utilities and extensions
  - `wemail/`: Email utilities
  - `wid/`: ID generation utilities
  - `hash/`: Hashing utilities
  - `healthcheck/`: Health check utilities
- **`third_party/`**: External dependencies and proto definitions
- **`deploy/`**: Deployment configurations

### Application Layer Structure (DDD Pattern)
Each app follows this structure:
- **`cmd/`**: Application entry points (server and worker)
- **`internal/biz/`**: Business logic layer (use cases)
- **`internal/data/`**: Data access layer (repositories)
- **`internal/service/`**: Service layer (gRPC/HTTP handlers)
- **`internal/server/`**: Server configuration and middleware
- **`internal/conf/`**: Configuration proto definitions
- **`config/`**: Configuration files

### Technology Stack
- **Framework**: Kratos v2 (Go microservices framework)
- **ORM**: GORM
- **Dependency Injection**: Wire (Google)
- **API**: gRPC with HTTP gateway support
- **Validation**: protoc-gen-validate
- **Configuration**: YAML-based with environment variable override
- **Module Management**: Go workspace (`go.work`) for multi-module monorepo

### Go Workspace Structure
- Root `go.work` file manages multiple modules
- Each app in `apps/` has its own `go.mod`
- Each package in `pkg/` has its own `go.mod`
- Shared `internal/` has its own `go.mod`
- Use `make tidy-modules` to update all modules at once
- Workspace allows local development without publishing packages

## Key Commands

### Root-Level Commands (Multi-App)
```bash
# Generate wire dependencies for all apps
make wire

# Generate proto APIs for all apps
make api

# Generate proto structs for all apps
make proto

# Generate mocks for teller service
make genmocks

# Update all modules
make tidy-modules
```

### Development with Podman
```bash
# Start all services (NATS as default broker)
./scripts/podman-up.sh

# Start with Redis instead
./scripts/podman-up.sh --redis

# Stop services
./scripts/podman-down.sh

# Run tests
./scripts/test-grpc.sh        # Test gRPC endpoints
./scripts/test-podman.sh       # Full integration test

# E-commerce service tests (if applicable)
./scripts/test-ecommerce.sh       # Basic test
./scripts/test-ecommerce-grpc.sh  # gRPC test
./scripts/test-ecommerce-all.sh   # All scenarios
./scripts/test-ecommerce-load.sh  # Load test
./scripts/test-ecommerce-errors.sh # Error scenarios
```

### Building and Code Generation (Per-App Commands)
Navigate to the specific app directory (e.g., `cd apps/teller`) and run:

```bash
# Generate all code (proto, wire, etc.)
make all

# Generate Wire dependencies (after modifying DI)
make wire

# Generate proto APIs (gRPC + HTTP + validation)
make api

# Build binaries
make build

# Run server locally with config
make run

# Run tests (runs all unit and integration tests)
make test

# Generate specific components
make grpc      # Only gRPC code
make http      # Only HTTP code
make swagger   # API documentation
make errors    # Error definitions
make config    # Generate internal proto configs
make proto     # Generate internal proto structs
```

### Unit Testing (Teller Service)
```bash
cd apps/teller

# Run all tests with verbose output
go test ./internal/biz -v

# Run with coverage
go test ./internal/biz -v -cover

# Generate coverage report
go test ./internal/biz -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific test
go test ./internal/biz -v -run TestNotification_SendToUser

# Regenerate mocks (from root directory)
make genmocks
```

### Container Operations
```bash
# View logs
podman logs teller-server
podman logs teller-worker-1

# Database access
podman exec teller-postgres psql -U postgres -d telltheworld

# Check NATS health
curl http://localhost:8222/healthz

# Monitor NATS JetStream
curl http://localhost:8222/jsz | jq
```

## Service Architecture

### Core Services & Ports
- **PostgreSQL**: 5432 (database)
- **NATS**: 4222 (client), 8222 (monitoring)
- **gRPC API**: 9000 (only working endpoint currently)
- **HTTP API**: 8000 (not implemented yet)
- **Workers**: 2 instances by default

### Notification Flow
1. Client sends notification via gRPC to `:9000`
2. Server validates and saves to PostgreSQL with status `pending`
3. Server publishes to NATS stream `TELLER.notifications.pending`
4. Worker subscribes via consumer `TELLER_notifications_pending_consumer`
5. Worker claims notification atomically (UPDATE RETURNING)
6. Worker processes and updates status to `sent`

### Worker Processing Pattern
- Uses atomic DB operations to prevent duplicate processing
- Implements stuck notification cleanup (5-minute threshold)
- Each worker has unique ID format: `{container_id}-{random_suffix}`
- Failed notifications are retried up to 5 times

## Configuration Structure

### Environment Variables
```bash
# PubSub selection
PUBSUB_TYPE=nats  # Options: nats, redis, kafka
PUBSUB_NATS_URL=nats://nats:4222

# Database (container names, not localhost)
DATA_DATABASE_HOST=postgres  # NOT localhost
DATA_DATABASE_PORT=5432

# Worker settings
WORKER_STUCK_THRESHOLD=5m
WORKER_MAX_ATTEMPTS=5
```

### Config Files
- `apps/teller/config/config.yaml` - Main configuration
- Container networking requires service names (postgres, nats) not localhost

## Testing Strategy

### gRPC Testing
```bash
# Send single notification
grpcurl -plaintext -d '{
  "notification": {
    "title": "Test",
    "body": "Message",
    "ref_id": "test-001"  # Required field
  },
  "destinations": [
    {"channel": 1, "destination": "test@example.com"}
  ]
}' localhost:9000 teller.v1.NotificationService/SendToUser

# Bulk send
grpcurl -plaintext -d '{
  "requests": [...]
}' localhost:9000 teller.v1.NotificationService/BulkSendToUser
```

### Database Verification
```sql
-- Check notification status
SELECT status, COUNT(*) FROM notifications GROUP BY status;

-- View stuck notifications
SELECT * FROM notifications
WHERE status = 'processing'
AND processing_started_at < NOW() - INTERVAL '5 minutes';

-- Worker distribution
SELECT processing_worker_id, COUNT(*)
FROM notifications
WHERE status = 'processing'
GROUP BY processing_worker_id;
```

## Common Issues and Solutions

### NATS Consumer Creation Fails
- **Error**: "invalid consumer name"
- **Cause**: Consumer names cannot contain dots or dashes
- **Solution**: Already fixed in `internal/pubsub/nats_pubsub.go` with sanitization

### Database Connection Refused
- **Cause**: Config uses `localhost` instead of container service name
- **Fix**: Ensure config uses `postgres` not `localhost`

### Workers Not Processing
- **Check**: Worker logs for subscription errors
- **Verify**: NATS stream exists: `curl http://localhost:8222/jsz`
- **Reset**: Update stuck notifications to `pending` status

### Container Build Issues
- **Golang version**: Uses 1.25-alpine
- **Go modules**: Uses go.work for workspace management
- **COPY paths**: Fixed to use `go.work` instead of root `go.mod`

## Project-Specific Patterns

### Wire Dependency Injection
- Define interfaces in `biz` layer
- Implement in `data` layer
- Wire binds implementations in `cmd/*/wire.go`
- Run `make wire` after changes

### Proto Generation Flow
1. Define `.proto` in `apis/teller/v1/`
2. Run `make api` to generate Go code
3. Implement service in `internal/service/`
4. Register in `internal/server/grpc.go`

### GORM Conventions
- Uses `deleted_at` for soft deletes
- Timestamps: `created_at`, `updated_at`
- UUID primary keys for notifications
- Enum fields use int32 (channel, status)

### PubSub Abstraction
- Shared package: `pkg/wpubsub/` (extracted from internal)
- Interface defined in `pkg/wpubsub/pubsub.go`
- Implementations: NATS, Memory (more can be added)
- Used by individual apps via dependency injection
- Message format includes metadata headers

### Testing Strategy
- **Unit Tests**: All business logic (biz layer) has comprehensive unit tests
- **Table-Driven Tests**: Uses standard Go table-driven test pattern
- **Mocking**: Uses `go.uber.org/mock/gomock` for mock generation
- **Test Package**: Tests use `biz_test` package to avoid import cycles
- **Coverage**: Target is comprehensive coverage of exported functions
- Mocks are auto-generated via `make genmocks` from root directory

## Directory-Specific Notes

### `/apps/`
Each app is self-contained with its own:
- Server and worker binaries (in `cmd/`)
- Configuration files (in `config/`)
- Dockerfiles for containerization
- All apps share the same Makefile structure (`app_makefile`)

### `/apps/teller/`
- Notification service application
- Contains both server and worker binaries
- Comprehensive unit tests in `internal/biz/*_test.go`
- Mocks generated in `internal/biz/*_mock_test.go`

### `/apps/ecommerce/`
- E-commerce service (in development)
- Follows same structure as teller
- Has dedicated test scripts in `/scripts/test-ecommerce*.sh`

### `/apis/`
- Proto definitions for gRPC/HTTP APIs per service
- Generated code lives here (`.pb.go`, `.pb.gw.go` files)
- **NEVER edit generated files manually** - regenerate via `make api`

### `/internal/`
- Shared internal packages (cannot be imported by external projects)
- `wkratos/`: Utilities and extensions for Kratos framework

### `/pkg/`
- Shared packages that CAN be imported by external projects
- `wpubsub/`: Pub/sub abstraction layer
- `wgorm/`, `wemail/`, `wid/`, etc.: Domain utilities
- Each has its own `go.mod` for independent versioning

### `/scripts/`
- Operational scripts for Docker/Podman
- Test scripts for different scenarios and services
- All scripts support both Docker and Podman