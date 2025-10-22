# HowlerOps SQL Studio

**A powerful, local-first desktop SQL client with AI-powered features**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

### 🚀 Core Capabilities

- **Multi-Database Support** - Connect to PostgreSQL, MySQL, SQLite, and more
- **Multi-Database Queries** - Query across multiple databases with `@connection.schema.table` syntax
- **AI-Powered SQL Generation** - Generate SQL from natural language
- **Smart Query Suggestions** - Context-aware query completion
- **Query History** - Track all your queries with performance metrics
- **Schema Explorer** - Browse tables, views, and relationships

### 🤖 AI Features

- **Natural Language to SQL** - Describe what you want, get the SQL
- **Query Optimization** - AI-powered query performance tips
- **Error Fixing** - Automatic SQL error detection and fixes
- **Smart Autocomplete** - Context-aware SQL suggestions
- **RAG-Powered Context** - Learns from your schema and past queries

### 💾 Storage

- **Local-First Architecture** - All data stored locally in SQLite
- **Encrypted Credentials** - Database passwords encrypted at rest
- **Offline Capable** - Works completely offline
- **Zero Dependencies** - No external services required
- **Team Mode Ready** - Optional Turso sync for team collaboration (coming soon)

## Quick Start

### Prerequisites

- **Go** 1.21+
- **Node.js** 22.12+ (recommended) or 20.19+
- **Wails CLI** v2.10.2+

### Installation

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

### Local-First Design

```
┌─────────────────────────────────────┐
│         Wails Desktop App           │
├─────────────────────────────────────┤
│                                     │
│  ┌──────────────┐  ┌─────────────┐ │
│  │  Frontend    │  │  Backend    │ │
│  │  React/TS    │◄─┤  Go         │ │
│  └──────────────┘  └─────────────┘ │
│                           │         │
│                           ▼         │
│                   ┌───────────────┐ │
│                   │ Storage Mgr   │ │
│                   └───────────────┘ │
│                           │         │
│        ┌──────────────────┼────────┐│
│        ▼                  ▼        ││
│  ┌──────────┐      ┌──────────┐   ││
│  │local.db  │      │vectors.db│   ││
│  │(Data)    │      │(RAG/AI)  │   ││
│  └──────────┘      └──────────┘   ││
└─────────────────────────────────────┘
```

### Storage Location

All data is stored in `~/.howlerops/`:
- `local.db` - Connections, queries, settings
- `vectors.db` - AI embeddings and RAG data
- `backups/` - Automatic backups
- `logs/` - Application logs

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

- [Storage Architecture](docs/STORAGE_ARCHITECTURE.md)
- [Migration from Qdrant](docs/MIGRATION_FROM_QDRANT.md)
- [AI Setup Guide](docs/AI_SETUP_GUIDE.md)
- [Multi-Database Queries](docs/PART_1_MULTI_DATABASE_QUERY.md)
- [AI/RAG Integration](docs/PART_2_AI_RAG_INTEGRATION.md)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Wails](https://wails.io)
- AI powered by OpenAI, Anthropic, and Ollama
- Storage powered by SQLite
- Vector search powered by sqlite-vec (optional)

## Support

- 🐛 [Report Bugs](https://github.com/yourusername/howlerops/issues)
- 💡 [Request Features](https://github.com/yourusername/howlerops/issues)
- 📖 [Read Docs](./docs/)
- 💬 [Discussions](https://github.com/yourusername/howlerops/discussions)

---

**Made with ❤️ by the HowlerOps team**

