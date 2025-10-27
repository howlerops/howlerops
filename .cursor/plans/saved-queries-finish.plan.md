<!-- c587902c-95dd-4e05-9dfb-8eb04ff0ca52 99a139f3-68d9-4140-bef3-4cfe3a7b00e8 -->
# Saved Queries: FE ↔ Backend Integration Plan

### Current state

- Frontend already has `SavedQueryRepository` (IndexedDB), `useSavedQueriesStore`, `SaveQueryDialog`, and `SavedQueriesPanel`.
- Backend exposes sync endpoints and data types in `backend-go/internal/sync` and routes under `/api/sync`.
- Gaps to close: endpoint param mismatches, conflict resolve path mismatch, and data-model mapping between FE and backend for saved queries; plus wiring the panel into the editor UI.

### Implementation steps

#### 1) Fix Sync HTTP client parameters and routes

- Update `frontend/src/lib/api/sync-client.ts`:
- Include `device_id` in download query params using `SyncService.getDeviceInfo().deviceId`.
- Send `since` as RFC3339 (`Date.toISOString()`), not epoch ms.
- Change conflict resolution endpoint to `POST /api/sync/conflicts/{id}/resolve`.

#### 2) Map saved query models between FE and backend

- In `frontend/src/lib/sync/sync-service.ts`:
- Add mapping helpers:
- local→sync: `{ id, user_id, name: title, description, query: query_text, tags, folder, favorite: is_favorite, created_at, updated_at, sync_version }`.
- sync→local: `{ id, user_id, title: name, description, query_text: query, tags, folder, is_favorite: favorite, created_at, updated_at, synced: true, sync_version }`.
- Use local→sync in `getLocalChanges()` when building `changes.savedQueries`.
- Use sync→local in `mergeRemoteChanges()` when writing to `STORE_NAMES.SAVED_QUERIES`.

#### 3) Ensure upload/download success updates local sync flags

- In `frontend/src/lib/sync/sync-service.ts`:
- After successful upload+merge, local records end up `synced: true` via merge; keep this behavior and increment `sync_version` consistently.

#### 4) Wire Saved Queries Panel in the editor

- In `frontend/src/components/query-editor.tsx`:
- Add a "Saved Queries" button to toggle `SavedQueriesPanel`.
- Pass `user.id` and an `onLoadQuery` handler that loads the selected query into the editor and closes the panel.

#### 5) Guardrails and UX polish

- Respect tier limits already enforced in `SavedQueryRepository` and `useTierStore`.
- Optional: add a keyboard shortcut entry and menu item for opening the Saved Queries panel.

#### 6) Configuration

- Confirm `VITE_API_URL` points to the backend HTTP server in dev/prod.
- Ensure auth header is accepted by backend (FE uses license key as bearer). If needed, align middleware expectations.

#### 7) Validation

- Run full repo validation after changes per project checklist to ensure types, lint, and builds all pass.

### To-dos

- [ ] Fix download params and conflict route in sync-client.ts
- [ ] Map SavedQueryRecord → backend sync shape in sync-service getLocalChanges
- [ ] Map backend saved query → SavedQueryRecord in mergeRemoteChanges
- [ ] Add SavedQueriesPanel toggle and onLoad in query-editor.tsx
- [ ] Add keyboard shortcut/menu item to open saved queries
- [ ] Verify VITE_API_URL and auth header compatibility with backend
- [ ] Run typecheck, lint, tests, and build per checklist