<!-- 546a1202-9d70-4abb-9e97-842640a075f4 4973a571-b401-4ebe-8680-8d651583f1ab -->
# Cloud + Team Features with Turso (libsql) and Cloudflare

## 1) Architecture overview

- **Data layer**: Turso (libsql) main DB. Desktop uses libsql replicating mode (local embedded replica ↔ remote Turso) for offline-first.
- **API layer**: Cloudflare Worker (TypeScript) for auth, teams, access control, webhooks; minimal state; persistence in Turso.
- **Frontend**: Cloud UI on Cloudflare Pages; Wails desktop reuses same auth flows and calls Worker.
- **Hosting**: Cloudflare Pages (FE) + Workers (API). If Turso/Workers connectivity limits arise, fallback is Render (Go API) proxied by Cloudflare.

## 2) Data model and migrations (SQLite/libsql)

- Create `migrations/sqlite/*.sql` for both local and cloud schemas (single source of truth).
- Core tables:
- `users(id, email, name, created_at)`
- `teams(id, name, plan, created_at)`
- `team_members(team_id, user_id, role, joined_at)` with roles: `owner`, `admin`, `editor`, `viewer`
- `connections(id, team_id, name, type, host, port, database, username, options_json, updated_at, updated_by)`
- `connection_secrets(connection_id, secret_type, ciphertext, updated_at, updated_by)`; client-side encrypted
- `permissions(resource_type, resource_id, principal_type, principal_id, action)` for future-proof fine-grained ACLs
- `audit_log(id, actor_user_id, action, resource, resource_id, meta_json, created_at)`
- `sync_clock(id, device_id, lamport, updated_at)`; optional if not relying only on libsql conflict handling
- Indices: per FK + `(team_id, name)` on connections; frequent filters indexed.
- Tooling: use `goose` or `golang-migrate` from `scripts/` with Make targets: `make migrate-up`, `make migrate-down`, and run on both local and cloud.

## 3) AuthN/AuthZ

- **AuthN**: Cloudflare Access (OIDC: GitHub/Google). Worker issues app JWT (short-lived) + refresh token stored httpOnly; desktop uses PKCE flow via embedded browser.
- **AuthZ**: RBAC via `team_members.role` + `permissions` view. Owner/Admin manage team/users, Editor can create/edit connections, Viewer read-only. Enforcement in Worker and in desktop UI guards.
- **Optional login**: If user skips cloud login, desktop runs fully local; features that need cloud are disabled with clear CTA.

## 4) Team/roles semantics (what, where, how)

- **Who can edit connections**: Owner/Admin/Editor on a team. Viewer cannot edit or create.
- **Where edits allowed**: Desktop or Cloud UI. For team-owned connections, edits sync to Turso and propagate to all members.
- **Change visibility**: All team members see latest after replication/refresh; audit trail records editor and changes.

## 5) Sync strategy (desktop ↔ cloud)

- Primary: libsql "replicating" mode for SQLite to Turso (automatic, WAL-based). Handles offline, resumes on reconnect, minimizes custom logic.
- For secrets: encrypt client-side before write; store only ciphertext in Turso; decryption key is device/user-scoped (no server plaintext).
- Conflict policy: last-writer-wins per field; `updated_at` + `updated_by`; present non-destructive history in `audit_log`.
- Real-time: Worker emits pub-sub via Durable Object room or periodic pull (backoff). Desktop subscribes; fallback to 30s pull.

## 6) Backend/API (Cloudflare Worker)

- Endpoints (all scoped by team):
- `POST /auth/callback` (OIDC) → issue app tokens
- `GET /me` → user profile + teams
- `GET/POST /teams` → create/manage team
- `GET/POST/PUT/DELETE /teams/:id/members` → invite, role change, remove
- `GET/POST/PUT/DELETE /teams/:id/connections` → CRUD (server enforces RBAC)
- `GET /teams/:id/audit` → list changes
- `GET /sync/clock` (optional) → observe lamport/etag for polling
- Worker connects to Turso via libsql HTTP driver; secrets provided by Wrangler env.

## 7) Desktop integration (Wails/Go)

- Add libsql replicating client: local file path + remote Turso URL/token.
- Migrations run on local replica at startup; remote migrations run from CI.
- Add cloud session store; PKCE-based login flow; token refresh; team context selector.
- Implement guarded operations in services so local operations respect remote RBAC when online and queue when offline.

## 8) Frontend work (React)

- New feature area `frontend/src/features/cloud/`:
- Auth screens (login, account, team switcher)
- Team management UI (members, roles, invites)
- Connections manager (team-scoped) with role-based editing
- Sync status indicator and conflict banner
- Wire to Worker API; reuse existing components where possible.

## 9) DevOps & deployment

- Cloudflare Pages for FE; CI pipeline builds and deploys on main tags.
- Cloudflare Worker (TypeScript) with Wrangler; staging + prod envs; secrets: `TURSO_DATABASE_URL`, `TURSO_AUTH_TOKEN`, `JWT_SECRET`, OIDC client vars.
- Turso org: staging/prod DBs; migration job from CI on deploy.
- Fallback: If Workers connectivity is blocked, deploy minimal Go API on Render; keep same REST contract; proxy via Cloudflare.

## 10) Security & privacy

- Client-side encryption of connection secrets; rotate keys; passphrase or OS keychain-backed key.
- Principle of least privilege via roles; audit all writes; rate limit Worker endpoints.
- No plaintext DB passwords in cloud by default; allow opt-in per connection.

## 11) Rollout plan

- Phase 1: Schema + migrations + Turso env + Worker skeleton + login.
- Phase 2: Teams + RBAC enforcement + connections CRUD in cloud UI.
- Phase 3: Desktop replication + guarded edits + audit log.
- Phase 4: Real-time updates + conflict UX + perf tuning.
- Phase 5: Hardening, docs, billing gates (future).

## 12) Performance optimizations

- libsql replication to avoid bespoke sync; batch writes; WAL-friendly indices.
- Use `updated_at` filters and ETags for delta fetches; cache-control on read endpoints.
- Index hot queries; avoid N+1; paginate lists; compact audit logs.
- Minimize Worker cold-start by small bundle; use Durable Objects only when needed.

## 13) Compatibility notes

- Offline-first preserved; if not logged in, nothing breaks.
- Current query editor and schema views unchanged; only add cloud-aware guards and team switcher.
- Existing local SQLite continues to work; on login, it becomes the libsql replica of Turso for team-scoped data.

### To-dos

- [ ] Add unified SQLite migrations and Make targets for Turso/local
- [ ] Provision Turso staging/prod DBs and service tokens
- [ ] Create Cloudflare Worker with OIDC login and /me
- [ ] Implement teams, members, permissions, audit tables + indices
- [ ] Implement team CRUD and member management endpoints
- [ ] Implement team-scoped connections CRUD with RBAC checks
- [ ] Implement client-side encryption for connection secrets
- [ ] Integrate libsql replicating client in Wails app
- [ ] Add PKCE login and cloud session management in desktop
- [ ] Add cloud auth, team management, and connections UI
- [ ] Add DO/pub-sub or polling for updates; desktop subscribe
- [ ] Add audit log list and drilldown
- [ ] Tune indices, pagination, ETags; compact audit log
- [ ] Write docs; set up CI deploys for Pages/Workers and migrations