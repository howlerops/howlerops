# Storage Architecture Analysis & Recommendations

## 🎯 Ultra-Deep Dive: Where Data Lives

After analyzing your codebase, here's the complete data storage landscape:

---

## Current Storage Layers

### 1. **Zustand** (In-Memory State Management)
**What it stores:**
- Current UI state (navigation, tabs, panels)
- Active connections (runtime only)
- Current query results (while displayed)
- Form state
- Loading states

**Persistence:**
- `auth-store.ts` uses `persist()` → Saves to **localStorage**
- Most other stores: **NO persistence** (memory only)

**When it's used:**
- Real-time UI interactions
- Fast state updates
- Cross-component communication

---

### 2. **IndexedDB** (Browser Storage)
**Currently used by:**
- `query-result-storage.ts` - Query result caching (large datasets)

**Purpose:**
- Cache large query results (100+ rows)
- Chunk-based loading for virtualization
- LRU eviction (max 50 results)

**Problem:** ⚠️ **Only works in browsers** - Wails desktop app CAN'T use IndexedDB!

---

### 3. **Local SQLite** (`~/.howlerops/local.db`)
**What SHOULD be stored:**
- ✅ Saved connections (passwords encrypted)
- ✅ Saved queries
- ✅ Query history (sanitized)
- ✅ User preferences/settings
- ❌ **Missing:** Query result cache

**Current State:**
- Desktop app HAS this
- Web app: DOESN'T have access (needs HTTP API)

---

### 4. **Turso** (Cloud Database)
**What it stores:**
- User accounts & authentication
- Connection metadata (sanitized, no passwords)
- Saved queries (cloud backup)
- Sanitized query history
- Sync state

**Purpose:**
- Multi-device sync
- Cloud backup
- Collaboration (future)

---

## 🚨 Architecture Issues Identified

### Issue #1: Query Result Storage Uses Wrong Backend

**Current Code:** `query-result-storage.ts`
```typescript
import { get, set, del, createStore } from 'idb-keyval'  // ❌ Browser-only API
const queryResultStore = createStore('sql-studio-db', 'query-results')
```

**Problem:**
- IndexedDB doesn't exist in Wails desktop apps
- Desktop app should use local SQLite instead

**Impact:**
- Query result caching BROKEN in desktop mode
- Large result sets cause memory issues
- Pagination won't work properly

---

### Issue #2: No Environment-Aware Storage Abstraction

**What we need:**
```typescript
// Desktop Mode (Wails)
QueryResults → Local SQLite (~/.howlerops/local.db)

// Web Mode (Browser)
QueryResults → IndexedDB

// Both use same interface, different backends
```

**Currently:**
- Hard-coded to IndexedDB everywhere
- No environment detection
- No fallback strategy

---

### Issue #3: Zustand Persistence Uses localStorage

**Current:**
- `auth-store.ts` persists to localStorage
- localStorage has ~5-10MB limit
- Not suitable for large data

**Should be:**
- Desktop: Use local SQLite (via Wails API)
- Web: Use IndexedDB (much larger capacity)

---

## ✅ Recommended Architecture

### **Layered Storage Strategy**

```
┌─────────────────────────────────────────────────────────┐
│                     ZUSTAND STORES                       │
│              (In-Memory, Runtime State)                  │
└─────────────────┬──────────────────────────────────────┘
                  │
        ┌─────────┴──────────┐
        │                    │
        ▼                    ▼
┌──────────────┐    ┌──────────────────┐
│   DESKTOP    │    │       WEB        │
│   (Wails)    │    │    (Browser)     │
└──────┬───────┘    └────────┬─────────┘
       │                     │
       ▼                     ▼
┌──────────────┐    ┌──────────────────┐
│ Local SQLite │    │    IndexedDB     │
│ ~/.howlerops/│    │  (Browser API)   │
│  local.db    │    │                  │
└──────┬───────┘    └────────┬─────────┘
       │                     │
       └──────────┬──────────┘
                  │
                  ▼
         ┌────────────────┐
         │  Turso Cloud   │
         │ (Sync/Backup)  │
         └────────────────┘
```

---

## 📋 Data Storage Mapping

| Data Type | Desktop (Wails) | Web (Browser) | Turso Cloud | Notes |
|-----------|----------------|---------------|-------------|-------|
| **Auth tokens** | localStorage | localStorage | ❌ Never | Stays client-side |
| **User preferences** | SQLite local.db | IndexedDB | ✅ Sync | Multi-device |
| **Saved connections** | SQLite local.db | ❌ Not stored | Metadata only | Passwords local-only |
| **Connection passwords** | SQLite (encrypted) | ❌ Not stored | ❌ Never | Security |
| **Saved queries** | SQLite local.db | IndexedDB | ✅ Sync | Multi-device |
| **Query history** | SQLite local.db | IndexedDB | ✅ Sanitized only | Privacy |
| **Query result cache** | **Should be SQLite** | IndexedDB | ❌ Never | Too large |
| **AI embeddings/RAG** | SQLite vectors.db | ❌ Server-side | ❌ Never | Desktop-only |
| **Runtime UI state** | Zustand (memory) | Zustand (memory) | ❌ Never | Ephemeral |

---

## 🔧 Implementation Plan

### Phase 1: Create Storage Abstraction Layer

**File:** `frontend/src/lib/storage-adapter.ts`

```typescript
import { isWailsEnvironment } from './wails-runtime'

// Universal storage interface
export interface StorageAdapter {
  get<T>(key: string): Promise<T | null>
  set<T>(key: string, value: T): Promise<void>
  delete(key: string): Promise<void>
  clear(): Promise<void>
}

// SQLite implementation (for Wails desktop)
class SqliteStorageAdapter implements StorageAdapter {
  async get<T>(key: string): Promise<T | null> {
    const { GetStorageItem } = await import('../wailsjs/go/main/App')
    const json = await GetStorageItem(key)
    return json ? JSON.parse(json) : null
  }

  async set<T>(key: string, value: T): Promise<void> {
    const { SetStorageItem } = await import('../wailsjs/go/main/App')
    await SetStorageItem(key, JSON.stringify(value))
  }

  async delete(key: string): Promise<void> {
    const { DeleteStorageItem } = await import('../wailsjs/go/main/App')
    await DeleteStorageItem(key)
  }

  async clear(): Promise<void> {
    const { ClearStorage } = await import('../wailsjs/go/main/App')
    await ClearStorage()
  }
}

// IndexedDB implementation (for web browsers)
class IndexedDBStorageAdapter implements StorageAdapter {
  async get<T>(key: string): Promise<T | null> {
    const { get } = await import('idb-keyval')
    return await get<T>(key)
  }

  async set<T>(key: string, value: T): Promise<void> {
    const { set } = await import('idb-keyval')
    await set(key, value)
  }

  async delete(key: string): Promise<void> {
    const { del } = await import('idb-keyval')
    await del(key)
  }

  async clear(): Promise<void> {
    const { clear } = await import('idb-keyval')
    await clear()
  }
}

// Auto-select based on environment
export const storage: StorageAdapter = isWailsEnvironment()
  ? new SqliteStorageAdapter()
  : new IndexedDBStorageAdapter()
```

---

### Phase 2: Update Query Result Storage

**File:** `frontend/src/lib/query-result-storage.ts`

```typescript
// Replace direct idb-keyval imports with storage adapter
import { storage } from './storage-adapter'

// Now works in both environments!
export async function storeQueryResult(result: StoredQueryResult) {
  await storage.set(result.id, result)
}

export async function getQueryResult(id: string) {
  return await storage.get<StoredQueryResult>(id)
}
```

---

### Phase 3: Add Wails Backend Methods

**File:** `app.go`

```go
// Add storage methods to App struct

func (a *App) GetStorageItem(key string) (string, error) {
    // Query local.db for item
    var value string
    err := a.storageManager.Get(key, &value)
    return value, err
}

func (a *App) SetStorageItem(key string, value string) error {
    // Store in local.db
    return a.storageManager.Set(key, value)
}

func (a *App) DeleteStorageItem(key string) error {
    return a.storageManager.Delete(key)
}

func (a *App) ClearStorage() error {
    return a.storageManager.Clear()
}
```

---

### Phase 4: Update Zustand Persistence

**Current (auth-store.ts):**
```typescript
export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({ /* ... */ }),
    {
      name: 'auth-storage',  // Uses localStorage
      partialize: (state) => ({ /* ... */ })
    }
  )
)
```

**Should be:**
```typescript
import { storage } from '@/lib/storage-adapter'

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({ /* ... */ }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => ({
        getItem: async (name) => {
          const item = await storage.get<string>(name)
          return item
        },
        setItem: async (name, value) => {
          await storage.set(name, value)
        },
        removeItem: async (name) => {
          await storage.delete(name)
        }
      }))
    }
  )
)
```

---

## 🎯 Decision Matrix: When to Use What

### Use **Zustand** (memory) for:
✅ Current UI state (tabs, panels, modals)
✅ Form inputs being actively edited
✅ Temporary selections
✅ Loading/error states
✅ Real-time data that updates frequently

### Use **Local Storage** (SQLite/IndexedDB) for:
✅ User preferences & settings
✅ Saved queries
✅ Connection configurations
✅ Query result cache (large data)
✅ Offline-capable data

### Use **Turso Cloud** for:
✅ User account data
✅ Multi-device sync
✅ Collaboration features
✅ Cloud backup
✅ Sanitized analytics

### **NEVER** store in cloud:
❌ Database passwords
❌ Full query result sets
❌ Personal/sensitive data
❌ Large binary data
❌ Temporary UI state

---

## 🚀 Migration Path

### Step 1: Add storage adapter (1-2 hours)
- Create `storage-adapter.ts`
- Add Wails backend methods
- Test both modes

### Step 2: Migrate query result storage (2-3 hours)
- Update `query-result-storage.ts`
- Test with large datasets
- Verify desktop & web modes

### Step 3: Update Zustand persist (1 hour)
- Migrate auth-store
- Migrate other persisted stores
- Test storage limits

### Step 4: Remove idb-keyval dependency (15 min)
- Only import conditionally for web mode
- Clean up unused code

---

## 📊 Current vs. Recommended

| Aspect | Current | Recommended |
|--------|---------|-------------|
| **Query results** | IndexedDB everywhere ❌ | SQLite (desktop) / IndexedDB (web) ✅ |
| **Auth tokens** | localStorage ⚠️ | localStorage (acceptable) ✅ |
| **Saved queries** | Unclear | SQLite/IndexedDB + Turso sync ✅ |
| **Environment detection** | Missing ❌ | isWailsEnvironment() ✅ |
| **Storage abstraction** | None ❌ | StorageAdapter interface ✅ |
| **Backend API** | Missing storage methods ❌ | Wails storage API ✅ |

---

## 💡 Key Insights

1. **IndexedDB doesn't work in Wails** - Need SQLite fallback
2. **LocalStorage is too small** - Use SQLite/IndexedDB for large data
3. **No storage abstraction** - Hard to maintain, test, migrate
4. **idb-keyval is fine** - But only for web mode
5. **Turso is perfect for sync** - But not for local caching

---

## ✅ Immediate Action Items

1. **Keep idb-keyval** - It's the right tool for web mode
2. **Create storage adapter** - Abstract away environment differences
3. **Add Wails storage API** - Access local SQLite from frontend
4. **Migrate query-result-storage** - Use adapter instead of direct idb-keyval
5. **Update Zustand persist** - Use adapter for better capacity
6. **Document the architecture** - Make it clear for future devs

---

## 🎓 Summary

**The Problem:**
Your app runs in two modes (desktop & web) but uses browser-only storage (IndexedDB) everywhere.

**The Solution:**
Create a storage adapter that uses:
- **Desktop (Wails)**: Local SQLite via Wails API
- **Web (Browser)**: IndexedDB via idb-keyval

**The Benefit:**
- ✅ Works in both environments
- ✅ Larger storage capacity
- ✅ Better performance
- ✅ Easier to test
- ✅ Future-proof architecture

**Bottom Line:**
Keep idb-keyval, but wrap it in an environment-aware storage adapter. Desktop mode uses SQLite, web mode uses IndexedDB. Best of both worlds!
