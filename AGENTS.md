# Repository Guidelines

## Project Structure & Module Organization
Core Wails code sits at the repository root (`main.go`, `app.go`, `services/`, `proto/`, `wails.json`). React UI assets live in `frontend/`, with generated bindings under `frontend/wailsjs/`. Supporting Go utilities live in `backend-go/` (`internal/`, `pkg/`). Automation, database bootstrap logic, and CI helpers sit in `scripts/`, while architecture notes stay in `docs/`. Build artifacts land in `build/bin/` and should remain untracked.

## Build, Test, and Development Commands
`make deps` syncs Go modules and npm packages after dependency changes. `make dev` starts the Vite dev server and Wails backend with hot reload. `make build` emits a production desktop bundle in `build/bin/`. `make test` runs Go coverage (`go test -cover ./...`) plus frontend linting, type-checking, and Vitest. Run `make lint` and `make validate` before a PR. Regenerate protobuf bindings with `make proto` when editing `proto/`.

## Coding Style & Naming Conventions
Format Go code with `go fmt`/`goimports` (`make fmt-go`); exported names stay PascalCase and package helpers camelCase. Keep Go packages focused per domain and mirror feature boundaries seen in `frontend/`. For TypeScript, rely on ESLint and the strict tsconfig; follow the prevailing 2-space indentation. Place UI modules in `frontend/src/<feature>/` using kebab-case filenames and colocate tests or hooks with their components.

## Testing Guidelines
Target ≥80% coverage for new logic. Add Go tests in `_test.go` files near the code under test and favor table-driven assertions. Frontend unit specs live as `.test.tsx`; integration variants use `.integration.test.tsx`. Run `npm run test:run` inside `frontend/` for watch mode, `npm run test:coverage` when tweaking analytics, and Playwright (`npm run test:e2e`) for workflow changes. Always run `make test` before pushing.

## Commit & Pull Request Guidelines
Use short, imperative commit subjects; prefixes such as `feat:` and `fix:` appear in history and streamline changelog tooling. Keep each commit focused and update generated assets (proto, icons, SQL) alongside the source. Pull requests need a concise summary, linked issues, screenshots or GIFs for UI changes, and a checklist of local commands (`make lint`, `make test`, `make proto`). Confirm CI is green before requesting review.

## Security & Configuration Tips
Never commit `.env.*`; copy from `.env.example` for local testing. Local SQLite state lives under `~/.howlerops/`; run `make init-local-db` to bootstrap and `make backup-local-db` before destructive experiments. When enabling optional AI providers, document new environment variables in `docs/` and load secrets through the settings UI rather than hard-coded values.
