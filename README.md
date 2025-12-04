# URL Shortening Service

A high-performance URL shortening service built with Go, designed to handle 1,000+ requests per second with sub-200ms latency. The service provides a REST API for URL shortening and a high-speed redirect service.

## Overview

This system consists of two main services:

- **shortening-service**: Accepts long URLs, validates them, generates unique short codes, and persists mappings to PostgreSQL
- **redirect-service**: Handles high-speed redirects from short codes to original URLs with Redis caching

## Architecture

The system is built on Google Cloud Platform (GCP) with the following components:

- **Backend**: Go 1.21+ with Chi v5 router
- **Database**: PostgreSQL 15 (Cloud SQL with HA)
- **Cache**: Redis 7.0 (Memorystore)
- **Runtime**: Cloud Run (serverless containers)
- **Load Balancing**: Cloud Load Balancer with HTTPS

## Project Structure

```
.
├── cmd/
│   ├── shortening-service/    # Shortening service entry point
│   └── redirect-service/      # Redirect service entry point
├── internal/
│   ├── shortcode/             # Short code generation logic
│   ├── storage/               # Database layer
│   ├── cache/                 # Redis cache layer
│   ├── middleware/            # HTTP middleware (rate limiting, logging)
│   └── validator/             # URL validation logic
├── pkg/
│   └── api/v1/                # API models and interfaces
├── configs/                   # Configuration files
├── deployments/
│   └── cloud-run/             # Cloud Run deployment configs
├── migrations/                # Database migrations
├── scripts/                   # Build and deployment scripts
└── go.mod                     # Go module definition
```

## Features

### URL Shortening (shortening-service)
- Accepts URLs via REST API (POST /api/v1/shorten)
- Validates URLs (http/https only, 10-2048 characters)
- Generates unique short codes using Snowflake-inspired algorithm
- Stores mappings in PostgreSQL with high availability
- Rate limiting: 100 requests/minute per IP, 1,000 req/s globally
- P99 latency < 200ms

### URL Redirect (redirect-service)
- High-speed redirects via short codes (GET /{shortCode})
- Redis caching with 95%+ cache hit rate
- Falls back to PostgreSQL read replica on cache miss
- P99 latency < 50ms
- Handles 10,000+ req/s

## Tech Stack

- **Language**: Go 1.21+
- **Router**: Chi v5
- **Database Driver**: pgx/v5
- **Redis Client**: go-redis/v9
- **Logging**: zerolog
- **Metrics**: Prometheus client

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (for local development)
- PostgreSQL 15
- Redis 7.0
- GCP account (for production deployment)

## Quick Start

### Local Development

1. Clone the repository:
```bash
git clone https://github.com/metaagenticai/shortening-service.git
cd shortening-service
```

2. Install dependencies:
```bash
go mod download
```

3. Start local infrastructure (PostgreSQL + Redis):
```bash
docker-compose up -d
```

4. Run database migrations:
```bash
make migrate-up
```

5. Start the shortening service:
```bash
make run-shortening
```

6. Start the redirect service:
```bash
make run-redirect
```

### Testing

Run unit tests:
```bash
make test
```

Run integration tests:
```bash
make test-integration
```

Run with coverage:
```bash
make test-coverage
```

### Building

Build both services:
```bash
make build
```

Build Docker images:
```bash
make docker-build
```

## Configuration

Configuration is managed through environment variables:

### Shortening Service

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | HTTP port | 8080 |
| DATABASE_URL | PostgreSQL connection string | - |
| REDIS_URL | Redis connection string | - |
| SHORT_DOMAIN | Short URL domain | short.link |
| LOG_LEVEL | Logging level | info |
| RATE_LIMIT_PER_IP | Requests per minute per IP | 100 |

### Redirect Service

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | HTTP port | 8081 |
| DATABASE_URL | PostgreSQL connection string (read replica) | - |
| REDIS_URL | Redis connection string | - |
| CACHE_TTL | Cache TTL in seconds | 3600 |
| LOG_LEVEL | Logging level | info |

## API Documentation

### Shorten URL

**Endpoint**: `POST /api/v1/shorten`

**Request**:
```json
{
  "url": "https://example.com/very/long/url/path?query=params"
}
```

**Response** (201 Created):
```json
{
  "short_code": "aB3xY9",
  "short_url": "https://short.link/aB3xY9",
  "long_url": "https://example.com/very/long/url/path?query=params",
  "created_at": "2024-01-15T10:30:00.123Z"
}
```

**Error Response** (400 Bad Request):
```json
{
  "error": "invalid_url",
  "message": "URL must start with http:// or https://",
  "details": {
    "field": "url",
    "provided": "ftp://example.com"
  }
}
```

### Redirect

**Endpoint**: `GET /{shortCode}`

**Response**: HTTP 301 redirect to original URL

**Error Response** (404 Not Found):
```json
{
  "error": "not_found",
  "message": "Short code not found"
}
```

### Health Check

**Endpoint**: `GET /health`

**Response** (200 OK):
```json
{
  "status": "healthy",
  "service": "shortening-service",
  "version": "1.0.0",
  "timestamp": "2024-01-15T10:30:00.123Z",
  "checks": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

## Database Schema

```sql
CREATE TABLE url_mappings (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(8) UNIQUE NOT NULL,
    long_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    created_by_ip INET,

    CONSTRAINT short_code_length CHECK (char_length(short_code) BETWEEN 6 AND 8),
    CONSTRAINT long_url_length CHECK (char_length(long_url) BETWEEN 10 AND 2048),
    CONSTRAINT long_url_scheme CHECK (long_url LIKE 'http://%' OR long_url LIKE 'https://%')
);

CREATE UNIQUE INDEX idx_short_code ON url_mappings(short_code);
CREATE INDEX idx_created_at ON url_mappings(created_at DESC);
CREATE INDEX idx_created_by_ip ON url_mappings(created_by_ip) WHERE created_by_ip IS NOT NULL;
```

## Short Code Generation

The service uses a Snowflake-inspired algorithm to generate unique 63-bit IDs:

- **41 bits**: Timestamp (milliseconds since custom epoch)
- **10 bits**: Worker ID (derived from Cloud Run instance)
- **12 bits**: Sequence number (incremented per millisecond)

IDs are then encoded to Base62 (0-9a-zA-Z) resulting in 6-8 character strings.

**Capacity**: 4.2 billion IDs per second across 1,024 workers.

## Deployment

### Google Cloud Run

1. Build and push Docker image:
```bash
make docker-build
make docker-push
```

2. Deploy to Cloud Run:
```bash
make deploy-staging  # Deploy to staging
make deploy-prod     # Deploy to production
```

### Infrastructure Setup

See `deployments/cloud-run/README.md` for detailed infrastructure setup instructions.

## Monitoring

The service exposes Prometheus metrics at `/metrics`:

- `shortening_requests_total` - Total shortening requests by status
- `shortening_latency_seconds` - Request latency distribution
- `redirect_requests_total` - Total redirect requests
- `cache_hit_rate` - Redis cache hit rate
- `db_connections_active` - Active database connections

## Performance

### Benchmarks

- **Shortening**: 1,000 req/s sustained, P99 < 200ms
- **Redirect**: 10,000 req/s sustained, P99 < 50ms
- **Cache Hit Rate**: 95%+ for steady-state traffic

### Load Testing

```bash
make load-test-shortening
make load-test-redirect
```

## Security

- HTTPS enforced for all endpoints
- Input validation and sanitization
- Rate limiting per IP address
- SQL injection prevention via parameterized queries
- No storage of sensitive information

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Support

For issues and questions, please open an issue in the GitHub repository.
