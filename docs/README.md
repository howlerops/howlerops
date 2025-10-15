# HowlerOps

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18+-61DAFB.svg)](https://reactjs.org/)

A modern, high-performance HowlerOps application for developers and data professionals. HowlerOps provides a unified interface for working with multiple database engines, featuring AI-powered query assistance and enterprise-grade security.

## Features

### 🗄️ Multi-Database Support
- PostgreSQL
- MySQL
- MongoDB
- S3 (via DuckDB)
- BigQuery
- TiDB
- ElasticSearch

### 🤖 AI-Powered Capabilities
- Natural language to SQL conversion
- Query optimization suggestions
- Intelligent auto-completion
- Multiple AI provider support:
  - OpenAI (GPT-4, GPT-4o)
  - Anthropic (Claude 3.5)
  - Claude Code (CLI-based local assistance)
  - OpenAI Codex
  - Ollama (local LLMs)
  - Hugging Face models

### 🔒 Enterprise Security
- End-to-end encryption
- System keyring integration
- Audit logging
- Role-based access control

### ⚡ Performance
- Built with Go for optimal performance
- WebSocket support for real-time query streaming
- Connection pooling and caching
- < 100ms query execution overhead

## Installation

### Using Homebrew (macOS/Linux)
```bash
brew install sql-studio
```

### Using Docker
```bash
docker run -p 8080:8080 sqlstudio/sql-studio:latest
```

### From Source
```bash
git clone https://github.com/yourusername/sql-studio.git
cd sql-studio
make build
./bin/sql-studio
```

## Quick Start

1. **Start the application:**
   ```bash
   sql-studio start
   ```

2. **Open the web interface:**
   Navigate to http://localhost:8080

3. **Add a database connection:**
   Click "New Connection" and provide your database credentials

4. **Start querying:**
   Use the SQL editor or natural language interface to query your data

## Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- Wails v2 CLI
- Make

### Building from Source
```bash
# Clone the repository
git clone https://github.com/yourusername/sql-studio.git
cd sql-studio

# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install dependencies
make deps

# Build the desktop application
make build

# Build for specific platforms
make build-mac     # macOS universal binary
make build-windows # Windows executable
make build-linux   # Linux executable

# Run tests
make test

# Start development server (opens desktop app)
make dev

# Start development in browser mode
make dev-browser
```

## Architecture

HowlerOps uses a modern desktop-first architecture designed for performance and extensibility:

- **Desktop Framework**: Wails v2 for cross-platform native desktop application
- **Backend**: Go with Wails runtime bindings for direct frontend-backend communication
- **Frontend**: React 19 with TypeScript, Vite, and Tailwind CSS
- **Storage**: Encrypted SQLite for connection management
- **Communication**: WebSocket for real-time features, Wails runtime for IPC
- **AI**: Pluggable architecture supporting multiple providers with native integration

For detailed architecture information, see [TECHNICAL_ARCHITECTURE.md](TECHNICAL_ARCHITECTURE.md)

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Roadmap

See [IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) for our detailed development plan.

## Documentation

- [Technical Architecture](TECHNICAL_ARCHITECTURE.md)
- [Implementation Roadmap](IMPLEMENTATION_ROADMAP.md)
- [API Documentation](docs/API.md)
- [Plugin Development Guide](docs/PLUGINS.md)
- [Security Guide](docs/SECURITY.md)

## Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/sql-studio/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/sql-studio/discussions)
- **Discord**: [Join our community](https://discord.gg/sql-studio)

## License

HowlerOps is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Acknowledgments

HowlerOps builds upon the excellent work of many open-source projects. Special thanks to:
- The Go community for excellent database drivers
- DuckDB for enabling SQL on S3
- The React ecosystem for modern UI capabilities

---

**Status**: Pre-Alpha Development

*HowlerOps is actively under development. Join us in building the future of database administration tools!*