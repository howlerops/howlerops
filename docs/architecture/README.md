# Architecture Documentation

System architecture, design decisions, and technical specifications for HowlerOps.

## Available Documents

- [Storage Architecture Analysis](storage-analysis.md) - Deep dive into data storage layers and design decisions
- [Test Database Environment](test-database.md) - Docker-based test database setup and usage

## Overview

This section contains architecture documentation that explains how HowlerOps is structured and why certain design decisions were made.

### Storage Architecture

HowlerOps uses a multi-layered storage approach:
- **Turso (LibSQL)**: Cloud SQLite for user data, connections, and metadata
- **Local Storage**: Browser localStorage for UI preferences
- **OS Keychain**: Platform-native secure credential storage
- **In-Memory**: Session caching for performance

### Test Environment

The test database environment provides Docker containers for:
- PostgreSQL
- MySQL
- SQLite
- MariaDB

This allows developers to test database connectivity without requiring production credentials.

## Related Documentation

- [Security Documentation](../security/) - Security architecture and encryption
- [Deployment Documentation](../deployment/) - Production deployment guides
