<!--
SQL Studio Release Notes Template

Use this template when creating GitHub releases. The release workflow will auto-populate
some sections based on commits and pull requests.

Version Format: v2.1.0 (semantic versioning)
- MAJOR: Breaking changes
- MINOR: New features (backwards compatible)
- PATCH: Bug fixes (backwards compatible)
-->

# SQL Studio Backend v2.1.0

## Overview

Brief description of this release (1-2 sentences).

Example:
> This release introduces multi-database sync capabilities, enhanced authentication with email verification, and significant performance improvements to the storage layer.

## What's New

### Major Features

- **Feature Name**: Description of the feature and its benefits
  - Implementation details (if relevant)
  - Usage example or documentation link

- **Another Feature**: What it does and why it matters
  - Key capabilities
  - Configuration notes

### Enhancements

- Improved performance in X by Y%
- Better error messages for Z
- Enhanced logging for easier debugging
- Updated dependencies for security

### Bug Fixes

- Fixed issue where X would cause Y (#123)
- Resolved race condition in Z component (#456)
- Corrected validation logic for A (#789)

## Breaking Changes

> If this is a MAJOR version bump, list all breaking changes here.
> If there are no breaking changes, write "None" and keep this section.

**None** (for this release)

<!-- Or if there are breaking changes: -->

- **API Change**: The `/api/v1/endpoint` now requires authentication
  - **Migration**: Update client code to include JWT token in headers
  - **Documentation**: See [API_DOCUMENTATION.md](../backend-go/API_DOCUMENTATION.md)

- **Configuration Change**: `DATABASE_URL` environment variable renamed to `TURSO_URL`
  - **Migration**: Update your `.env` files and deployment configs
  - **Backwards Compatibility**: Old variable name supported until v3.0.0

## Installation

### Quick Install

**macOS / Linux:**
```bash
curl -fsSL https://install.sqlstudio.io/install.sh | bash
```

**Windows:**
```powershell
irm https://install.sqlstudio.io/install.ps1 | iex
```

### Direct Download

Choose the binary for your platform:

| Platform | Download | SHA256 Checksum |
|----------|----------|-----------------|
| macOS (Intel) | [sql-studio-2.1.0-darwin-amd64.tar.gz](https://github.com/yourusername/sql-studio/releases/download/v2.1.0/sql-studio-2.1.0-darwin-amd64.tar.gz) | `abc123...` |
| macOS (Apple Silicon) | [sql-studio-2.1.0-darwin-arm64.tar.gz](https://github.com/yourusername/sql-studio/releases/download/v2.1.0/sql-studio-2.1.0-darwin-arm64.tar.gz) | `def456...` |
| Linux (AMD64) | [sql-studio-2.1.0-linux-amd64.tar.gz](https://github.com/yourusername/sql-studio/releases/download/v2.1.0/sql-studio-2.1.0-linux-amd64.tar.gz) | `ghi789...` |
| Linux (ARM64) | [sql-studio-2.1.0-linux-arm64.tar.gz](https://github.com/yourusername/sql-studio/releases/download/v2.1.0/sql-studio-2.1.0-linux-arm64.tar.gz) | `jkl012...` |
| Windows (AMD64) | [sql-studio-2.1.0-windows-amd64.zip](https://github.com/yourusername/sql-studio/releases/download/v2.1.0/sql-studio-2.1.0-windows-amd64.zip) | `mno345...` |

**Verify checksums:**
```bash
# Download checksums file
curl -L -O https://github.com/yourusername/sql-studio/releases/download/v2.1.0/checksums.txt

# Verify (macOS/Linux)
shasum -a 256 -c checksums.txt --ignore-missing
```

### Docker

```bash
docker pull sqlstudio/backend:2.1.0

# Or use latest
docker pull sqlstudio/backend:latest
```

## Upgrading

### From v2.0.x

No breaking changes - drop-in replacement:

```bash
# Stop current instance
killall sql-studio

# Install new version (see Installation above)

# Start
sql-studio
```

### From v1.x

**Important**: This is a major version upgrade with breaking changes.

1. **Backup your data:**
   ```bash
   cp ~/.sql-studio/local.db ~/.sql-studio/local.db.backup
   ```

2. **Update configuration:**
   - Review [MIGRATION_GUIDE.md](../docs/MIGRATION_GUIDE.md) for config changes
   - Update environment variables (see Breaking Changes section)

3. **Run migrations:**
   ```bash
   sql-studio migrate
   ```

4. **Verify:**
   ```bash
   sql-studio --version
   curl http://localhost:8080/health
   ```

## Configuration

### Required Environment Variables

```bash
# Turso Database (required in production)
TURSO_URL=libsql://your-database.turso.io
TURSO_AUTH_TOKEN=your-auth-token

# JWT Secret (required, min 32 characters)
JWT_SECRET=your-secret-key-min-32-chars

# Optional: Email service
RESEND_API_KEY=re_your_api_key
RESEND_FROM_EMAIL=noreply@yourdomain.com

# Optional: Server configuration
HTTP_PORT=8080
GRPC_PORT=9090
LOG_LEVEL=info
```

### New Configuration Options (v2.1.0)

- `SYNC_MAX_UPLOAD_SIZE`: Maximum sync payload size (default: 10MB)
- `SYNC_CONFLICT_STRATEGY`: Conflict resolution strategy (default: `last_write_wins`)
- `SYNC_RETENTION_DAYS`: How long to keep sync history (default: 30 days)

See [Configuration Guide](../backend-go/README.md#configuration) for full details.

## Known Issues

List any known issues or limitations in this release:

- Issue with X on Windows when Y (#999) - workaround available
- Performance degradation with Z under high load (#888) - fix planned for v2.1.1

## Deprecation Notices

Features/APIs that will be removed in future versions:

- **Deprecated in v2.1.0**: `OLD_API_ENDPOINT` will be removed in v3.0.0
  - **Use instead**: `NEW_API_ENDPOINT`
  - **Migration deadline**: Before v3.0.0 release (estimated Q2 2026)

## Performance Improvements

- Query execution: 45% faster on average
- Sync operations: 60% reduction in latency
- Memory usage: 30% reduction for large datasets
- Startup time: 20% faster initialization

## Security Updates

- Updated dependencies with known vulnerabilities
- Enhanced JWT token validation
- Improved rate limiting for authentication endpoints
- Added security headers to HTTP responses

## Documentation

- [Installation Guide](../INSTALL.md)
- [Release Process](../RELEASE.md)
- [API Documentation](../backend-go/API_DOCUMENTATION.md)
- [Deployment Guide](../backend-go/DEPLOYMENT.md)
- [Migration Guide](../docs/MIGRATION_GUIDE.md)

## Full Changelog

**All Changes:** [v2.0.0...v2.1.0](https://github.com/yourusername/sql-studio/compare/v2.0.0...v2.1.0)

### Commits Since Last Release

<!-- Auto-generated by release workflow -->

### Contributors

Thank you to everyone who contributed to this release:

- @contributor1 (#PR1, #PR2)
- @contributor2 (#PR3)
- @contributor3 (#PR4, #PR5, #PR6)

**New Contributors:**
- @new-contributor made their first contribution in #PR7

## Checksums

Complete SHA256 checksums for all release artifacts:

```
abc123def456... sql-studio-2.1.0-darwin-amd64.tar.gz
def456ghi789... sql-studio-2.1.0-darwin-arm64.tar.gz
ghi789jkl012... sql-studio-2.1.0-linux-amd64.tar.gz
jkl012mno345... sql-studio-2.1.0-linux-arm64.tar.gz
mno345pqr678... sql-studio-2.1.0-windows-amd64.zip
```

Full checksums available in [checksums.txt](https://github.com/yourusername/sql-studio/releases/download/v2.1.0/checksums.txt)

## Support

Need help?

- Documentation: https://docs.sqlstudio.io
- GitHub Issues: https://github.com/yourusername/sql-studio/issues
- Discussions: https://github.com/yourusername/sql-studio/discussions
- Email: support@sqlstudio.io

## Next Release

Planning for v2.2.0:

- Feature: Real-time collaboration
- Feature: Advanced query analytics
- Enhancement: Improved WebSocket performance
- See [Roadmap](../docs/ROADMAP.md) for full details

---

**Assets:**

The release includes the following downloadable assets:

- Source code (zip)
- Source code (tar.gz)
- sql-studio-2.1.0-darwin-amd64.tar.gz
- sql-studio-2.1.0-darwin-arm64.tar.gz
- sql-studio-2.1.0-linux-amd64.tar.gz
- sql-studio-2.1.0-linux-arm64.tar.gz
- sql-studio-2.1.0-windows-amd64.zip
- checksums.txt

**Release Date:** October 23, 2025

**Release Manager:** @username
