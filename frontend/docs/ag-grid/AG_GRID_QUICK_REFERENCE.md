# ag-Grid Performance Patterns - Quick Reference

## Problem: White Chunks/Flashing During Fast Scrolling

## Solution: Multi-Layer Prevention Strategy

### 1. Scroll Debouncing (100ms)
- Reduces redundant render cycles during rapid scrolling
- Optional per gridOption: `debounceVerticalScrollbar`
- **Key:** Cancel pending timeout, wait for scroll to settle

### 2. Buffer Strategy (10 rows)
- Default overscan: 10 rows above + 10 rows below visible viewport
- Converted to pixels based on actual row heights
- Caps max rendered rows at 500 (safety guard)
- **Key:** Pre-render content before user scrolls to it

### 3. Frame Scheduling (requestAnimationFrame)
- Batches row creation tasks into priority levels:
  1. P1: Row backgrounds
  2. P2: Cell renderers & hover states
  3. F1: Framework-specific renderers
  4. Destroy: Cleanup old rows
- **Key:** Directional rendering - render visible rows first based on scroll direction

### 4. DOM Recycling
- Reuse row DOM elements instead of create/destroy
- ID-based mapping for reuse on data model updates
- Two-phase destruction: fade-out (400ms) then remove
- Zombie rows stay visible during animations
- **Key:** Expensive DOM operations done once, reused many times

### 5. Lazy Height Measurement
- Cache viewport dimensions (avoid repeated DOM reads)
- Only measure heights for visible + buffer range
- Loop to handle dynamic row heights (autoHeight)
- **Key:** Avoid forced reflows on every scroll event

### 6. Scroll Source Tracking
- One scroll source "controls" for next 150ms
- Prevents conflicting updates from multiple sources
- Resets after scroll ends
- **Key:** Sync virtual and fake scrollbars without fighting

---

## Implementation Checklist

- [ ] Debounce scroll handler (100ms)
- [ ] Detect scroll end (150ms timeout)
- [ ] Track scroll direction
- [ ] Implement row buffer (10 rows default)
- [ ] Create AnimationFrameService with priority queues
- [ ] Implement directional task sorting
- [ ] Add time-budgeting (60ms max per frame)
- [ ] Implement row recycling by ID
- [ ] Add two-phase destruction with animations
- [ ] Cache viewport dimensions
- [ ] Lazy measure only visible row range
- [ ] Handle dynamic row heights

---

## Key Numbers (From ag-Grid)

| Parameter | Value | Purpose |
|-----------|-------|---------|
| Debounce timeout | 100ms | Reduce scroll handler calls |
| Scroll end timeout | 150ms | Consider scroll complete |
| Row buffer | 10 rows | Overscan above/below viewport |
| Max rows rendered | 500 | Safety cap to prevent crashes |
| Frame budget | 60ms | Time for one animation frame |
| Animation duration | 400ms | Row fade-out animation |
| Resize debounce | RAF | Prevent cascading resize events |

---

## Critical Code Patterns

### Pattern 1: Debounce with Bean Lifecycle
```typescript
const resetScroll = _debounce(this, () => {
    this.scrollSource = null;
}, 150);

// Cancel previous, queue new
resetScroll();
```

### Pattern 2: Directional Task Sorting
```typescript
tasks.sort((a, b) => {
    if (scrollGoingDown) {
        return b.rowIndex - a.rowIndex;  // Bottom first
    } else {
        return a.rowIndex - b.rowIndex;  // Top first
    }
});
```

### Pattern 3: Row Reuse by ID
```typescript
// Get existing control by ID, not index
const rowCtrl = rowsToRecycle[rowNode.id];
if (rowCtrl && rowNode.alreadyRendered) {
    reuse(rowCtrl);  // Don't recreate
}
```

### Pattern 4: Viewport Caching
```typescript
if (!isVerticalPositionInvalidated) {
    return cachedPosition;  // Avoid DOM read
}
isVerticalPositionInvalidated = false;
cachedPosition = eBodyViewport.scrollTop;
```

### Pattern 5: Two-Phase Destruction
```typescript
// Phase 1: Start animation
rowCtrl.destroyFirstPass(!animate);

if (animate) {
    zombieRowCtrls[id] = rowCtrl;
    // Phase 2: Remove after animation
    setTimeout(() => rowCtrl.destroySecondPass(), 400);
}
```

---

## Common Pitfalls

1. **Creating/destroying rows on every scroll** - Use recycling instead
2. **Measuring all row heights upfront** - Lazy measure visible + buffer only
3. **No time budget for frame work** - Cap execution time per frame
4. **Ignoring scroll direction** - Render visible content first (directional)
5. **Multiple scroll sources fighting** - Track & lock to controlling source
6. **DOM reads in hot paths** - Cache dimensions, invalidate selectively
7. **Destroying DOM immediately** - Use animations (fade-out while visible)

---

## Performance Debugging Tips

### Check for white chunks:
- Increase `rowBuffer` (more overscan = safer but slower)
- Verify animation frame scheduling is active
- Ensure rows are recycled by ID, not recreated

### Check for flashing:
- Verify two-phase destruction is implemented
- Check that zombie rows stay in DOM during animation
- Ensure scroll debounce is preventing rapid re-renders

### Check for lag:
- Verify directional rendering (bottom-first when scrolling down)
- Check time budget isn't exceeded (60ms)
- Ensure lazy height measurement (not all rows upfront)

### Measure improvements:
- Monitor "Rows created per scroll event" (should decrease)
- Monitor "DOM nodes in tree" (should stay constant)
- Monitor "Frame time" (should stay below 16.67ms for 60fps)

