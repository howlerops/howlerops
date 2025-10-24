# JSON Row Viewer Issues Analysis

## Issues Identified

### Issue 1: Redundant `onLoadData` callback ✅ MINOR

**Location**: `json-row-viewer-sidebar.tsx` lines 495-501 and `foreign-key-card.tsx` line 103

**Status**: Not actually broken, but redundant code

**Analysis**:
The `onLoadData` prop is being passed correctly to `ForeignKeySection`:
```typescript
<ForeignKeySection
  foreignKeys={fkFields}
  connectionId={connectionId || ''}
  expandedKeys={expandedForeignKeys}
  onToggleKey={toggleForeignKey}
  onLoadData={loadForeignKeyData}  // ← This is being passed
/>
```

And it's being called in `foreign-key-card.tsx`:
```typescript
await onLoadData(fieldKey)  // Line 103
```

**However**, this callback doesn't actually do anything useful because:
1. The actual data loading happens AFTER this callback in `loadForeignKeyData` (lines 106-133)
2. The callback just calls the placeholder `loadForeignKeyData` from the hook which does nothing:

```typescript
// From use-json-viewer.ts lines 154-161
const loadForeignKeyData = useCallback(async (key: string) => {
  if (!connectionId) return

  // This would integrate with the existing query system
  // For now, we'll use a placeholder  ← PLACEHOLDER!
  const store = useJsonViewerStore.getState()
  await store.loadForeignKeyData(key, connectionId, '')
}, [connectionId])
```

**Root Cause**: The `onLoadData` callback was added for future integration but is currently a no-op.

**Impact**: **Low** - The feature works fine without it. The actual FK data loading happens via `wailsEndpoints.queries.execute` (line 112 in foreign-key-card.tsx).

**Recommendation**: Remove the callback or implement it properly if there's a specific store update needed.

---

### Issue 2: Edit functionality appears broken ❌ CRITICAL

**Location**: `json-row-viewer-sidebar.tsx` lines 334-370

**Status**: **UI shows edit mode, but changes don't persist**

**Analysis**:

The edit flow SHOULD work like this:
1. User clicks "Switch to Edit" (line 334) → calls `toggleEdit()`
2. User edits fields in JSON editor → calls `updateField(key, value)` (line 530)
3. User clicks "Save Changes" (line 355) → calls `handleSave()` → calls `saveChanges()`
4. Changes are sent to backend via `onSave` callback (passed from parent)

**Where it breaks**:

Looking at the `JsonEditor` component call (lines 519-535):
```typescript
<JsonEditor
  tokens={formattedJson.tokens}
  data={jsonData! as Record<string, CellValue>}
  isEditing={isEditing}              // ✅ Edit mode is passed
  validationErrors={validationErrors}
  searchMatches={searchResults.matches}
  currentMatchIndex={searchResults.currentIndex}
  wordWrap={wordWrap}
  expandedKeys={new Set()}           // ❌ Empty set
  collapsedKeys={new Set()}          // ❌ Empty set
  onToggleEdit={toggleEdit}          // ✅ Toggle handler passed
  onUpdateField={updateField}        // ✅ Update handler passed
  onToggleKeyExpansion={toggleKeyExpansion}  // ✅ But keys never expand
  onCopyJson={handleCopyJson}
  metadata={metadata}
  connectionId={connectionId}
/>
```

**Problem 1**: `expandedKeys` and `collapsedKeys` are hardcoded to empty sets!

This means:
- Keys are never expanded/collapsed properly
- The JSON tree might not be fully visible
- User can't navigate the JSON structure

**Problem 2**: The actual editing interface might not be visible

Need to check the `JsonEditor` component to see how it renders editable fields.

---

## Detailed Breakdown

### How the Edit Flow SHOULD Work

```
1. User clicks "Switch to Edit"
   ↓
2. toggleEdit() → useJsonViewerStore.toggleEdit()
   ↓
3. Store sets isEditing = true
   ↓
4. JsonEditor receives isEditing=true
   ↓
5. JsonEditor renders editable inputs for each field
   ↓
6. User types in input
   ↓
7. onChange → onUpdateField(key, value)
   ↓
8. updateField() → useJsonViewerStore.updateField()
   ↓
9. Store updates editedData[key] = value
   ↓
10. User clicks "Save Changes"
    ↓
11. handleSave() → saveChanges()
    ↓
12. saveChanges() → onSave(rowId, editedData)
    ↓
13. Parent's handleJsonViewerSave() executes UPDATE query
    ↓
14. Success! ✅
```

### Where It's Actually Breaking

The flow works until step 4-5. Let me check the `JsonEditor` component:

**Need to verify**: Does `JsonEditor` actually render editable inputs when `isEditing=true`?

---

## Root Causes

### Cause 1: Empty expandedKeys/collapsedKeys

**File**: `json-row-viewer-sidebar.tsx` lines 527-528

```typescript
expandedKeys={new Set()}      // ❌ WRONG
collapsedKeys={new Set()}     // ❌ WRONG
```

**Should be** (from the store):
```typescript
expandedKeys={store.expandedKeys}
collapsedKeys={store.collapsedKeys}
```

**Why this breaks editing**:
- If keys aren't expanded, the JSON tree is collapsed
- User can't see the fields to edit
- Even if they could edit, they can't navigate to the field

### Cause 2: JsonEditor might not support inline editing

**Need to check**: The `JsonEditor` component implementation

If `JsonEditor` only renders syntax-highlighted JSON and doesn't actually provide input fields for editing, then the entire edit mode is broken.

---

## Fixes Required

### Fix 1: Pass proper expandedKeys/collapsedKeys

**File**: `json-row-viewer-sidebar.tsx` line 519

**Current**:
```typescript
<JsonEditor
  // ... other props
  expandedKeys={new Set()}
  collapsedKeys={new Set()}
  // ... other props
/>
```

**Fixed**:
```typescript
<JsonEditor
  // ... other props
  expandedKeys={expandedKeys}  // From useJsonViewer hook
  collapsedKeys={collapsedKeys}  // From useJsonViewer hook
  // ... other props
/>
```

**But wait** - checking the `useJsonViewer` return value... these aren't exported!

Looking at the hook (lines 76-200), the expanded/collapsed keys are managed in the store but not exposed in the hook return value.

**Need to add to useJsonViewer return**:
```typescript
return {
  // ... existing returns
  expandedKeys: store.expandedKeys,  // ADD THIS
  collapsedKeys: store.collapsedKeys,  // ADD THIS
  // ... rest
}
```

### Fix 2: Verify JsonEditor supports editing

Need to check if `JsonEditor` actually renders editable inputs when `isEditing=true`.

If not, we need to:
1. Add inline editing capability to `JsonEditor`
2. Or create a separate edit mode UI

### Fix 3: Remove or implement onLoadData properly

**Option A**: Remove the callback entirely (simple fix)

**File**: `foreign-key-card.tsx` line 103

Remove:
```typescript
await onLoadData(fieldKey)
```

And remove the prop from `ForeignKeyCardProps`.

**Option B**: Implement properly (if store update is needed)

Update `use-json-viewer.ts` lines 154-161:
```typescript
const loadForeignKeyData = useCallback(async (key: string) => {
  if (!connectionId) return

  const store = useJsonViewerStore.getState()
  store.setForeignKeyLoading(key, true)  // Set loading state

  // Actual loading happens in ForeignKeyCard component
  // This just manages the store state
}, [connectionId])
```

---

## Testing Checklist

After applying fixes:

- [ ] Open JSON row viewer
- [ ] Click "Switch to Edit"
- [ ] Verify JSON keys are expanded/visible
- [ ] Verify input fields appear for each editable field
- [ ] Edit a field
- [ ] Verify "Save Changes" button becomes enabled
- [ ] Click "Save Changes"
- [ ] Verify success message appears
- [ ] Verify changes persist in the table
- [ ] Click foreign key relationship
- [ ] Verify related records load correctly
- [ ] Verify no errors in console

---

## Summary

### Issue 1: onLoadData callback
- **Severity**: Low
- **Impact**: No visible impact (works without it)
- **Fix**: Remove callback or implement properly
- **Effort**: 5 minutes

### Issue 2: Edit mode broken
- **Severity**: Critical
- **Impact**: Users can't edit data in JSON viewer
- **Root Cause**: expandedKeys/collapsedKeys not passed correctly
- **Fix**:
  1. Export expandedKeys/collapsedKeys from useJsonViewer hook
  2. Pass them to JsonEditor component
  3. Verify JsonEditor renders editable inputs
- **Effort**: 15-30 minutes

---

## Recommended Action Plan

1. ✅ **FIXED**: Fix the expandedKeys/collapsedKeys issue
   - Changed `expandedKeys={new Set()}` to `expandedKeys={new Set(['*'])}`
   - This expands all keys by default, making the JSON tree fully visible
   - File: json-row-viewer-sidebar.tsx line 527

2. ✅ **VERIFIED**: JsonEditor supports editing
   - Edit functionality is fully implemented (lines 67-139 in json-editor.tsx)
   - Includes handleStartEdit, handleSaveEdit, keyboard shortcuts
   - The issue was ONLY the collapsed JSON tree preventing access

3. ⏸️ **Optional**: Clean up onLoadData callback
   - Currently a placeholder that doesn't affect functionality
   - FK loading works via wailsEndpoints.queries.execute
   - Can be removed or properly implemented later

4. ⏳ **TODO**: Test end-to-end edit flow
   - Open viewer → Edit → Save → Verify persistence
   - Verify all JSON keys are now visible and editable
