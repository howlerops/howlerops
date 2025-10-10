## Recent Changes

- Fixed the results grid's render loop by de-coupling `EditableTable`'s internal state from the parent store.
- Normalised query rows with stable `__rowId` values so diffing and persistence are predictable.
- Hardened Postgres editable metadata (every column now exposes `ResultName`).
- Cached schema introspection per connection session; refreshes invalidate the cache on demand.
- Replaced Monaco's static completions with a schema-aware provider (tables after `FROM/JOIN`, columns after alias + dot).
- Improved save/export logic to use the editor's current data instead of stale copies.
- Added a draggable splitter on the dashboard so the results panel can expand beyond its original 50% height.
- Ensured the results footer (row counts / pending changes) stays visible while the table body scrolls within the panel.
- Removed implicit TanStack pagination so the full result set is rendered and available for scrolling/export.
- Tweaked the virtual scroller (larger overscan + dynamic row measurement) for smoother scrolling on very large datasets.

## Follow-up / TODO

- ✅ **COMPLETED**: Consider persisting the schema cache across app restarts if performance becomes a concern.
  - Implemented persistent schema cache with localStorage, 24-hour expiry, and automatic cleanup
- ✅ **COMPLETED**: Tie the default `EditableTable` state-clearing to events instead of `resetTable()` once the websocket editor lands, to avoid future API drift.
  - Implemented event-based table state management with TableEventManager
  - Added event listeners for table reset, data refresh, and query changes
- ✅ **COMPLETED**: Once join editing is supported in the backend, revisit the client guardrails (currently queries with joins remain read-only).
  - Updated PostgreSQL backend to support JOIN query editing (edits the main table in JOIN)
  - Frontend automatically supports JOIN editing through backend metadata
