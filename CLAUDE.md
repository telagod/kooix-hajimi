# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with the Go version of Kooix Hajimi.

## Project Overview

Kooix Hajimi is a high-performance GitHub API key discovery tool, completely rewritten in Go from the original Python version. It provides enhanced concurrency, intelligent rate limiting, and a modern web interface for discovering and managing Gemini API keys across GitHub repositories.

## Development Commands

### CI/CD
```bash
# GitHub Actions workflows configured for:
# - Automatic Docker builds on push to main/develop
# - Multi-platform builds (linux/amd64, linux/arm64)  
# - Publishing to GitHub Container Registry (ghcr.io)
# - No additional secrets required (uses GITHUB_TOKEN)

# Image available at: ghcr.io/your-username/kooix-hajimi
```


### Local Development
```bash
# Install dependencies
go mod tidy

# Build all components
./scripts/build.sh all

# Build only binaries
./scripts/build.sh build

# Run server (development)
go run cmd/server/main.go

# Run CLI tool
go run cmd/cli/main.go scan --query "AIzaSy in:file"

# Run with custom config
go run cmd/server/main.go --config configs/config.yaml

# Check dependencies
./scripts/build.sh check
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test -v ./internal/scanner/

# Run integration tests (if available)
go test -tags=integration ./tests/...

# Use build script for comprehensive testing
./scripts/build.sh test
```

### Docker Development
```bash
# Build Docker image locally
docker build -t kooix-hajimi:latest .
# OR use build script
./scripts/build.sh docker

# Pull from GitHub Container Registry
docker pull ghcr.io/your-username/kooix-hajimi:latest

# Run with docker-compose
docker-compose up -d

# View logs
docker-compose logs -f kooix-hajimi

# Deploy locally with build script
./scripts/build.sh deploy

# Stop services
./scripts/build.sh stop
```

### Database Management
```bash
# SQLite development
# Database auto-created at: data/kooix-hajimi.db

# PostgreSQL setup (production)
docker-compose --profile postgres up -d

# Multiple deployment profiles available in deployments/
# - local: Basic SQLite setup
# - production: PostgreSQL + Nginx
# - quick: Fast deployment setup
```

## Architecture Overview

### Core Components

**Main Application (`cmd/`)**:
- `server/main.go`: Web server entry point with graceful shutdown
- `cli/main.go`: Command-line interface with Cobra CLI framework

**Internal Packages (`internal/`)**:
- `config/`: Viper-based configuration system with environment variable support
- `ratelimit/`: Intelligent rate limiting with adaptive algorithms and token rotation
- `github/`: GitHub API client with automatic retry and base64 content decoding
- `scanner/`: Core scanning engine with worker pools and concurrent processing
- `validator/`: Gemini API key validation with batch processing capabilities
- `storage/`: Abstracted storage layer supporting SQLite and PostgreSQL
- `sync/`: External service synchronization (Gemini Balancer, GPT Load Balancer)
- `web/`: Gin-based REST API and WebSocket server

**Public Packages (`pkg/`)**:
- `logger/`: Structured logging with Logrus and file rotation
- `utils/`: Common utilities and helper functions

**Web Interface (`web/`)**:
- `static/`: CSS, JavaScript, and other static assets
- `templates/`: HTML templates for the web interface

### Key Architectural Improvements Over Python Version

**Performance Enhancements**:
- **Concurrent Processing**: Worker pools with configurable goroutine counts (default 20)
- **Smart Rate Limiting**: Adaptive rate limiting based on success rates and historical data
- **Memory Optimization**: Object pooling and efficient memory management
- **Non-blocking I/O**: Async operations with context cancellation support

**Scalability Features**:
- **Database Abstraction**: Easy switching between SQLite and PostgreSQL
- **Horizontal Scaling**: Stateless design supporting multiple instances
- **Resource Management**: Configurable limits and graceful degradation
- **Health Monitoring**: Built-in health checks and metrics collection

**Modern Infrastructure**:
- **REST API**: Full RESTful interface for programmatic access
- **WebSocket Support**: Real-time updates and live monitoring
- **Container-First**: Optimized Docker images and compose configurations
- **Configuration Management**: Environment-based config with validation

## Feature Migration Completeness

### ✅ Core Functionality (100% Complete)

**GitHub Integration**:
- [x] GitHub Code Search API with pagination
- [x] Multiple token rotation and management  
- [x] File content retrieval with base64 decoding fallback
- [x] Repository age filtering and blacklist support
- [x] Incremental scanning with SHA-based deduplication

**Key Discovery & Validation**:
- [x] Regex-based Gemini API key extraction (`AIzaSy[A-Za-z0-9\-_]{33}`)
- [x] Placeholder key filtering (YOUR_, EXAMPLE, TODO, etc.)
- [x] Batch validation with configurable worker pools
- [x] Google Generative AI API validation
- [x] Error classification (invalid, rate_limited, disabled, etc.)

**Rate Limiting & Resilience**:
- [x] Intelligent token state management
- [x] Adaptive rate limiting based on success rates
- [x] Exponential backoff with jitter
- [x] Cooldown period management
- [x] Automatic recovery from rate limits

**Data Management**:
- [x] SQLite storage with WAL mode for development
- [x] PostgreSQL support for production deployments
- [x] Checkpoint system for resumable scans
- [x] Processed query tracking to avoid duplicates
- [x] File-based export compatibility with Python version

### ✅ Enhanced Functionality (Go Improvements)

**Web Interface (New)**:
- [x] Real-time dashboard with WebSocket updates
- [x] Key management interface (view, search, delete)
- [x] Scan control and monitoring
- [x] Live progress tracking and statistics
- [x] System health and performance metrics

**API Interface (New)**:
- [x] RESTful endpoints for all operations
- [x] Pagination support for large datasets
- [x] Search and filtering capabilities
- [x] Real-time status reporting
- [x] Configuration management API

**Monitoring & Observability (Enhanced)**:
- [x] Structured logging with configurable levels and outputs
- [x] Performance metrics and statistics collection
- [x] Error tracking and categorization
- [x] Resource usage monitoring
- [x] Health check endpoints

### ✅ External Integration (100% Complete)

**Gemini Balancer Sync**:
- [x] Automatic key synchronization to external balancer
- [x] Configuration-based API integration
- [x] Error handling and retry logic
- [x] Queue-based async processing
- [x] Duplicate prevention and batch operations

**GPT Load Balancer Sync**:
- [x] Multi-group key distribution
- [x] Group ID caching and management
- [x] Async task monitoring
- [x] Configurable sync intervals
- [x] Fallback error handling

### ✅ Configuration & Deployment (Enhanced)

**Configuration System**:
- [x] YAML-based configuration with validation
- [x] Environment variable override support
- [x] Runtime configuration updates
- [x] Profile-based deployment configs
- [x] Backward compatibility with Python configs

**Container Deployment**:
- [x] Multi-stage Docker builds for optimization
- [x] Docker Compose with service profiles
- [x] Health checks and resource limits
- [x] Volume mounting for persistent data
- [x] Network isolation and security
- [x] GitHub Actions CI/CD for automated builds
- [x] Multi-platform Docker images (AMD64/ARM64)
- [x] GitHub Container Registry integration

**Production Features**:
- [x] Graceful shutdown handling
- [x] Signal-based process management
- [x] Log rotation and retention
- [x] Database connection pooling
- [x] TLS/SSL support preparation

## Migration Mapping

### Python → Go Component Mapping

| Python Component | Go Equivalent | Status |
|-------------------|---------------|---------|
| `app/hajimi_king.py` | `internal/scanner/` | ✅ Complete |
| `common/config.py` | `internal/config/` | ✅ Enhanced |
| `common/Logger.py` | `pkg/logger/` | ✅ Enhanced |
| `utils/github_client.py` | `internal/github/` | ✅ Enhanced |
| `utils/file_manager.py` | `internal/storage/` | ✅ Enhanced |
| `utils/sync_utils.py` | `internal/sync/` | ✅ Complete |
| N/A (new) | `internal/web/` | ✅ New Feature |
| N/A (new) | `internal/ratelimit/` | ✅ Enhanced |
| N/A (new) | `internal/validator/` | ✅ Enhanced |

### Function Migration Status

**Core Functions**:
- [x] `normalize_query()` → `scanner/query.go` (enhanced with better parsing)
- [x] `extract_keys_from_content()` → `scanner/extractor.go` (same regex pattern)
- [x] `should_skip_item()` → `scanner/filter.go` (enhanced with more filters)
- [x] `process_item()` → `scanner/processor.go` (concurrent with worker pools)
- [x] `validate_gemini_key()` → `validator/validator.go` (batch processing)

**Data Management**:
- [x] `Checkpoint` class → `storage/interface.go` (database-backed)
- [x] `FileManager` class → `storage/sqlite.go` (abstracted storage)
- [x] File operations → Database operations (enhanced reliability)

**Integration Functions**:
- [x] `SyncUtils` class → `sync/` package (enhanced error handling)
- [x] Balancer sync → `sync/balancer.go` (improved queueing)
- [x] GPT Load sync → `sync/gptload.go` (multi-group support)

## Important Implementation Details

### Performance Optimizations

**Memory Management**:
- Object pooling for frequently allocated structs
- Streaming processing for large files
- Efficient string operations and minimal allocations
- Garbage collection optimization through proper resource lifecycle

**Concurrency Patterns**:
- Worker pool pattern for bounded parallelism  
- Context-based cancellation for clean shutdowns
- Channel-based communication for goroutine coordination
- Mutex-free designs where possible using channels

**Database Optimizations**:
- Connection pooling with configurable limits
- Prepared statements for repeated queries
- Batch inserts for improved throughput
- Database indexes on commonly queried fields

### Error Handling & Resilience

**Retry Strategies**:
- Exponential backoff with jitter for API calls
- Circuit breaker pattern for external service failures
- Dead letter queues for failed synchronization attempts
- Graceful degradation when services are unavailable

**Data Consistency**:
- Transactional operations for critical data updates
- Checkpoint-based recovery from interruptions
- Idempotent operations for safe retries
- Data validation at input and storage boundaries

### Security Considerations

**Credential Management**:
- Environment variable-based secret injection
- Token rotation and lifecycle management
- Secure logging (credential masking)
- Principle of least privilege for API permissions

**Network Security**:
- TLS/SSL support for external communications
- Request timeout and rate limiting
- Input validation and sanitization
- CORS configuration for web interface

## Environment Configuration

### GitHub Actions Secrets

For automated Docker builds, configure these repository secrets:

**Required (GitHub Container Registry)**:
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

**Workflow Triggers**:
- Push to `main` or `develop` branches → Build and push to GHCR
- Tagged releases (`v*`) → Build and push versioned images
- Pull requests → Build only (no push)

### Required Environment Variables

```bash
# Core Configuration
HAJIMI_GITHUB_TOKENS="token1,token2,token3"  # Required: GitHub API tokens

# Optional Configuration  
HAJIMI_LOG_LEVEL="info"                       # debug, info, warn, error
HAJIMI_WEB_PORT=8080                         # Web server port
HAJIMI_SCANNER_WORKER_COUNT=20               # Concurrent workers
HAJIMI_STORAGE_TYPE="sqlite"                 # sqlite, postgres
HAJIMI_RATE_LIMIT_REQUESTS_PER_MINUTE=30     # Base rate limit
```

### Production Deployment Variables

```bash
# Database (PostgreSQL)
HAJIMI_STORAGE_DSN="postgres://user:pass@host:5432/hajimi_king"

# External Service Integration
HAJIMI_SYNC_GEMINI_BALANCER_URL="https://balancer.example.com"
HAJIMI_SYNC_GEMINI_BALANCER_AUTH="auth_token"
HAJIMI_SYNC_GPT_LOAD_BALANCER_URL="https://gpt-load.example.com"
HAJIMI_SYNC_GPT_LOAD_BALANCER_AUTH="auth_token"

# Monitoring
HAJIMI_LOG_OUTPUT="file"
HAJIMI_LOG_FILENAME="/app/logs/hajimi-king.log"
```

## File Structure and Data Compatibility

### Data Migration from Python Version

**Checkpoint Data**:
- Python pickle files → SQLite database
- Automatic migration script included
- Preserved scan state and progress

**Key Export Formats**:
- Same file naming conventions maintained
- Compatible log format for external tools
- JSON export added for programmatic access

**Configuration Compatibility**:
- Environment variable names prefixed with `HAJIMI_`
- Same configuration semantics with enhanced validation
- Migration guide for deployment updates

## Troubleshooting

### Common Issues

**High Memory Usage**:
- Reduce `SCANNER_WORKER_COUNT` if memory limited
- Enable garbage collection tuning with `GOGC` environment variable
- Use PostgreSQL instead of SQLite for large datasets

**Rate Limiting**:
- Monitor token usage via `/api/stats` endpoint
- Adjust `RATE_LIMIT_REQUESTS_PER_MINUTE` based on available tokens
- Enable adaptive rate limiting for automatic optimization

**Database Errors**:
- Check file permissions for SQLite database
- Verify PostgreSQL connection string and credentials
- Review database migration logs for schema issues

**External Service Sync Failures**:
- Verify network connectivity to external services
- Check authentication credentials and permissions
- Monitor sync queue status via web interface

### Performance Tuning

**Optimal Settings for Different Scales**:

```yaml
# Small Scale (1-5 tokens, development)
scanner:
  worker_count: 5
  batch_size: 50
rate_limit:
  requests_per_minute: 20

# Medium Scale (5-20 tokens, staging)  
scanner:
  worker_count: 20
  batch_size: 100
rate_limit:
  requests_per_minute: 30

# Large Scale (20+ tokens, production)
scanner:
  worker_count: 50
  batch_size: 200
rate_limit:
  requests_per_minute: 50
  adaptive_enabled: true
```

This Go version represents a complete modernization of the Hajimi King project while maintaining 100% feature parity with the Python version and adding significant enhancements for performance, scalability, and usability.