# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HowlerOps is a modern desktop SQL client application built with Wails (Go + React). It provides a unified interface for working with multiple database engines, featuring AI-powered query assistance and enterprise-grade security.

## Key Architecture

- **Backend**: Go with Wails framework for desktop application
- **Frontend**: React 19 with TypeScript, Vite, and Tailwind CSS
- **Desktop Framework**: Wails v2 for cross-platform desktop application
- **State Management**: Zustand
- **Testing**: Vitest for unit tests, Playwright for E2E tests
- **Database Support**: PostgreSQL, MySQL, MongoDB, S3 (DuckDB), BigQuery, TiDB, ElasticSearch

## Common Development Commands

### Building and Running
```bash
# Install dependencies and setup
make install          # Install Wails CLI and project dependencies
make deps            # Install all Go and Node dependencies

# Development
make dev             # Start Wails development mode with hot reload (opens desktop app)
make dev-browser     # Start development mode in browser

# Building
make build           # Build desktop application
make build-debug     # Build with debug symbols
make build-mac       # Build macOS universal binary
make build-windows   # Build Windows executable
make build-linux     # Build Linux executable

# Protobuf generation (if needed)
make proto           # Generate protobuf files
```

### Testing
```bash
# Run all tests
make test            # Run both Go and frontend tests

# Backend tests
make test-go         # Run Go tests
make test-coverage   # Generate coverage report

# Frontend tests
cd frontend
npm test             # Run Vitest tests
npm run test:ui      # Run tests with UI
npm run test:coverage # Generate coverage report
npm run test:e2e     # Run Playwright E2E tests
npm run test:component # Run component tests
```

### Code Quality
```bash
# Linting
make lint            # Run all linters
make lint-go         # Run Go linters
make lint-frontend   # Run frontend linters (ESLint)

# Formatting
make fmt             # Format all code
make fmt-go          # Format Go code
make fmt-frontend    # Format frontend code
```

## Project Structure

```
.
├── main.go              # Wails application entry point
├── app.go               # Main application logic and API endpoints
├── wails.json           # Wails configuration
├── services/            # Go backend services
│   ├── database.go      # Database connection management
│   ├── file.go          # File operations service
│   └── keyboard.go      # Keyboard shortcuts service
├── backend-go/          # Legacy backend structure (being migrated)
│   ├── internal/        # Internal packages
│   └── pkg/             # Reusable packages
├── frontend/            # React frontend application
│   ├── src/
│   │   ├── components/  # React components
│   │   ├── hooks/       # Custom React hooks
│   │   ├── pages/       # Page components
│   │   ├── services/    # API services
│   │   ├── store/       # Zustand stores
│   │   ├── types/       # TypeScript types
│   │   └── utils/       # Utility functions
│   ├── vite.config.ts   # Vite configuration
│   └── package.json     # Frontend dependencies
└── proto/               # Protocol buffer definitions
```

## Key Implementation Patterns

### Wails Integration
- The application uses Wails v2 to create a desktop application with Go backend and React frontend
- Communication between frontend and backend happens through Wails runtime APIs
- Menu events are handled via `runtime.EventsEmit` and listened to in the frontend

### Backend Services Pattern
Services in the `services/` directory follow this pattern:
- Each service is instantiated in `app.go` and attached to the App struct
- Services expose methods that can be called from the frontend via Wails bindings
- Database connections are managed through the DatabaseService

### Frontend Architecture
- Uses React 19 with TypeScript for type safety
- Component structure follows atomic design principles
- API calls are made through Wails runtime, not traditional HTTP
- State management uses Zustand for global state
- Styling uses Tailwind CSS with custom components

### Database Connectivity
- Multiple database types are supported through a plugin architecture
- Connection configurations are stored securely
- Query execution happens through the backend DatabaseService
- Results are streamed back to the frontend via WebSocket when needed

## Important Notes

- This is a Wails desktop application, not a traditional web app
- The frontend runs inside a WebView provided by Wails
- Backend API endpoints are exposed through Wails bindings, not HTTP routes
- Development mode opens a desktop window automatically
- Hot reload is enabled in development for both frontend and backend changes
- The application can be built for macOS, Windows, and Linux from a single codebase