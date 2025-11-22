# HowlerOps Frontend

A modern TypeScript/React frontend built with Vite.

## Quick Start

```bash
# Install dependencies
make install

# Start development server
make dev

# Build for production
make build
```

## Development Commands

All common commands are available through the Makefile:

```bash
make help          # Show all available commands
make install       # Install dependencies
make dev           # Start development server
make build         # Build for production
make lint          # Run linter (check only)
make format        # Auto-fix issues and sort imports
make check         # Run all checks (lint + typecheck)
make test          # Run tests in watch mode
make test-run      # Run tests once
make clean         # Remove build artifacts
```

## Code Quality

### Linting and Formatting

The project uses ESLint for linting and auto-formatting:

- **Auto-fix on save**: VSCode is configured to auto-fix issues when you save files
- **Import sorting**: Imports are automatically sorted (standard lib → third-party → local)
- **Unused code removal**: Unused imports and variables are automatically removed
- **TypeScript best practices**: Enforces TypeScript and React best practices

### Running Manually

```bash
# Check for issues
npm run lint

# Auto-fix all issues (removes unused imports, sorts imports, etc.)
npm run lint:fix

# Or use the Makefile
make format
```

### VSCode Setup

The project includes VSCode settings (`.vscode/settings.json`) that:
- Auto-fix on save
- Sort imports automatically
- Remove unused imports
- Enforce TypeScript best practices

To use these settings, just open the project in VSCode. No additional setup needed.

### ESLint Configuration

The project uses:
- **ESLint 9** with flat config format
- **TypeScript ESLint** for TypeScript-specific rules
- **eslint-plugin-unused-imports** for automatic unused import removal
- **eslint-plugin-simple-import-sort** for automatic import sorting
- **eslint-plugin-react-hooks** for React hooks best practices

Configuration is in `eslint.config.js`.

## Tech Stack

- **React 19** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **Tailwind CSS** - Styling
- **TanStack Query** - Data fetching
- **React Router** - Routing
- **Vitest** - Testing
- **Playwright** - E2E testing

## Project Structure

```
src/
├── components/     # React components
├── hooks/          # Custom React hooks
├── lib/            # Utility functions and configurations
├── pages/          # Page components
├── services/       # API services
├── store/          # State management (Zustand)
├── types/          # TypeScript type definitions
└── utils/          # Helper utilities
```

## Available Scripts

### Development
- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build

### Testing
- `npm run test` - Run tests in watch mode
- `npm run test:run` - Run tests once
- `npm run test:ui` - Run tests with UI
- `npm run test:coverage` - Run tests with coverage
- `npm run test:e2e` - Run E2E tests
- `npm run test:e2e:ui` - Run E2E tests with UI

### Quality Checks
- `npm run lint` - Check for lint errors
- `npm run lint:fix` - Auto-fix lint errors
- `npm run typecheck` - Check TypeScript types

### Build Analysis
- `npm run analyze` - Analyze bundle size

## Configuration Files

- `eslint.config.js` - ESLint configuration
- `tsconfig.json` - TypeScript base configuration
- `tsconfig.app.json` - TypeScript app configuration
- `tsconfig.node.json` - TypeScript Node configuration
- `vite.config.ts` - Vite configuration
- `vitest.config.ts` - Vitest configuration
- `playwright.config.ts` - Playwright configuration
- `tailwind.config.js` - Tailwind CSS configuration
- `.vscode/settings.json` - VSCode workspace settings
- `Makefile` - Build commands

## Philosophy

This project follows a "ruthless simplicity" philosophy:
- Minimal abstractions
- Direct integration with libraries
- Focus on working features over perfect architecture
- Code you don't write has no bugs

See the main project documentation for more details on the implementation philosophy.
