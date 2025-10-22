# Repository Guidelines

## Project Structure & Module Organization
HowlerOps ships as a Wails desktop app. Core Go entrypoints live in `main.go`, `app.go`, and domain services under `services/`. Desktop build artifacts land in `build/`. The React/TypeScript UI resides in `frontend/` (feature modules in `src/components`, `pages`, shared state in `src/store`, helpers in `src/lib`, and generated gRPC stubs in `src/generated/`). Experimental gRPC backends and deployment assets are contained in `backend-go/`. Shared protobuf definitions are stored in `proto/`, while helper scripts and CI hooks live in `scripts/`.

## Build, Test, and Development Commands
Use `make` as the primary interface:
```bash
make deps          # install Go modules, npm packages, and ensure the Wails CLI
make dev           # launch the Wails dev server with hot reload
make build         # produce the desktop bundle in build/bin
make proto         # regenerate TypeScript clients from proto/
make test          # run Go and frontend unit suites
make lint          # execute golangci-lint plus eslint
```
Frontend-only tasks can be driven with `cd frontend && npm run dev|build|test`.

## Coding Style & Naming Conventions
Go code is formatted with `gofmt`/`goimports` (invoke `make fmt-go`); keep package names lowercase and export identifiers in PascalCase. Frontend files follow the TypeScript ESLint presets with 2-space indentation, PascalCase component modules, and kebab-case assets—enforced by `npm run lint` and component-focused tests. Tailwind utilities live alongside components; prefer design tokens over hard-coded colors.

## Testing Guidelines
Run `make test-go` for backend packages (`*_test.go` colocated with source) and keep new logic covered with table-driven cases. Frontend tests live in `frontend/src/__tests__` and use Vitest; add coverage via `npm run test:coverage` and Playwright journeys for UI changes (`npm run test:e2e`). Target the 80% coverage expectation from `CONTRIBUTING.md`, and execute `make validate` before requesting review.

## Task Completion Requirements

**CRITICAL: Before considering any task complete, you MUST run the following validation steps:**

1. **Frontend Validation:**
   ```bash
   cd frontend
   npm run typecheck    # TypeScript type checking
   npm run lint         # ESLint validation
   npm run test:run     # Unit tests
   ```

2. **Backend Validation:**
   ```bash
   go mod tidy          # Clean up Go modules
   go fmt ./...         # Format Go code
   go test ./...        # Run Go tests
   ```

3. **Full Validation:**
   ```bash
   make validate        # Runs lint + test for both frontend and backend
   ```

**Task completion checklist:**
- [ ] All TypeScript types are valid (`npm run typecheck`)
- [ ] Frontend code passes linting (`npm run lint`)
- [ ] Frontend tests pass (`npm run test:run`)
- [ ] Go modules are tidy (`go mod tidy`)
- [ ] Go code is formatted (`go fmt ./...`)
- [ ] Go tests pass (`go test ./...`)
- [ ] Full validation passes (`make validate`)
- [ ] Code compiles successfully (`make build`)

**Never mark a task as complete without running these validation steps.**

## Commit & Pull Request Guidelines
Keep commits focused and written in imperative mood (e.g., `Add connection pooling guard`), optionally prefixed with Conventional Commit types (`feat`, `fix`, `chore`) when they clarify scope. Squash noisy work-in-progress history before review. Pull requests should link the motivating issue, describe the testing performed, and include screenshots or recordings for UI changes. Regenerate protobufs and Wails bindings when interfaces change and mention that step in the PR notes.
