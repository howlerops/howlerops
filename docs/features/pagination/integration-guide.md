# Pagination Integration Guide - Quick Start

## TL;DR

Add pagination to AI query results in 3 steps:

### Step 1: Import Components

```typescript
import { QueryPagination } from '@/components/query-pagination'
```

### Step 2: Add State Management

```typescript
const [paginationState, setPaginationState] = useState({
  page: 1,
  pageSize: 200,
})
```

### Step 3: Add Pagination UI

```tsx
{attachment.result?.totalPages > 1 && (
  <QueryPagination
    currentPage={attachment.result.page}
    totalPages={attachment.result.totalPages}
    pageSize={attachment.result.pageSize}
    totalRows={attachment.result.totalRows}
    onPageChange={(page) => handlePageChange(page)}
    onPageSizeChange={(size) => handlePageSizeChange(size)}
    loading={session?.status === 'streaming'}
  />
)}
```

## Full Integration Example

### In `ai-query-tab.tsx` or similar component:

```typescript
import { useState, useCallback } from 'react'
import { QueryPagination } from '@/components/query-pagination'
import { useAIQueryAgentStore } from '@/store/ai-query-agent-store'

export function AIQueryTab() {
  const { sendMessage, activeSession } = useAIQueryAgentStore()
  const [paginationState, setPaginationState] = useState({
    page: 1,
    pageSize: 200,
  })

  // Store the last query to re-run with different pagination
  const [lastQuery, setLastQuery] = useState<{
    message: string
    connectionId: string
    provider: string
    model: string
  } | null>(null)

  const handleSendQuery = async (message: string, connectionId: string) => {
    const queryParams = {
      sessionId: activeSession?.id || '',
      message,
      provider: 'openai',
      model: 'gpt-4',
      connectionId,
      page: paginationState.page,
      pageSize: paginationState.pageSize,
    }

    // Store for pagination handlers
    setLastQuery({
      message,
      connectionId,
      provider: queryParams.provider,
      model: queryParams.model,
    })

    await sendMessage(queryParams)
  }

  const handlePageChange = useCallback(async (page: number) => {
    if (!lastQuery || !activeSession) return

    setPaginationState(prev => ({ ...prev, page }))

    await sendMessage({
      sessionId: activeSession.id,
      message: lastQuery.message,
      provider: lastQuery.provider,
      model: lastQuery.model,
      connectionId: lastQuery.connectionId,
      page,
      pageSize: paginationState.pageSize,
    })
  }, [lastQuery, activeSession, paginationState.pageSize, sendMessage])

  const handlePageSizeChange = useCallback(async (pageSize: number) => {
    if (!lastQuery || !activeSession) return

    setPaginationState({ page: 1, pageSize })

    await sendMessage({
      sessionId: activeSession.id,
      message: lastQuery.message,
      provider: lastQuery.provider,
      model: lastQuery.model,
      connectionId: lastQuery.connectionId,
      page: 1,
      pageSize,
    })
  }, [lastQuery, activeSession, sendMessage])

  return (
    <div>
      {/* Your query input and results */}

      {/* Render results */}
      {activeSession?.messages.map(message => (
        <div key={message.id}>
          {message.attachments?.map(attachment => (
            <div key={`${message.id}-${attachment.type}`}>
              {attachment.type === 'result' && attachment.result && (
                <>
                  {/* Results table */}
                  <QueryResultsTable data={attachment.result} />

                  {/* Pagination - only show if multiple pages */}
                  {attachment.result.totalPages > 1 && (
                    <QueryPagination
                      currentPage={attachment.result.page}
                      totalPages={attachment.result.totalPages}
                      pageSize={attachment.result.pageSize}
                      totalRows={attachment.result.totalRows}
                      onPageChange={handlePageChange}
                      onPageSizeChange={handlePageSizeChange}
                      loading={activeSession?.status === 'streaming'}
                    />
                  )}
                </>
              )}
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}
```

## Props Reference

### QueryPagination Props

| Prop | Type | Description |
|------|------|-------------|
| `currentPage` | `number` | Current page number (1-indexed) |
| `totalPages` | `number` | Total number of pages |
| `pageSize` | `number` | Rows per page |
| `totalRows` | `number` | Total rows in result set |
| `onPageChange` | `(page: number) => void` | Called when page changes |
| `onPageSizeChange` | `(size: number) => void` | Called when page size changes |
| `loading?` | `boolean` | Disables controls during query |

## Backend Data Structure

### Request (from Frontend to Backend)

```typescript
interface AIQueryAgentRequest {
  sessionId: string
  message: string
  provider: string
  model: string
  connectionId?: string
  // ... other fields ...
  page?: number      // Current page (1-indexed)
  pageSize?: number  // Rows per page
}
```

### Response (from Backend to Frontend)

```typescript
interface AgentResultAttachment {
  columns: string[]
  rows: Record<string, unknown>[]
  rowCount: number
  executionTimeMs: number
  limited: boolean
  connectionId?: string
  // Pagination metadata
  totalRows?: number    // Total rows available (unpaged)
  page?: number         // Current page number
  pageSize?: number     // Page size used
  totalPages?: number   // Total pages available
  hasMore?: boolean     // More pages available
}
```

## Common Patterns

### Pattern 1: Store Last Query for Pagination

```typescript
const [lastQueryParams, setLastQueryParams] = useState<{
  message: string
  connectionId: string
  // ... other params
} | null>(null)

const handleQuery = async (params) => {
  setLastQueryParams(params)
  await sendMessage({ ...params, page: 1, pageSize: 200 })
}

const handlePageChange = async (page: number) => {
  if (!lastQueryParams) return
  await sendMessage({ ...lastQueryParams, page, pageSize: currentPageSize })
}
```

### Pattern 2: Sync Pagination State with Results

```typescript
useEffect(() => {
  if (latestResult?.page) {
    setPaginationState({
      page: latestResult.page,
      pageSize: latestResult.pageSize,
    })
  }
}, [latestResult])
```

### Pattern 3: Disable Actions During Loading

```typescript
const isLoading = activeSession?.status === 'streaming'

<QueryPagination
  // ... other props
  loading={isLoading}
/>
```

### Pattern 4: Show/Hide Pagination

```typescript
const shouldShowPagination =
  attachment.result?.totalPages > 1 &&
  !attachment.result?.limited // Don't show if manually limited

{shouldShowPagination && (
  <QueryPagination {...paginationProps} />
)}
```

## Styling

The pagination component uses Tailwind CSS and shadcn/ui theming. To customize:

### Custom Colors

```typescript
<QueryPagination
  className="bg-gray-50 dark:bg-gray-900"
  {...props}
/>
```

### Custom Size

```typescript
// In pagination.tsx
<PaginationLink
  size="sm"  // or "default" or "lg"
  {...props}
/>
```

## Troubleshooting

### Pagination doesn't appear

✅ Check: `attachment.result?.totalPages > 1`
✅ Check: Backend is returning pagination metadata
✅ Check: Query has more rows than pageSize

### Page change doesn't work

✅ Check: `lastQuery` is stored correctly
✅ Check: `onPageChange` is called with correct params
✅ Check: Backend receives `page` and `pageSize` params

### Loading state doesn't work

✅ Check: `loading` prop is set from `session?.status === 'streaming'`
✅ Check: Session status updates correctly

### Pagination resets unexpectedly

✅ Check: Not resetting `page` to 1 on every query
✅ Check: Storing pagination state separately from query params
✅ Check: Using `useCallback` to prevent unnecessary re-renders

## Performance Tips

### 1. Debounce Page Changes

```typescript
const debouncedPageChange = useMemo(
  () => debounce(handlePageChange, 300),
  [handlePageChange]
)
```

### 2. Prefetch Next Page

```typescript
useEffect(() => {
  if (currentPage < totalPages) {
    // Prefetch next page in background
    prefetchPage(currentPage + 1)
  }
}, [currentPage, totalPages])
```

### 3. Cache Page Data

```typescript
const pageCache = useRef<Map<number, QueryResult>>(new Map())

const loadPage = async (page: number) => {
  if (pageCache.current.has(page)) {
    return pageCache.current.get(page)
  }

  const result = await fetchPage(page)
  pageCache.current.set(page, result)
  return result
}
```

## Testing

### Unit Test Example

```typescript
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryPagination } from '@/components/query-pagination'

describe('QueryPagination', () => {
  it('calls onPageChange when Next is clicked', () => {
    const onPageChange = jest.fn()

    render(
      <QueryPagination
        currentPage={1}
        totalPages={5}
        pageSize={100}
        totalRows={500}
        onPageChange={onPageChange}
        onPageSizeChange={jest.fn()}
      />
    )

    fireEvent.click(screen.getByText('Next'))
    expect(onPageChange).toHaveBeenCalledWith(2)
  })
})
```

## FAQ

**Q: Can I use different page sizes than the defaults?**
A: Yes, edit `PAGE_SIZE_OPTIONS` in `query-pagination.tsx`

**Q: Can I start on a different page?**
A: Yes, pass initial `page` in the query params

**Q: Does it work with all databases?**
A: Yes, it uses database-native pagination (LIMIT/OFFSET, skip/limit, etc.)

**Q: What about very large offsets (millions of rows)?**
A: Consider cursor-based pagination for such cases (future enhancement)

**Q: Can I export all pages?**
A: Not yet - this is a future enhancement

## Complete Working Example

See `frontend/src/components/ai-query-tab.tsx` for a complete integration example (after integration is complete).

## Support

For issues or questions:
1. Check `PAGINATION_IMPLEMENTATION.md` for detailed implementation
2. Check `PAGINATION_SUMMARY.md` for overview
3. Review backend code in `ai_query_agent.go`
4. Review frontend code in `frontend/src/components/query-pagination.tsx`
