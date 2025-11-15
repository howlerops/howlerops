# Pagination Implementation for Auto-Limited Query Results

## Overview

This document describes the pagination implementation added to support navigating through auto-limited query results (default 200 rows).

## Changes Made

### Backend (Go)

#### 1. Data Model Updates

**File: `ai_query_agent.go`**

- Updated `AIQueryAgentRequest` struct to include pagination parameters:
  ```go
  Page          int  `json:"page,omitempty"`     // Current page number (1-indexed)
  PageSize      int  `json:"pageSize,omitempty"` // Rows per page
  ```

- Updated `AIQueryAgentResultAttachment` struct with pagination metadata:
  ```go
  TotalRows  int64 `json:"totalRows,omitempty"`  // Total rows available
  Page       int   `json:"page,omitempty"`       // Current page
  PageSize   int   `json:"pageSize,omitempty"`   // Page size
  TotalPages int   `json:"totalPages,omitempty"` // Total pages
  HasMore    bool  `json:"hasMore,omitempty"`    // More pages available
  ```

- Updated `ReadOnlyQueryResult` struct with same pagination fields

- Added new function `ExecuteReadOnlyQueryWithPagination()` that:
  - Calculates offset from page and pageSize
  - Uses `QueryOptions.Offset` for database query
  - Computes pagination metadata (totalPages, hasMore, etc.)
  - Returns results with full pagination info

- Added corresponding `executeMultiReadOnlyQueryWithPagination()` for multi-database queries

- Updated `convertToResultAttachment()` to include pagination fields

**File: `backend-go/pkg/database/types.go`**

- `QueryOptions` already had `Offset` field (added comment for clarity)
- `QueryResult` already had pagination fields `TotalRows`, `PagedRows`, `HasMore`, `Offset`

**File: `backend-go/pkg/database/multiquery/types.go`**

- Added `Offset` field to `Options` struct:
  ```go
  Offset   int // Offset for pagination
  ```

#### 2. Query Execution Flow

The pagination flow in `StreamAIQueryAgent()`:

1. Check if `req.Page` and `req.PageSize` are provided
2. Calculate offset: `offset = (page - 1) * pageSize`
3. Call `ExecuteReadOnlyQueryWithPagination()` instead of `ExecuteReadOnlyQuery()`
4. Return results with pagination metadata attached

### Frontend (TypeScript/React)

#### 1. UI Components

**File: `frontend/src/components/ui/pagination.tsx`** (NEW)

- Created shadcn/ui compatible pagination component with:
  - `Pagination`, `PaginationContent`, `PaginationItem`
  - `PaginationLink`, `PaginationPrevious`, `PaginationNext`
  - `PaginationFirst`, `PaginationLast`, `PaginationEllipsis`

**File: `frontend/src/components/query-pagination.tsx`** (NEW)

- Smart pagination controls component featuring:
  - Page number display with ellipsis for large page counts
  - Previous/Next/First/Last navigation buttons
  - Page size selector (50, 100, 200, 500, 1000)
  - "Jump to page" input with validation
  - Row range display (e.g., "Showing 201-400 of 5000 rows")
  - Loading state support
  - Keyboard shortcuts (Enter to jump)

#### 2. Store Updates

**File: `frontend/src/store/ai-query-agent-store.ts`**

- Updated `SendMessageOptions` interface:
  ```typescript
  page?: number        // Current page number (1-indexed)
  pageSize?: number    // Rows per page
  ```

- Updated `sendMessage` function to pass pagination params to Wails backend

### Integration Points

To integrate pagination into the AI query results display:

1. **Extract pagination metadata** from `AgentResultAttachment`:
   ```typescript
   const { totalRows, page, pageSize, totalPages, hasMore } = attachment.result
   ```

2. **Add QueryPagination component** to your results display:
   ```tsx
   {attachment.result?.totalPages > 1 && (
     <QueryPagination
       currentPage={attachment.result.page}
       totalPages={attachment.result.totalPages}
       pageSize={attachment.result.pageSize}
       totalRows={attachment.result.totalRows}
       onPageChange={(page) => handlePageChange(page)}
       onPageSizeChange={(size) => handlePageSizeChange(size)}
       loading={isLoading}
     />
   )}
   ```

3. **Implement handlers** for page changes:
   ```typescript
   const handlePageChange = async (page: number) => {
     await sendMessage({
       ...currentOptions,
       page,
       pageSize: currentPageSize,
       message: lastQuery, // Re-run same query with new page
     })
   }

   const handlePageSizeChange = async (pageSize: number) => {
     await sendMessage({
       ...currentOptions,
       page: 1, // Reset to first page
       pageSize,
       message: lastQuery,
     })
   }
   ```

## Database Support

The pagination implementation works with:

- **PostgreSQL** - Uses `LIMIT` and `OFFSET` clauses
- **MySQL/MariaDB** - Uses `LIMIT` and `OFFSET` clauses
- **SQLite** - Uses `LIMIT` and `OFFSET` clauses
- **ClickHouse** - Uses `LIMIT` and `OFFSET` clauses
- **TiDB** - Uses `LIMIT` and `OFFSET` clauses
- **MongoDB** - Uses `.skip()` and `.limit()` methods
- **Elasticsearch/OpenSearch** - Uses `from` and `size` parameters
- **Multi-database queries** - Supports pagination across federated queries

## Features

### Pagination Controls

1. **Page Navigation**:
   - First, Previous, Next, Last buttons
   - Direct page number selection
   - Disabled states for boundary conditions

2. **Page Size Selection**:
   - Dropdown with common sizes: 50, 100, 200, 500, 1000
   - Resets to page 1 when size changes

3. **Jump to Page**:
   - Text input for direct page entry
   - Validation (must be between 1 and totalPages)
   - Enter key support

4. **Status Display**:
   - Row range: "Showing 201-400 of 5000 rows"
   - Current page indicator
   - Total pages count

### Edge Cases Handled

1. **Page > maxPages**: Validated and rejected
2. **Invalid page numbers**: Input validation prevents
3. **Page size changes**: Resets to page 1
4. **Loading states**: All controls disabled during query
5. **Single page results**: Pagination hidden when totalPages === 1
6. **Empty results**: Gracefully handled

## Breaking Changes

None - this is a backward-compatible addition. Queries without pagination params continue to work with auto-limit behavior.

## Migration Notes

- Existing queries will continue using auto-limit (default 200 rows)
- No database migrations required
- No frontend state migrations required
- Pagination UI automatically appears when results have multiple pages

## Testing

### Manual Testing Steps

1. **Basic Pagination**:
   - Run query returning >200 rows
   - Verify pagination controls appear
   - Click Next, verify page 2 loads
   - Click Previous, verify page 1 loads

2. **Page Size Changes**:
   - Change page size to 50
   - Verify results reload with 50 rows
   - Verify pagination recalculates

3. **Jump to Page**:
   - Enter page number
   - Press Enter or click Go
   - Verify correct page loads

4. **Edge Cases**:
   - Try page > totalPages (should reject)
   - Try page 0 or negative (should reject)
   - Try invalid input (should disable Go button)

5. **Loading States**:
   - Verify all controls disabled during query
   - Verify loading indicator shows
   - Verify controls re-enable after load

6. **Different Databases**:
   - Test with PostgreSQL
   - Test with MySQL
   - Test with SQLite
   - Test with multi-database queries

### Automated Testing

Add integration tests for:

```go
// Backend tests
func TestExecuteReadOnlyQueryWithPagination(t *testing.T) {
    // Test page 1
    // Test page 2
    // Test page size changes
    // Test offset calculation
    // Test totalPages calculation
}

func TestPaginationBoundaries(t *testing.T) {
    // Test first page
    // Test last page
    // Test page > maxPages
    // Test invalid page numbers
}
```

```typescript
// Frontend tests
describe('QueryPagination', () => {
  it('renders pagination controls', () => {})
  it('handles page navigation', () => {})
  it('handles page size changes', () => {})
  it('validates jump to page input', () => {})
  it('shows correct row range', () => {})
  it('disables controls when loading', () => {})
})
```

## Performance Considerations

1. **Efficient SQL**: Uses `OFFSET` and `LIMIT` for minimal data transfer
2. **No full table scans**: Database indexes should be used for optimal performance
3. **Metadata caching**: Consider caching totalRows for expensive COUNT queries
4. **Lazy loading**: Only requested page is loaded, not all data
5. **Request batching**: Multiple page changes within short time could be debounced

## Future Enhancements

1. **Count optimization**: Cache total row count for repeated queries
2. **Cursor-based pagination**: For very large datasets (millions of rows)
3. **Virtual scrolling**: Infinite scroll instead of discrete pages
4. **Prefetching**: Load next page in background for instant navigation
5. **Deep linking**: URL params for shareable paginated results
6. **Export pagination**: Allow exporting specific page ranges
7. **Server-side sorting**: Sort paginated results without refetching all data

## Implementation Checklist

- [x] Backend: Add pagination fields to request/response types
- [x] Backend: Implement ExecuteReadOnlyQueryWithPagination
- [x] Backend: Update multiquery package for pagination
- [x] Backend: Calculate pagination metadata
- [x] Frontend: Create pagination UI component
- [x] Frontend: Create pagination controls component
- [x] Frontend: Update AI query store to pass pagination params
- [ ] Frontend: Integrate pagination into AI query results display
- [ ] Testing: Add backend pagination tests
- [ ] Testing: Add frontend pagination tests
- [ ] Documentation: Update user documentation
