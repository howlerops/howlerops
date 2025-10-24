# SQL Studio Backend (Go)

A high-performance backend service for SQL Studio, built with Go. Supports local SQLite development and Turso cloud production deployment with advanced features like streaming queries, real-time updates, and comprehensive monitoring.

## Features

### Database Support
- **Turso Cloud** - Distributed SQLite with edge replication (production)
- **Local SQLite** - Fast development with file-based database
- **PostgreSQL** - Full support with connection pooling
- **MySQL/MariaDB** - Optimized for performance
- **ClickHouse** - High-performance analytics database
- **TiDB** - MySQL-compatible distributed SQL database
- **Elasticsearch/OpenSearch** - Full-text search and analytics via SQL API

### Core Capabilities
- **gRPC Services** - High-performance API with Protocol Buffers
- **HTTP Gateway** - RESTful API gateway for web clients
- **Query Streaming** - Handle 1M+ rows efficiently with NDJSON streaming
- **Real-time Updates** - WebSocket to gRPC bridge for live data
- **Connection Pooling** - Intelligent connection management
- **Authentication** - JWT-based auth with session management
- **Rate Limiting** - Protect against abuse with configurable limits
- **Health Monitoring** - Comprehensive health checks and metrics

### Performance Features
- **Concurrent Processing** - Handle 100k+ concurrent connections
- **Optimistic Locking** - Safe concurrent data editing
- **Streaming Responses** - Sub-100ms response times
- **Connection Management** - Efficient resource utilization

## Quick Start (Local Development)

### Prerequisites
- Go 1.24.0 or higher
- Make (optional, but recommended)

### 1. Setup Local Environment

```bash
cd backend-go
make setup-local
```

This will:
- Create the `./data` directory for local SQLite database
- Copy `.env.example` to `.env.development` (if not exists)
- Set up the development environment

### 2. Install Dependencies

```bash
# Install godotenv for environment loading
go get github.com/joho/godotenv
go mod tidy
```

### 3. Configure Environment (Optional)

Edit `.env.development` if you need to customize:

```bash
# Database - defaults to local SQLite
TURSO_URL=file:./data/development.db
TURSO_AUTH_TOKEN=

# Email - optional for local dev
RESEND_API_KEY=
RESEND_FROM_EMAIL=noreply@localhost

# Auth - secure defaults provided
JWT_SECRET=local-dev-secret-key-not-for-production-use-only-32-chars-min
JWT_EXPIRATION=24h

# Ports
SERVER_HTTP_PORT=8080
SERVER_GRPC_PORT=9090
SERVER_METRICS_PORT=9100
```

### 4. Run Database Migrations

```bash
make migrate-local
```

This creates all required tables in the local SQLite database.

### 5. Start the Server

```bash
make dev
```

The server will start with:
- HTTP API: http://localhost:8080
- gRPC API: localhost:9090
- Metrics: http://localhost:9100/metrics

### 6. Verify It's Working

```bash
# Check server health
curl http://localhost:8080/health

# Check metrics
curl http://localhost:9100/metrics
```

### Docker Setup

1. **Start all services**:
```bash
docker-compose up -d
```

2. **View logs**:
```bash
docker-compose logs -f sql-studio-backend
```

3. **Stop services**:
```bash
docker-compose down
```

## Development Commands

```bash
# Local Development
make setup-local      # Setup local environment with SQLite
make dev              # Run server in development mode
make migrate-local    # Run database migrations
make clean-db         # Clean local database
make reset-local      # Reset everything (clean + setup)
make test-local       # Run tests with local database

# Building
make build            # Build the binary
make run              # Run the built binary
make release          # Build release versions for all platforms

# Testing & Quality
make test             # Run all tests
make test-coverage    # Run tests with coverage report
make bench            # Run benchmarks
make lint             # Run linter
make fmt              # Format code
make check            # Run all checks (format, lint, test)

# Code Generation
make proto            # Generate protobuf code

# Utilities
make clean            # Clean build artifacts
make tidy             # Run go mod tidy
make deps             # Download dependencies
make help             # Show all available commands
```

## Configuration

Configuration is managed through environment files with automatic loading based on `ENVIRONMENT`:

### Configuration Priority

Configuration is loaded in this order (later overrides earlier):

1. Default values in code
2. YAML config file (if present)
3. `.env` file (if present)
4. `.env.{ENVIRONMENT}` file (if present)
5. System environment variables

For local development, `.env.development` is automatically loaded when `ENVIRONMENT=development`.

### Environment Variables

#### Required

- `TURSO_URL`: Database URL
  - Local: `file:./data/development.db`
  - Production: `libsql://your-database.turso.io`
- `JWT_SECRET`: Secret key for JWT tokens (min 32 characters)

#### Optional

- `TURSO_AUTH_TOKEN`: Required for Turso cloud, empty for local
- `RESEND_API_KEY`: Resend API key for emails
- `RESEND_FROM_EMAIL`: Email sender address
- `JWT_EXPIRATION`: JWT token expiration (default: 24h for dev, 15m for prod)
- `JWT_REFRESH_EXPIRATION`: Refresh token expiration (default: 30d for dev, 7d for prod)
- `SERVER_HTTP_PORT`: HTTP server port (default: 8080)
- `SERVER_GRPC_PORT`: gRPC server port (default: 9090)
- `SERVER_METRICS_PORT`: Metrics server port (default: 9100)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `LOG_FORMAT`: Log format (text, json)
- `ENVIRONMENT`: Environment name (development, production)

## API Documentation

### gRPC Services

The backend provides the following gRPC services:

#### AuthService
- `Login` - Authenticate users
- `Verify` - Validate tokens
- `Logout` - Terminate sessions
- `GetProfile` - Get user profile
- `RefreshToken` - Refresh access tokens

#### DatabaseService
- `CreateConnection` - Add database connections
- `GetConnection` - Retrieve connection details
- `ListConnections` - List all connections
- `TestConnection` - Test connectivity
- `GetSchemas` - List database schemas
- `GetTables` - List tables in schema
- `GetTableMetadata` - Get table structure

#### QueryService
- `ExecuteQuery` - Run SQL queries
- `ExecuteStreamingQuery` - Stream large result sets
- `CancelQuery` - Cancel running queries
- `GetQueryStatus` - Check query progress
- `ExplainQuery` - Get execution plans

#### TableService
- `GetTableData` - Retrieve table data
- `GetTableDataStream` - Stream table data
- `UpdateTableRow` - Update specific rows
- `InsertTableRow` - Insert new rows
- `DeleteTableRow` - Delete rows
- `CreateTable` - Create new tables
- `AlterTable` - Modify table structure

#### RealtimeService
- `Subscribe` - Subscribe to real-time events
- `Unsubscribe` - Unsubscribe from events
- `PublishEvent` - Publish events

#### HealthService
- `Check` - Health check
- `Watch` - Stream health status
- `GetSystemMetrics` - System metrics
- `GetDatabaseMetrics` - Database metrics

### HTTP Gateway

All gRPC services are also available via HTTP/JSON through the gateway:

```bash
# Health check
curl http://localhost:8080/health

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'

# Execute query
curl -X POST http://localhost:8080/api/queries/execute \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"connection_id": "conn-123", "sql": "SELECT * FROM users LIMIT 10"}'
```

## SQL Studio Database (Turso/SQLite)

### Local Development (SQLite)

The local SQLite database is stored in `./data/development.db`. This file is gitignored.

**Advantages:**
- No external dependencies
- Fast development iteration
- Works offline
- Easy to reset (`make clean-db`)

**Schema:** Automatically created by `make migrate-local`

### Production (Turso Cloud)

Turso is a distributed SQLite database with:
- Global edge replication
- Automatic backups
- Same SQL interface as SQLite
- libSQL protocol for efficient sync

**Setup:**

1. Create a Turso database:
```bash
turso db create sql-studio-prod
```

2. Get the database URL:
```bash
turso db show sql-studio-prod --url
```

3. Create an auth token:
```bash
turso db tokens create sql-studio-prod
```

4. Set environment variables:
```bash
TURSO_URL=libsql://your-database.turso.io
TURSO_AUTH_TOKEN=your_token_here
```

## Database Connections (for users)

SQL Studio supports connecting to various external databases:

### Creating Connections

```json
{
  "name": "Production DB",
  "description": "Main production database",
  "config": {
    "type": "postgresql",
    "host": "localhost",
    "port": 5432,
    "database": "myapp",
    "username": "user",
    "password": "pass",
    "ssl_mode": "require",
    "max_connections": 25
  }
}
```

### Supported Connection Types

#### PostgreSQL
```yaml
type: postgresql
host: localhost
port: 5432
database: mydb
username: user
password: pass
ssl_mode: prefer  # disable, allow, prefer, require
parameters:
  application_name: sql-studio
```

#### MySQL/MariaDB
```yaml
type: mysql
host: localhost
port: 3306
database: mydb
username: user
password: pass
parameters:
  parseTime: "true"
  loc: "UTC"
```

#### SQLite
```yaml
type: sqlite
database: "/path/to/database.db"
parameters:
  cache: shared
  mode: rwc
```

#### ClickHouse
```yaml
type: clickhouse
host: localhost
port: 9000
database: mydb
username: user
password: pass
ssl_mode: disable  # disable, require, skip-verify
parameters:
  dial_timeout: 30s
```

#### TiDB
```yaml
type: tidb
host: localhost
port: 4000
database: mydb
username: user
password: pass
parameters:
  parseTime: "true"
  loc: "UTC"
```

#### Elasticsearch/OpenSearch
```yaml
type: elasticsearch  # or opensearch
host: localhost
port: 9200
database: default  # logical name for the connection
username: elastic  # optional
password: pass     # optional
ssl_mode: disable  # disable, require, skip-verify
parameters:
  api_key: "base64-encoded-key"  # alternative to username/password
```

## Monitoring and Observability

### Metrics
Prometheus metrics are available at `http://localhost:9100/metrics`:

- Connection pool metrics
- Query execution times
- Error rates
- System resource usage

### Health Checks
- `GET /health` - Basic health check
- `GET /api/health/check` - Detailed health status
- gRPC health service for load balancers

### Logging
Structured JSON logging with configurable levels:
- Request/response logging
- Error tracking
- Performance metrics
- Security events

## Security

### Authentication
- JWT-based authentication
- Configurable token expiration
- Refresh token support
- Session management

### Authorization
- Role-based access control
- Method-level permissions
- Connection-level access control

### Security Features
- Rate limiting (per-IP and per-user)
- Request size limits
- CORS configuration
- TLS support
- Input validation

## Performance Tuning

### Connection Pooling
```yaml
database:
  max_connections: 25      # Maximum open connections
  max_idle_connections: 5  # Maximum idle connections
  connection_timeout: 30s  # Connection timeout
  idle_timeout: 5m        # Idle connection timeout
  connection_lifetime: 1h  # Maximum connection lifetime
```

### Query Optimization
- Streaming for large result sets
- Configurable batch sizes
- Query timeout protection
- Execution plan analysis

### Memory Management
- Connection pool limits
- Streaming response chunks
- Garbage collection tuning

## Development

### Building
```bash
# Build binary
make build

# Build with race detection
go build -race -o build/sql-studio-backend cmd/server/main.go

# Cross-compile
make release
```

### Testing
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run benchmarks
make bench
```

### Code Quality
```bash
# Format code
make fmt

# Lint code
make lint

# Run all checks
make check
```

## Troubleshooting

### Database locked error

If you see "database is locked":
- Stop all running instances of the server
- Run `make clean-db` to remove lock files

### Port already in use

Change the ports in `.env.development`:
```bash
SERVER_HTTP_PORT=8081
SERVER_GRPC_PORT=9091
SERVER_METRICS_PORT=9101
```

### JWT secret too short

The JWT secret must be at least 32 characters. Generate a secure one:
```bash
openssl rand -base64 48
```

### Missing godotenv dependency

Install the required dependency:
```bash
go get github.com/joho/godotenv
go mod tidy
```

## Production Deployment

### Environment Setup

1. Copy `.env.production.example` to `.env.production`
2. Fill in all required values with production credentials
3. Ensure `JWT_SECRET` is a secure random string (min 32 chars)
4. Set `TURSO_URL` to your Turso database URL
5. Set `TURSO_AUTH_TOKEN` to your Turso auth token
6. Configure `RESEND_API_KEY` for email service

### Security Checklist

- [ ] Change `JWT_SECRET` to a secure random value
- [ ] Use a production Turso database (not local SQLite)
- [ ] Enable TLS/HTTPS
- [ ] Set `LOG_LEVEL=info` (not debug)
- [ ] Set `LOG_FORMAT=json` for structured logging
- [ ] Configure proper CORS origins (not "*")
- [ ] Enable rate limiting
- [ ] Set up monitoring and alerting
- [ ] Configure database backups

### Running in Production

```bash
# Set environment
export ENVIRONMENT=production

# Run migrations
go run cmd/migrate/main.go

# Start server
./build/sql-studio-backend
```

### Docker

```bash
# Build image
docker build -t sql-studio/backend-go .

# Run container with environment file
docker run -d \
  --name sql-studio-backend \
  -p 8080:8080 \
  -p 9090:9090 \
  -p 9100:9100 \
  --env-file .env.production \
  sql-studio/backend-go:latest
```

### Kubernetes
See `k8s/` directory for Kubernetes manifests.

### Production Considerations
- Use TLS certificates
- Configure proper secrets management
- Set up monitoring and alerting
- Configure backup strategies
- Implement log aggregation

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make check`
6. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

- GitHub Issues: Report bugs and feature requests
- Documentation: See `docs/` directory
- Examples: See `examples/` directory