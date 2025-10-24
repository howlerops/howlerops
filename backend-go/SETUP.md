# SQL Studio Backend - Local Development Setup Guide

This guide will help you get the SQL Studio backend running locally in under 5 minutes.

## Prerequisites

- Go 1.24.0 or higher
- Make (optional, but recommended)

## Step-by-Step Setup

### 1. Navigate to Backend Directory

```bash
cd backend-go
```

### 2. Install Dependencies

```bash
go get github.com/joho/godotenv
go mod download
go mod tidy
```

### 3. Setup Local Environment

```bash
make setup-local
```

This command will:
- Create a `./data` directory for your local SQLite database
- Copy `.env.example` to `.env.development` (if it doesn't exist)

### 4. Run Database Migrations

```bash
make migrate-local
```

This creates all the necessary tables in your local SQLite database.

### 5. Start the Server

```bash
make dev
```

The server will start and be available at:
- HTTP API: http://localhost:8080
- gRPC API: localhost:9090
- Metrics: http://localhost:9100/metrics

### 6. Verify Everything Works

Open a new terminal and test the server:

```bash
# Check health endpoint
curl http://localhost:8080/health

# Check metrics
curl http://localhost:9100/metrics
```

If you see responses from both endpoints, you're all set!

## What Just Happened?

1. **Dependencies Installed**: All Go dependencies were downloaded
2. **Environment Created**: A local SQLite database configuration was set up
3. **Database Created**: A local SQLite file was created at `./data/development.db`
4. **Schema Applied**: All tables, indexes, and constraints were created
5. **Server Started**: The backend is now running with:
   - HTTP/REST API on port 8080
   - gRPC API on port 9090
   - Prometheus metrics on port 9100

## Configuration

The local development environment is pre-configured with sensible defaults in `.env.development`:

```bash
# Database - Local SQLite (no external dependencies!)
TURSO_URL=file:./data/development.db
TURSO_AUTH_TOKEN=

# Server ports
SERVER_HTTP_PORT=8080
SERVER_GRPC_PORT=9090
SERVER_METRICS_PORT=9100

# Auth (development only - not secure for production!)
JWT_SECRET=local-dev-secret-key-not-for-production-use-only-32-chars-min
JWT_EXPIRATION=24h

# Logging
LOG_LEVEL=debug
LOG_FORMAT=text
```

You can customize any of these values by editing `.env.development`.

## Common Commands

```bash
# Start the server
make dev

# Run migrations
make migrate-local

# Reset the database (delete all data)
make clean-db

# Reset everything (database + setup)
make reset-local

# Run tests
make test-local

# See all available commands
make help
```

## Troubleshooting

### Port Already in Use

If port 8080 is already in use, edit `.env.development` and change:
```bash
SERVER_HTTP_PORT=8081
```

### Database Locked

If you see "database is locked":
1. Stop all running instances of the server (Ctrl+C)
2. Run `make clean-db` to remove lock files
3. Run `make migrate-local` to recreate the database
4. Run `make dev` to start again

### Missing godotenv

If you see an import error for godotenv:
```bash
go get github.com/joho/godotenv
go mod tidy
```

## Next Steps

Now that your backend is running:

1. **Explore the API**: Check out the README.md for full API documentation
2. **Connect the Frontend**: The frontend expects the backend at http://localhost:8080
3. **Add Test Data**: Create some test users and connections through the API
4. **Try Different Databases**: Configure connections to PostgreSQL, MySQL, etc.

## Production Deployment

When you're ready to deploy to production:

1. Copy `.env.production.example` to `.env.production`
2. Set up a Turso database (see README.md)
3. Configure production environment variables
4. Build the binary: `make build`
5. Deploy to your hosting platform

For detailed production setup, see the main README.md file.

## Need Help?

- **README.md**: Comprehensive documentation
- **GitHub Issues**: Report bugs or request features
- **Discord/Slack**: Join our community (if available)

## What's Next?

The local development environment is now ready. You can:

- Start building features
- Run tests with `make test`
- Profile performance with `make bench`
- Generate protobuf code with `make proto`

Happy coding!
