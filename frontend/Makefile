.PHONY: help install dev build lint lint-fix format check test clean

# Default target
help:
	@echo "Available commands:"
	@echo "  make install    - Install dependencies"
	@echo "  make dev        - Start development server"
	@echo "  make build      - Build for production"
	@echo "  make lint       - Run linter (check only)"
	@echo "  make lint-fix   - Run linter and auto-fix issues"
	@echo "  make format     - Alias for lint-fix (auto-fix + sort imports)"
	@echo "  make check      - Run all checks (lint + typecheck)"
	@echo "  make test       - Run tests"
	@echo "  make test-run   - Run tests once (no watch)"
	@echo "  make clean      - Remove build artifacts and node_modules"

# Install dependencies
install:
	npm install

# Development server
dev:
	npm run dev

# Production build
build:
	npm run build

# Linting
lint:
	npm run lint

lint-fix:
	npm run lint:fix

# Format is an alias for lint-fix (removes unused imports, sorts imports, fixes other issues)
format: lint-fix

# Run all checks
check: lint
	npm run typecheck

# Testing
test:
	npm run test

test-run:
	npm run test:run

# Clean
clean:
	rm -rf dist node_modules
