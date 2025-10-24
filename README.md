# HowlerOps SQL Studio

**A powerful, cloud-enabled desktop SQL client with AI-powered features and multi-device sync**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Latest Release](https://img.shields.io/github/v/release/yourusername/sql-studio?include_prereleases)](https://github.com/yourusername/sql-studio/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/yourusername/sql-studio/ci-cd.yml?branch=main)](https://github.com/yourusername/sql-studio/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yourusername/sql-studio?filename=backend-go%2Fgo.mod)](https://github.com/yourusername/sql-studio/blob/main/backend-go/go.mod)
[![codecov](https://codecov.io/gh/yourusername/sql-studio/branch/main/graph/badge.svg)](https://codecov.io/gh/yourusername/sql-studio)
[![Phase 2 Complete](https://img.shields.io/badge/Phase%202-Complete-success)](docs/progress-tracker.md)

## Features

### 🚀 Core Capabilities

- **Multi-Database Support** - Connect to PostgreSQL, MySQL, SQLite, MongoDB, ElasticSearch, and more
- **Multi-Database Queries** - Query across multiple databases with `@connection.schema.table` syntax
- **AI-Powered SQL Generation** - Generate SQL from natural language
- **Smart Query Suggestions** - Context-aware query completion
- **Query History** - Track all your queries with performance metrics
- **Schema Explorer** - Browse tables, views, and relationships

### ☁️ Cloud Sync (Phase 2 - NEW!)

- **Multi-Device Sync** - Access your queries, connections, and history across all devices
- **Automatic Conflict Resolution** - Smart merging with three resolution strategies
- **Offline-First** - Work offline, sync automatically when connected
- **Secure Cloud Storage** - Powered by Turso with edge replication
- **Credential Protection** - Passwords never leave your device
- **Real-Time Updates** - See changes across devices in seconds

### 🔐 Authentication & Security

- **JWT Authentication** - Secure token-based authentication
- **Email Verification** - Verify your account via email
- **Password Reset** - Secure password recovery flow
- **Auto Token Refresh** - Seamless session management
- **Encrypted Storage** - Database credentials encrypted at rest
- **Sanitized Sync** - Sensitive data never synced to cloud

### 🤖 AI Features

- **Natural Language to SQL** - Describe what you want, get the SQL
- **Query Optimization** - AI-powered query performance tips
- **Error Fixing** - Automatic SQL error detection and fixes
- **Smart Autocomplete** - Context-aware SQL suggestions
- **RAG-Powered Context** - Learns from your schema and past queries

### 💾 Hybrid Storage

- **Local-First Architecture** - All data stored locally in SQLite/IndexedDB
- **Cloud Backup** - Optional Turso cloud sync for multi-device access
- **Offline Capable** - Works completely offline
- **Automatic Sync** - Background synchronization when online
- **Conflict Detection** - Intelligent merge strategies for concurrent edits

## Installation

### Quick Install (Recommended)

Install SQL Studio with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
```

**Other options:**

```bash
# Install specific version
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0

# Preview what will be installed (dry-run)
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --dry-run

# Verbose output for troubleshooting
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --verbose
```

**Supported platforms:**
- macOS (Intel & Apple Silicon)
- Linux (x86_64, ARM64, ARM)
- Windows (via Git Bash or WSL)

📖 **[Full installation guide](docs/INSTALLATION.md)** | 📋 **[Quick reference](INSTALL_QUICK_REFERENCE.md)**

### Alternative Methods

- **Homebrew** (coming soon): `brew install sqlstudio/tap/sqlstudio`
- **Direct Download**: [Latest Release](https://github.com/yourusername/sql-studio/releases/latest)
- **Build from Source**: See [Development](#development) section below

For detailed installation instructions, platform-specific guides, and troubleshooting, see [INSTALL.md](INSTALL.md).

## Quick Start

### Prerequisites (Development Only)

- **Go** 1.21+
- **Node.js** 22.12+ (recommended) or 20.19+
- **Wails CLI** v2.10.2+

### Development Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/howlerops-sql-studio.git
cd howlerops-sql-studio

# Install Wails CLI (if not installed)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install dependencies & initialize databases
make deps
make init-local-db

# Start development server
make dev
```

The app will automatically:
1. Initialize local SQLite databases
2. Set up vector storage for AI/RAG
3. Start the development server with hot reload

### First Run

On first launch:

1. **Add a Database Connection**
   - Click "Add Connection"
   - Enter your database details
   - Test connection
   - Save

2. **Configure AI (Optional)**
   - Go to Settings → AI
   - Add your OpenAI, Anthropic, or Ollama credentials
   - AI features will activate automatically

3. **Start Querying!**
   - Write SQL or use natural language
   - Query multiple databases at once
   - Save your favorite queries

## Architecture

### Hybrid Cloud Architecture (Phase 2)

SQL Studio uses a **hybrid local-first + cloud sync** architecture for the best of both worlds:

```
┌─────────────────────────────────────────────────────────┐
│                   Desktop Application                    │
│              (Wails - React + Go Backend)               │
└─────────────────────┬───────────────────────────────────┘
                      │
         ┌────────────┴────────────┐
         │                         │
         ▼                         ▼
  ┌─────────────┐          ┌──────────────┐
  │   Local     │          │    Cloud     │
  │  SQLite DB  │◄────────►│  Turso DB    │
  │  (Primary)  │   Sync   │  (Backup)    │
  └─────────────┘          └──────┬───────┘
                                  │
                     ┌────────────┴────────────┐
                     │                         │
                     ▼                         ▼
              ┌─────────────┐          ┌──────────────┐
              │   Resend    │          │   GitHub     │
              │  (Emails)   │          │  (Releases)  │
              └─────────────┘          └──────────────┘
```

### Key Components

**Frontend (React 19 + TypeScript)**:
- React components with shadcn/ui
- Zustand state management
- IndexedDB for browser storage
- WebSocket for real-time updates

**Backend (Go)**:
- Wails v2 desktop framework
- RESTful API endpoints
- JWT authentication
- gRPC for future services

**Storage Layers**:
- **Local**: SQLite (desktop) or IndexedDB (browser) - primary storage
- **Cloud**: Turso/libSQL database with edge replication
- **Sync Engine**: Bidirectional sync with conflict resolution

**External Services**:
- **Turso**: Cloud database with global edge replication
- **Resend**: Transactional email delivery
- **GitHub Actions**: CI/CD and automated releases

### Data Flow

1. **Local-First**: All operations happen locally first (fast, offline-capable)
2. **Background Sync**: Changes sync to cloud in the background
3. **Conflict Detection**: Automatic detection when devices modify same data
4. **Resolution**: Three strategies (Last Write Wins, Keep Both, User Choice)
5. **Multi-Device**: Changes propagate to other devices in real-time

### Storage Location

**Desktop Application** (`~/.howlerops/`):
- `local.db` - Connections, queries, settings (primary storage)
- `vectors.db` - AI embeddings and RAG data
- `backups/` - Automatic local backups
- `logs/` - Application logs

**Cloud Storage** (Turso):
- Connection metadata (passwords stored locally only)
- Saved queries
- Sanitized query history
- Sync state and conflict resolution

📖 **[Complete Architecture Documentation](ARCHITECTURE.md)**

## Configuration

### Environment Variables

```bash
# Optional: User configuration
export HOWLEROPS_USER_ID=your-user-id
export HOWLEROPS_DATA_DIR=~/.howlerops
export HOWLEROPS_MODE=local  # or 'team' (future)

# Optional: AI provider keys (can also set in UI)
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...
export OLLAMA_ENDPOINT=http://localhost:11434
```

### AI Providers

HowlerOps supports multiple AI providers:

| Provider | Setup | Notes |
|----------|-------|-------|
| **OpenAI** | Add API key in Settings | Best quality, paid |
| **Anthropic (Claude)** | Add API key in Settings | Great for complex SQL |
| **Ollama** | Install locally, auto-detected | Free, runs locally |
| **Claude Code** | Binary path in Settings | Advanced coding model |

## Development

### Project Structure

```
sql-studio/
├── app.go                 # Wails app entry point
├── main.go               # Main application
├── backend-go/
│   ├── pkg/
│   │   ├── database/     # Database connections
│   │   ├── storage/      # Local storage layer
│   │   └── ai/           # AI service wrapper
│   └── internal/
│       ├── ai/           # AI providers
│       └── rag/          # RAG implementation
├── frontend/
│   └── src/
│       ├── components/   # React components
│       ├── pages/        # App pages
│       └── services/     # API clients
├── services/             # Wails services
└── docs/                 # Documentation
```

### Make Commands

```bash
make deps            # Install all dependencies
make init-local-db   # Initialize SQLite databases
make dev             # Start development server
make build           # Build production app
make test            # Run all tests
make test-go         # Run Go tests only
make test-frontend   # Run frontend tests
make backup-local-db # Backup databases
make reset-local-db  # Reset databases (with backup)
make lint            # Run linters
make fmt             # Format code
```

### Testing

```bash
# Run all tests
make test

# CRITICAL: Complete validation before any task
make validate        # Runs lint + test for both frontend and backend

# Frontend validation
cd frontend
npm run typecheck    # TypeScript type checking
npm run lint         # ESLint validation
npm run test:run     # Unit tests

# Backend validation
go mod tidy          # Clean up Go modules
go fmt ./...         # Format Go code
go test ./...        # Run Go tests

# Run specific tests
go test ./backend-go/pkg/storage/...
cd frontend && npm run test

# Run with coverage
go test -cover ./...
```

## Database Management

### Backup & Restore

```bash
# Create backup
make backup-local-db

# Backups are stored in ~/.howlerops/backups/

# Restore from backup
cp ~/.howlerops/backups/local_TIMESTAMP.db ~/.howlerops/local.db
cp ~/.howlerops/backups/vectors_TIMESTAMP.db ~/.howlerops/vectors.db
```

### Reset Databases

```bash
# This will:
# 1. Create automatic backup
# 2. Delete existing databases
# 3. Reinitialize fresh databases
make reset-local-db
```

## AI & RAG

### How It Works

1. **Schema Learning** - AI indexes your database schemas
2. **Query Learning** - Successful queries are embedded and stored
3. **Context Retrieval** - When you ask questions, relevant context is fetched
4. **Smart Generation** - AI generates SQL using your specific schema

### Performance

- Vector search: ~10-50ms (pure Go implementation)
- Query generation: ~1-3s (depends on AI provider)
- Schema indexing: Background, non-blocking

### Optional: SQLite Vector Extension

For faster vector search on large datasets:

```bash
# Install sqlite-vec C extension (optional)
bash scripts/install-sqlite-vec.sh
```

This provides 2-3x faster vector search but is not required.

## Troubleshooting

### Databases Won't Initialize

```bash
# Check permissions
ls -la ~/.howlerops/

# Manually initialize
sqlite3 ~/.howlerops/local.db < backend-go/pkg/storage/migrations/001_init_local_storage.sql
```

### Cloud Sync Not Working

1. **Check authentication**: Click user menu → verify logged in
2. **Check network**: Ensure internet connection is active
3. **View sync status**: Look for sync indicator in header
4. **Check conflicts**: Go to Settings → Sync → View Conflicts
5. **Force sync**: Click "Sync Now" in settings
6. **Check logs**: `tail -f ~/.howlerops/logs/sync.log`

### AI Features Not Working

1. Check AI provider keys in Settings → AI
2. Test provider connection
3. Check logs: `tail -f ~/.howlerops/logs/howlerops.log`

### App Won't Start

```bash
# Check system requirements
make doctor

# Verify Node.js version
node --version  # Should be 20.19+ or 22.12+

# Verify Wails
wails doctor
```

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for development guidelines.

### Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. **CRITICAL: Complete validation checklist:**
   - [ ] All TypeScript types are valid (`npm run typecheck`)
   - [ ] Frontend code passes linting (`npm run lint`)
   - [ ] Frontend tests pass (`npm run test:run`)
   - [ ] Go modules are tidy (`go mod tidy`)
   - [ ] Go code is formatted (`go fmt ./...`)
   - [ ] Go tests pass (`go test ./...`)
   - [ ] Full validation passes (`make validate`)
   - [ ] Code compiles successfully (`make build`)
5. Submit pull request

## Documentation

### User Guides

- [User Guide](docs/USER_GUIDE.md) - Complete user manual (NEW)
- [Installation Guide](INSTALL.md) - Detailed installation instructions for all platforms
- [Quick Start Guide](INSTALL_QUICK_REFERENCE.md) - Get started in 5 minutes
- [AI Setup Guide](docs/AI_SETUP_GUIDE.md) - Configure AI providers
- [Multi-Database Queries](docs/PART_1_MULTI_DATABASE_QUERY.md) - Cross-database queries
- [AI/RAG Integration](docs/PART_2_AI_RAG_INTEGRATION.md) - AI features

### Phase 2 Documentation (Cloud Sync & Auth)

- [Architecture Overview](ARCHITECTURE.md) - System design and components (NEW)
- [API Reference](API_REFERENCE.md) - Complete API documentation (NEW)
- [Cloud Sync Guide](docs/README-TURSO-SYNC.md) - Multi-device synchronization
- [Sync Protocol](docs/sync-protocol.md) - Technical sync implementation
- [Authentication Guide](frontend/AUTH_SYSTEM_DOCUMENTATION.md) - Auth system details
- [Email Service](backend-go/internal/email/) - Email integration

### Developer Guides

- [Contributing Guide](docs/CONTRIBUTING.md) - Development guidelines
- [Storage Architecture](docs/STORAGE_ARCHITECTURE.md) - Database design
- [Turso Integration](backend-go/pkg/storage/turso/README.md) - Cloud storage
- [Backend API](backend-go/API_DOCUMENTATION.md) - Backend endpoints
- [Frontend Integration](backend-go/FRONTEND_INTEGRATION_GUIDE.md) - Frontend setup
- [Deployment Guide](backend-go/DEPLOYMENT.md) - Production deployment
- [Release Process](RELEASE.md) - Creating releases
- [Change Log](CHANGELOG.md) - Version history (NEW)

### Technical Specifications

- [Phase 2 Tech Specs](docs/phase-2-tech-specs.md) - Detailed specifications
- [Phase 2 Timeline](docs/phase-2-timeline.md) - Project timeline
- [Cost Analysis](docs/phase-2-costs.md) - Infrastructure costs
- [Risk Register](docs/phase-2-risks.md) - Risk management
- [Testing Checklist](docs/phase-2-testing.md) - QA procedures

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Wails](https://wails.io) - Go + React desktop framework
- AI powered by OpenAI, Anthropic, and Ollama
- Local storage powered by SQLite
- Cloud storage powered by [Turso](https://turso.tech) - Edge database platform
- Email delivery powered by [Resend](https://resend.com)
- UI components from [shadcn/ui](https://ui.shadcn.com)
- Vector search powered by sqlite-vec (optional)
- CI/CD by GitHub Actions

## Support

- 🐛 [Report Bugs](https://github.com/yourusername/howlerops/issues)
- 💡 [Request Features](https://github.com/yourusername/howlerops/issues)
- 📖 [Read Docs](./docs/)
- 💬 [Discussions](https://github.com/yourusername/howlerops/discussions)

---

**Made with ❤️ by the HowlerOps team**

