# HowlerOps Desktop - Wails Application

HowlerOps Desktop is a powerful cross-platform SQL client built with [Wails v2](https://wails.io/), Go, and React. It combines the performance of a native application with the flexibility of modern web technologies.

## Features

- **Multi-Database Support**: PostgreSQL, MySQL, MariaDB, and SQLite
- **Native Desktop Experience**: Built with Wails for true native performance
- **Advanced Query Editor**: Monaco Editor with SQL syntax highlighting and autocomplete
- **Real-time Query Streaming**: Handle large result sets efficiently
- **Connection Management**: Save and manage multiple database connections
- **File Operations**: Open, save, and manage SQL files with recent files tracking
- **Keyboard Shortcuts**: Comprehensive keyboard shortcuts for productivity
- **Cross-Platform**: Available for Windows, macOS, and Linux

## Quick Start

### Prerequisites

- **Go 1.21+**: [Download Go](https://golang.org/dl/)
- **Node.js 18+**: [Download Node.js](https://nodejs.org/)
- **Wails CLI**: Install with `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/sql-studio/sql-studio.git
   cd sql-studio
   ```

2. **Install dependencies**:
   ```bash
   make deps
   ```

3. **Run in development mode**:
   ```bash
   make dev
   ```

4. **Build for production**:
   ```bash
   make build
   ```

### Building from Source

#### Development Build
```bash
# Start development server with hot reload
wails dev

# Or using make
make dev
```

#### Production Build
```bash
# Build for current platform
wails build

# Build for all platforms
make build-all

# Build for specific platform
make build-darwin    # macOS
make build-windows   # Windows
make build-linux     # Linux
```

## Project Structure

```
sql-studio/
├── app.go                 # Main Wails application
├── main.go               # Application entry point
├── wails.json            # Wails configuration
├── go.mod                # Go module definition
├── Makefile.wails        # Build automation
├── backend-go/           # Existing Go backend services
│   ├── internal/
│   │   ├── database/     # Database connection management
│   │   ├── services/     # Business logic services
│   │   └── auth/         # Authentication services
│   └── pkg/              # Shared packages
├── services/             # Wails service wrappers
│   ├── database.go       # Database service wrapper
│   ├── file.go          # File operations service
│   └── keyboard.go      # Keyboard shortcuts service
├── frontend/             # React frontend
│   ├── src/
│   │   ├── components/   # React components
│   │   ├── hooks/        # Custom React hooks
│   │   ├── store/        # State management
│   │   └── generated/    # Generated gRPC types
│   ├── dist/            # Built frontend assets
│   └── package.json     # Frontend dependencies
└── build/               # Build resources
    ├── icons/           # Application icons
    └── windows/         # Windows-specific resources
```

## Configuration

### Wails Configuration (`wails.json`)

The application is configured through `wails.json`. Key settings include:

- **Frontend**: React with Vite
- **Build Mode**: Production optimizations
- **Asset Compression**: Enabled for smaller builds
- **Cross-Platform**: Support for Windows, macOS, and Linux

### Environment Variables

Development environment variables can be set in `.env.development`:

```bash
# Database settings
DB_HOST=localhost
DB_PORT=5432
DB_NAME=sqlstudio
DB_USER=user
DB_PASSWORD=password

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Security
JWT_SECRET=your-secret-key
BCRYPT_COST=12
```

## Development

### Hot Reload Development

```bash
# Start with hot reload
wails dev

# The application will automatically reload when you change:
# - Go source files (app.go, services/, backend-go/)
# - Frontend files (frontend/src/)
```

### Building Frontend

```bash
cd frontend
npm install
npm run build
```

### Building Backend

```bash
# Build Go backend
go build -o build/sql-studio

# Run tests
go test ./...

# Generate protobuf files
cd frontend && npm run proto:build
```

### Testing

```bash
# Run all tests
make test

# Run Go tests only
go test ./...

# Run frontend tests only
cd frontend && npm test

# Run integration tests
make test-integration
```

## Database Support

### Supported Databases

- **PostgreSQL** (9.6+)
- **MySQL** (5.7+)
- **MariaDB** (10.3+)
- **SQLite** (3.8+)

### Connection Examples

#### PostgreSQL
```json
{
  "type": "postgresql",
  "host": "localhost",
  "port": 5432,
  "database": "myapp",
  "username": "user",
  "password": "password",
  "sslMode": "prefer"
}
```

#### MySQL
```json
{
  "type": "mysql",
  "host": "localhost",
  "port": 3306,
  "database": "myapp",
  "username": "user",
  "password": "password"
}
```

#### SQLite
```json
{
  "type": "sqlite",
  "database": "/path/to/database.db"
}
```

## Keyboard Shortcuts

### File Operations
- `Ctrl+N` / `Cmd+N` - New Query
- `Ctrl+O` / `Cmd+O` - Open File
- `Ctrl+S` / `Cmd+S` - Save File
- `Ctrl+Shift+S` / `Cmd+Shift+S` - Save As
- `Ctrl+W` / `Cmd+W` - Close Tab

### Query Operations
- `Ctrl+Enter` / `Cmd+Enter` - Run Query
- `Ctrl+Shift+Enter` / `Cmd+Shift+Enter` - Run Selection
- `Ctrl+E` / `Cmd+E` - Explain Query
- `Ctrl+Shift+F` / `Cmd+Shift+F` - Format Query

### Connection Operations
- `Ctrl+Shift+N` / `Cmd+Shift+N` - New Connection
- `Ctrl+T` / `Cmd+T` - Test Connection
- `Ctrl+R` / `Cmd+R` - Refresh

### View Operations
- `Ctrl+B` / `Cmd+B` - Toggle Sidebar
- `Ctrl+Shift+R` / `Cmd+Shift+R` - Toggle Results Panel
- `Ctrl+=` / `Cmd+=` - Zoom In
- `Ctrl+-` / `Cmd+-` - Zoom Out
- `Ctrl+0` / `Cmd+0` - Reset Zoom

## API Integration

### Backend Services

The desktop application integrates with the existing Go backend services:

- **Database Manager**: Connection pooling and management
- **Query Service**: SQL execution and result handling
- **Auth Service**: User authentication and authorization
- **Health Service**: Connection health monitoring

### Event System

Wails provides a bi-directional event system between Go and JavaScript:

```javascript
// Listen for events from Go
window.wails.Events.On('connection:created', (data) => {
  console.log('New connection created:', data);
});

// Emit events to Go
window.wails.Events.Emit('query:run', {
  connectionId: 'conn-123',
  query: 'SELECT * FROM users'
});
```

## Deployment

### Desktop Distribution

#### macOS
```bash
# Build and create DMG
make build-darwin
# Creates: build/bin/HowlerOps.app

# For distribution, you'll need to sign the app:
# codesign --deep --force --verify --verbose --sign "Developer ID Application: Your Name" "HowlerOps.app"
```

#### Windows
```bash
# Build and create installer
make build-windows
# Creates: build/bin/sql-studio.exe

# For distribution, consider using tools like:
# - NSIS for installer creation
# - Code signing certificates for trust
```

#### Linux
```bash
# Build AppImage
make build-linux
# Creates: build/bin/sql-studio

# For distribution:
# - AppImage (portable)
# - Snap packages
# - Flatpak packages
# - Distribution-specific packages (deb, rpm)
```

### Docker Build

```bash
# Build Docker image
make docker-build

# Run in container
make docker-run
```

## Advanced Configuration

### Custom Build Flags

```bash
# Build with custom tags
wails build -tags "production,mysql,postgres"

# Build with custom ldflags
wails build -ldflags "-X main.version=1.0.0 -X main.build=$(date +%s)"

# Debug build
wails build -debug
```

### Performance Optimization

1. **Enable compression** in `wails.json`
2. **Use production build** for React
3. **Optimize Go build** with appropriate flags
4. **Enable connection pooling** for databases

### Security Considerations

1. **Code signing** for distribution
2. **Secure credential storage** using OS keychain
3. **Input validation** for SQL queries
4. **Connection encryption** (SSL/TLS)

## Troubleshooting

### Common Issues

1. **Build Failures**:
   ```bash
   # Check Go and Node versions
   go version
   node --version

   # Clean and rebuild
   wails clean
   make clean
   make deps
   make build
   ```

2. **Frontend Issues**:
   ```bash
   # Clear frontend cache
   cd frontend
   rm -rf node_modules package-lock.json
   npm install
   npm run build
   ```

3. **Database Connection Issues**:
   - Check firewall settings
   - Verify database credentials
   - Test connection using database client
   - Check SSL/TLS configuration

### Debug Mode

```bash
# Run in debug mode
wails dev -debug

# Enable verbose logging
LOG_LEVEL=debug wails dev
```

### Logs Location

- **macOS**: `~/Library/Logs/HowlerOps/`
- **Windows**: `%APPDATA%\HowlerOps\Logs\`
- **Linux**: `~/.local/share/HowlerOps/logs/`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

### Development Guidelines

- Follow Go best practices
- Use TypeScript for frontend code
- Write tests for new features
- Update documentation
- Follow semantic versioning

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [HowlerOps Docs](https://docs.sqlstudio.app)
- **Issues**: [GitHub Issues](https://github.com/sql-studio/sql-studio/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sql-studio/sql-studio/discussions)
- **Community**: [Discord Server](https://discord.gg/sqlstudio)

## Acknowledgments

- [Wails](https://wails.io/) - Go + Web framework
- [Monaco Editor](https://microsoft.github.io/monaco-editor/) - Code editor
- [React](https://reactjs.org/) - Frontend framework
- [Tailwind CSS](https://tailwindcss.com/) - CSS framework