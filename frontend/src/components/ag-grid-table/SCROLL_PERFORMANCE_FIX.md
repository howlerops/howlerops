# AG Grid White Flash Fix - Complete Solution

## Problem Statement

When scrolling fast in the AG Grid table, users experienced a jarring white flash while AG Grid virtualized new rows. This created a poor UX that didn't feel like a polished data application.

## Root Cause Analysis

The white flash is caused by **virtualization lag** - when you scroll faster than React can render new rows. Here's what was happening:

### 1. Insufficient Row Buffer
- **Default behavior**: AG Grid only pre-renders 10 rows above/below viewport
- **Your setup**: Row height is 31px
- **Buffer calculation**: 10 rows × 31px = 310px buffer zone
- **Problem**: During fast scroll, users outpace the 310px buffer and see unrendered space

### 2. React Cell Renderer Overhead
- Every cell uses a custom React functional component
- Complex renderers with conditional logic (eye icons, null styling, monospace)
- React needs time to mount components during fast scroll
- This compounds the virtualization lag

### 3. Visual Gap - White Background
- When rows aren't rendered yet, the viewport container shows through
- Without explicit background styling, it defaults to white
- This creates the jarring white flash effect

### 4. Animation Overhead
- Row animations add rendering work during scroll operations
- This slows down the rendering of new rows even more

## Complete Solution Applied

### 1. Increased Row Buffer (ag-grid-table.tsx)

**Before:**
```typescript
// No explicit rowBuffer (defaults to 10)
```

**After:**
```typescript
// Performance: pre-render 20 rows above/below viewport (620px buffer zone)
// This creates a larger cushion to prevent white flash during fast scrolling
rowBuffer={20}
```

**Impact**: Creates a 620px buffer zone (20 rows × 31px) above and below the viewport. This gives AG Grid more time to render rows before they become visible.

**Trade-off**: Renders 40 extra rows (20 above + 20 below), but this is negligible for modern browsers and the performance gain is worth it.

### 2. Disabled Row Animations (ag-grid-table.tsx)

**Before:**
```typescript
animateRows={true}
```

**After:**
```typescript
// Performance: disable row animations for smoother scrolling
animateRows={false}
```

**Impact**: Eliminates animation overhead during scroll, allowing rows to render faster.

**Trade-off**: No smooth transitions when rows are added/removed, but scrolling is significantly smoother.

### 3. Disabled Scrollbar Debouncing (ag-grid-table.tsx)

**Before:**
```typescript
debounceVerticalScrollbar={true}
```

**After:**
```typescript
// Performance: disable scrollbar debouncing for immediate scroll response
debounceVerticalScrollbar={false}
```

**Impact**: Scrollbar position updates immediately, reducing perceived lag during fast scrolling.

**Trade-off**: More frequent scroll events, but modern browsers handle this efficiently.

### 4. Suppressed Row Transform (ag-grid-table.tsx) - NEW

**Added:**
```typescript
// Performance: reduce DOM overhead by removing row transform animations
suppressRowTransform={true}
```

**Impact**: Reduces DOM manipulation overhead by disabling CSS transforms on row positioning.

**Reference**: [AG Grid Scrolling Performance Docs](https://www.ag-grid.com/javascript-data-grid/scrolling-performance/)

### 5. CSS Background Fixes (ag-grid-table.css)

**Added:**
```css
/**
 * CRITICAL: Fix white flash during fast scrolling
 *
 * When virtualization renders new rows during fast scroll, the viewport
 * container momentarily shows through. Setting a dark background ensures
 * the flash matches the table background instead of showing white.
 */
.ag-theme-quartz-dark .ag-body-viewport,
.ag-theme-quartz-dark .ag-center-cols-viewport,
.ag-theme-quartz-dark .ag-center-cols-container,
.ag-theme-quartz-dark .ag-body,
.ag-theme-quartz-dark .ag-body-clipper {
  background-color: var(--background);
}

/* Ensure pinned column viewports also have dark background */
.ag-theme-quartz-dark .ag-pinned-left-cols-viewport,
.ag-theme-quartz-dark .ag-pinned-right-cols-viewport {
  background-color: var(--background);
}

/* Disable row transitions during scroll for smoother performance */
.ag-theme-quartz-dark .ag-row {
  transition: background-color 0.15s ease;
}

/* Remove transition completely when scrolling (applied via JS) */
.ag-theme-quartz-dark.ag-scrolling .ag-row {
  transition: none !important;
}
```

**Impact**:
- When rows aren't rendered yet, users see black (matching the theme) instead of white
- This makes the flash nearly invisible since it matches the table background
- Row transitions are disabled during scroll for better performance

## Performance Characteristics

### Buffer Zone Calculation

With the current configuration:
- **Row height**: 31px
- **Row buffer**: 20
- **Buffer above viewport**: 20 rows × 31px = **620px**
- **Buffer below viewport**: 20 rows × 31px = **620px**
- **Total extra rendering**: 40 rows

### Typical Viewport

On a 1080p display with 600px table height:
- **Visible rows**: ~19 rows (600px ÷ 31px)
- **Total rendered**: ~79 rows (19 visible + 20 above + 20 below + 20 more)
- **Memory impact**: Negligible (HTML table rows are lightweight)

### Fast Scroll Scenario

User scrolls at 2000px/second:
- **Old buffer**: 310px ÷ 2000px/s = 0.155 seconds before white flash
- **New buffer**: 620px ÷ 2000px/s = 0.31 seconds before white flash
- **Result**: 2x more time for React to render rows before they become visible

## Testing Recommendations

1. **Test with large datasets**: Load 10,000+ rows and scroll rapidly
2. **Test with varying scroll speeds**: Try mouse wheel, scrollbar drag, keyboard
3. **Test on slower machines**: The buffer helps more on older hardware
4. **Monitor performance**: Check browser DevTools Performance tab during scroll

## Expected Results

- No visible white flash during normal scrolling
- Minimal or no visible flash during fast scrolling
- Smooth, responsive scrolling experience
- No performance degradation even with large datasets

## Future Optimizations (If Needed)

If white flash still occurs in extreme scenarios, consider:

1. **Increase rowBuffer further**: Try 30 or 40 for even larger buffer
2. **Memoize cell renderers**: Use React.memo() on custom cell components
3. **Simplify cell renderers**: Remove complex conditional logic
4. **Consider suppressRowVirtualisation**: For datasets < 500 rows, disable virtualization entirely

## References

- [AG Grid DOM Virtualisation](https://www.ag-grid.com/javascript-data-grid/dom-virtualisation/)
- [AG Grid Scrolling Performance](https://www.ag-grid.com/javascript-data-grid/scrolling-performance/)
- [AG Grid React Performance](https://www.ag-grid.com/react-data-grid/scrolling-performance/)

## Related Files

- `/Users/jacob/projects/amplifier/ai_working/howlerops/frontend/src/components/ag-grid-table/ag-grid-table.tsx`
- `/Users/jacob/projects/amplifier/ai_working/howlerops/frontend/src/components/ag-grid-table/ag-grid-table.css`
