# Deployment Documentation

Production deployment guides, CI/CD configuration, and migration procedures.

## Available Documents

- [Deployment Overview](overview.md) - Complete guide to deploying HowlerOps to production
- [Migration Deployment](migration-deployment.md) - How database migrations work automatically

## Overview

HowlerOps deploys to Google Cloud Run with automatic database migrations.

### Deployment Process

Every push to `main` triggers:
1. GitHub Actions workflow
2. Docker image build
3. Push to Google Container Registry
4. Deploy to Cloud Run
5. Container startup → automatic migrations
6. Health checks
7. Traffic switch

### Automatic Migrations

**Key Point**: Migrations run automatically on every deployment. No manual intervention required.

The app startup sequence (`cmd/server/main.go`) includes:
```go
turso.InitializeSchema(tursoClient, logger)
turso.RunMigrations(tursoClient, logger)
```

This ensures:
- All pending migrations execute before serving traffic
- Version tracking prevents duplicate migrations
- Transaction safety (all-or-nothing execution)
- Idempotent operations (safe to retry)

### Migration System

- **Tracking**: `schema_migrations` table records applied versions
- **Safety**: Each migration runs in a transaction
- **Idempotency**: Uses `IF NOT EXISTS` clauses
- **Ordering**: Migrations run in version order

See [Migration Deployment](migration-deployment.md) for detailed flow diagrams and troubleshooting.

## CI/CD Configuration

The deployment workflow is defined in:
```
.github/workflows/deploy-cloud-run.yml
```

### Environment Variables

Production environment variables are stored in Google Cloud Secret Manager:
- `TURSO_URL` - Database connection string
- `TURSO_AUTH_TOKEN` - Database authentication
- `JWT_SECRET` - JWT signing key
- Other service credentials

## Related Documentation

- [Architecture Documentation](../architecture/) - System architecture
- [Migration Files](../../backend-go/pkg/storage/turso/migrations/) - SQL migration files
