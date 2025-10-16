<!-- 2af03667-9170-42db-9531-dc4357d09b16 a796473c-e764-4d30-bb3c-d6ebaf2cdc37 -->
# Generic AI Chat Sidebar Implementation

## Overview

Create a generic chat sidebar that operates alongside the SQL AI Assistant, sharing the same memory store and AI provider configuration but focusing on general-purpose conversations with optional database schema context.

## Key Design Decisions

- Single button with dropdown/menu to choose between SQL Assistant and Generic Chat
- Shared memory store (sessions work for both modes)
- Optional schema context selection in generic chat
- Only one sidebar open at a time
- Reuse existing AI provider infrastructure

## Implementation Steps

### 1. Create Generic Chat Component

**File**: `frontend/src/components/generic-chat-sidebar.tsx`

Create new component with:

- Chat interface (message history display, input field)
- Memory session management (reuse from query-editor)
- Optional schema context selector (checkbox-based UI)
- Send/receive messages to AI provider
- No SQL-specific features (no "Generate SQL" or "Fix SQL")

Key sections:

```tsx
- Message history display (user/assistant messages)
- Input textarea for prompts
- Optional collapsible "Add Context" section:
  - Schema/table selector (checkboxes)
  - Custom context text input
- Memory session switcher (same as SQL Assistant)
- Provider/model info display
```

### 2. Update AI Store

**File**: `frontend/src/store/ai-store.ts`

Add new action:

- `sendGenericMessage(prompt: string, context?: string)` - Generic chat without SQL generation
- Returns assistant response text
- Similar to generateSQL but without SQL-specific prompt engineering

### 3. Extend Memory Store (Optional)

**File**: `frontend/src/store/ai-memory-store.ts`

Consider adding:

- Optional `metadata.chatType: 'sql' | 'generic'` to distinguish session types
- This helps filter/organize sessions by type in the UI

### 4. Modify Query Editor

**File**: `frontend/src/components/query-editor.tsx`

Changes:

- Add state: `const [aiSidebarMode, setAISidebarMode] = useState<'sql' | 'generic'>('sql')`
- Replace single AI button with dropdown menu button (using DropdownMenu component)
- Menu options: "SQL Assistant" and "Generic Chat"
- Show appropriate sidebar based on mode
- Both sidebars use `showAIDialog` state (only one open at a time)

Update AI button section (~line 724):

```tsx
<DropdownMenu>
  <DropdownMenuTrigger asChild>
    <Button variant={showAIDialog ? "default" : "ghost"} size="sm">
      <Sparkles className="h-4 w-4" />
      <ChevronDown className="h-3 w-3 ml-1" />
    </Button>
  </DropdownMenuTrigger>
  <DropdownMenuContent>
    <DropdownMenuItem onClick={() => { setAISidebarMode('sql'); setShowAIDialog(true) }}>
      SQL Assistant
    </DropdownMenuItem>
    <DropdownMenuItem onClick={() => { setAISidebarMode('generic'); setShowAIDialog(true) }}>
      Generic Chat
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
```

Add conditional rendering:

```tsx
{aiEnabled && showAIDialog && aiSidebarMode === 'sql' && (
  // Existing SQL Assistant Sheet
)}
{aiEnabled && showAIDialog && aiSidebarMode === 'generic' && (
  <GenericChatSidebar
    open={showAIDialog}
    onClose={() => setShowAIDialog(false)}
    connections={getFilteredConnections()}
    schemasMap={multiDBSchemas}
  />
)}
```

### 5. Create Schema Context Selector

**File**: `frontend/src/components/schema-context-selector.tsx`

Component for selecting optional schema context:

- Display available connections/schemas/tables
- Checkbox tree structure (similar to AISchemaDisplay but with selection)
- Return selected items as formatted context string
- Props: `connections`, `schemasMap`, `onContextChange`

### 6. Backend Integration (if needed)

**File**: `frontend/src/lib/wails-ai-api.ts`

May need to add:

- Generic chat API function that doesn't force SQL generation
- Reuse existing provider infrastructure but with different system prompts

### 7. Update Types

**Files**: Various

- Import `DropdownMenu` components from shadcn/ui
- Add `MessageCircle` icon from lucide-react for generic chat
- Ensure proper types for chat messages vs SQL suggestions

## Technical Considerations

### Memory Session Sharing

- Both SQL and Generic chat use same `useAIMemoryStore`
- Sessions can be resumed in either mode
- Consider adding visual indicator (badge/icon) for session type

### Context Building

- Generic chat: optional schema context + conversation history
- SQL Assistant: required schema context + conversation history + SQL-specific instructions
- Both use same memory context builder

### Provider Configuration

- Reuse existing AI config (provider, API keys, models)
- Same connection status checks
- Same sync to backend if enabled

### UI/UX

- Clear visual distinction between modes (titles, descriptions)
- "SQL Assistant" - focused on SQL generation/fixing
- "Generic Chat" - general questions, explanations, documentation help
- Smooth transition when switching modes (close one, open other)

## Files to Create

1. `frontend/src/components/generic-chat-sidebar.tsx` - Main chat component
2. `frontend/src/components/schema-context-selector.tsx` - Optional context selector

## Files to Modify

1. `frontend/src/components/query-editor.tsx` - Add mode toggle and conditional rendering
2. `frontend/src/store/ai-store.ts` - Add generic chat action
3. `frontend/src/store/ai-memory-store.ts` - Optional metadata for session types
4. `frontend/src/lib/wails-ai-api.ts` - Generic chat API call (if needed)

### To-dos

- [ ] Create GenericChatSidebar component with message history and input
- [ ] Create SchemaContextSelector component for optional database context
- [ ] Add sendGenericMessage action to AI store
- [ ] Add optional chatType metadata to memory sessions
- [ ] Update query-editor with dropdown menu and mode toggle
- [ ] Test both sidebars, memory sharing, and context selection