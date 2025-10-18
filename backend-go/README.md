# HowlerOps Backend (Go + gRPC)

A high-performance, production-ready backend for HowlerOps built with Go and gRPC, supporting multiple database types with advanced features like streaming queries, real-time updates, and comprehensive monitoring.

## Features

### Database Support
- **PostgreSQL** - Full support with connection pooling
- **MySQL/MariaDB** - Optimized for performance
- **SQLite** - Perfect for development and embedded use
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

## Quick Start

### Prerequisites
- Go 1.21+
- Protocol Buffers compiler (`protoc`)
- Docker and Docker Compose (optional)

### Development Setup

1. **Clone and setup**:
```bash
cd backend-go
make setup
```

2. **Generate protobuf code**:
```bash
make proto
```

3. **Run in development mode**:
```bash
make dev
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

## Configuration

Configuration is managed through YAML files and environment variables:

### Configuration File
See `configs/config.yaml` for the complete configuration structure.

### Environment Variables
All configuration options can be overridden with environment variables using the prefix `SQL_STUDIO_`:

```bash
export SQL_STUDIO_SERVER_PORT=8080
export SQL_STUDIO_AUTH_JWT_SECRET="your-secret-key"
export SQL_STUDIO_LOG_LEVEL=debug
```

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

## Database Connections

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

## Deployment

### Docker
```bash
# Build image
docker build -t sql-studio/backend-go .

# Run container
docker run -p 8080:8080 -p 9090:9090 sql-studio/backend-go
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