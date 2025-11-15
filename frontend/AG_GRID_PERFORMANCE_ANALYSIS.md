# ag-Grid Performance Patterns: Preventing White Chunks/Flashing During Fast Scrolling

## Executive Summary

This analysis reveals ag-Grid's sophisticated multi-layer approach to preventing white chunks and flashing during fast scrolling. Their strategy combines debouncing/throttling, intelligent buffering, frame scheduling, DOM recycling, and careful height estimation.

---

## 1. SCROLL HANDLING & DEBOUNCING

### Key Concept: Multi-Timeout Strategy
ag-Grid uses **two different debounce timeouts** for scroll events:

```typescript
// From: gridBodyScrollFeature.ts (lines 32-35)
const SCROLL_DEBOUNCE_TIMEOUT = 100;      // Debounce vertical scroll bar
const SCROLL_END_TIMEOUT = 150;           // Time to consider scroll "ended"
```

### Implementation Details

**Debounce Option (Optional - Performance Mode):**
```typescript
// gridBodyScrollFeature.ts (lines 156-169)
private addVerticalScrollListeners(): void {
    const fakeVScrollComp = this.ctrlsSvc.get('fakeVScrollComp');
    const isDebounce = this.gos.get('debounceVerticalScrollbar');

    // If enabled, debounce the scroll events
    const onVScroll = isDebounce
        ? _debounce(this, this.onVScroll.bind(this, VIEWPORT), SCROLL_DEBOUNCE_TIMEOUT)
        : this.onVScroll.bind(this, VIEWPORT);
    
    this.addManagedElementListeners(this.eBodyViewport, { scroll: onVScroll });
}
```

**Debounce Implementation (Utility):**
```typescript
// From: agStack/utils/function.ts (lines 76-97)
export function _debounce<TArgs extends any[], TContext>(
    bean: { isAlive(): boolean },
    func: (this: TContext, ...args: TArgs) => void,
    delay: number
): (this: TContext, ...args: TArgs) => void {
    let timeout: any;

    return function (this: TContext, ...args: TArgs) {
        const context = this as any;
        window.clearTimeout(timeout);  // Cancel previous timeout

        // Set new timeout - only executes if bean is still alive
        timeout = window.setTimeout(function () {
            if (bean.isAlive()) {
                func.apply(context, args);
            }
        }, delay);
    };
}
```

**Scroll End Detection:**
```typescript
// gridBodyScrollFeature.ts (lines 306-326)
private fireScrollEvent(direction: Direction): void {
    const bodyScrollEvent: WithoutGridCommon<BodyScrollEvent> = {
        type: 'bodyScroll',
        direction: direction === Direction.Horizontal ? 'horizontal' : 'vertical',
        left: this.scrollLeft,
        top: this.scrollTop,
    };
    this.isScrollActive = true;
    this.eventSvc.dispatchEvent(bodyScrollEvent);

    window.clearTimeout(this.scrollTimer);

    // After 150ms with no scroll events, consider scroll "ended"
    this.scrollTimer = window.setTimeout(() => {
        this.scrollTimer = 0;
        this.isScrollActive = false;
        this.eventSvc.dispatchEvent({
            ...bodyScrollEvent,
            type: 'bodyScrollEnd',
        });
    }, SCROLL_END_TIMEOUT);
}
```

**Scroll Source Tracking (Multi-Source Sync):**
```typescript
// gridBodyScrollFeature.ts (lines 57-58, 219-231)
private lastScrollSource: [VerticalScrollSource | null, HorizontalScrollSource | null] = [null, null];

private isControllingScroll(source: HorizontalScrollSource | VerticalScrollSource, direction: Direction): boolean {
    if (this.lastScrollSource[direction] == null) {
        // First scroll source becomes the controlling source
        if (direction === Direction.Vertical) {
            this.lastScrollSource[0] = source as VerticalScrollSource;
        } else {
            this.lastScrollSource[1] = source as HorizontalScrollSource;
        }
        return true;
    }
    
    // Only the controlling source can update scroll position
    return this.lastScrollSource[direction] === source;
}

private resetLastVScrollDebounced: () => void = _debounce(
    this,
    () => (this.lastScrollSource[Direction.Vertical] = null),
    SCROLL_END_TIMEOUT
);
```

**Why This Works:**
- Prevents flickering from scroll events firing from multiple sources simultaneously
- Debouncing (when enabled) reduces render cycles during rapid scroll
- Scroll end detection allows special handling (different render strategy)
- Resets scroll source after scroll completes

---

## 2. BUFFER STRATEGY

### Default Row Buffer

```typescript
// From: gridOptionsDefault.ts (line 62)
rowBuffer: 10,
```

**How Buffer is Used:**
```typescript
// rowRenderer.ts (lines 1328-1337)
private getRowBuffer(): number {
    return this.gos.get('rowBuffer');
}

private getRowBufferInPixels() {
    const rowsToBuffer = this.getRowBuffer();
    const defaultRowHeight = _getRowHeightAsNumber(this.beans);
    return rowsToBuffer * defaultRowHeight;
}
```

**Applied During Viewport Calculation:**
```typescript
// rowRenderer.ts (lines 1353-1377)
const bufferPixels = this.getRowBufferInPixels();
const scrollFeature = this.ctrlsSvc.getScrollFeature();

let firstPixel: number;
let lastPixel: number;

if (suppressRowVirtualisation) {
    firstPixel = pageFirstPixel + divStretchOffset;
    lastPixel = pageLastPixel + divStretchOffset;
} else {
    // Buffer extends the visible range
    firstPixel = Math.max(
        bodyTopPixel + paginationOffset - bufferPixels,  // Above viewport
        pageFirstPixel
    ) + divStretchOffset;
    
    lastPixel = Math.min(
        bodyBottomPixel + paginationOffset + bufferPixels,  // Below viewport
        pageLastPixel
    ) + divStretchOffset;
}
```

**Max Row Restriction (Safety Guard):**
```typescript
// rowRenderer.ts (lines 1408-1417)
const rowLayoutNormal = _isDomLayout(this.gos, 'normal');
const suppressRowCountRestriction = this.gos.get('suppressMaxRenderedRowRestriction');
const rowBufferMaxSize = Math.max(this.getRowBuffer(), 500);  // Min 500 rows

if (rowLayoutNormal && !suppressRowCountRestriction) {
    if (newLast - newFirst > rowBufferMaxSize) {
        newLast = newFirst + rowBufferMaxSize;  // Cap rendered rows
    }
}
```

**Why This Works:**
- Default buffer of 10 rows (typically 300-500px) provides cushion
- Prevents white space when scrolling fast
- Dynamic pixel-based calculation uses actual row heights
- Maximum row cap prevents browser crashes on misconfigured grids

---

## 3. FRAME SCHEDULING & RENDER BATCHING

### AnimationFrameService: The Core Scheduling Engine

```typescript
// From: animationFrameService.ts (lines 18-42)
export class AnimationFrameService extends BeanStub {
    // Priority levels for work scheduling
    private readonly p1: TaskList = { list: [], sorted: false };  // Row backgrounds
    private readonly p2: TaskList = { list: [], sorted: false };  // Cell renderers, hover
    private readonly f1: TaskList = { list: [], sorted: false };  // Framework renderers
    private readonly destroyTasks: (() => void)[] = [];
    
    private ticking = false;
    public active: boolean;
    private scrollGoingDown = true;
    private lastScrollTop = 0;
}
```

**Directional Sorting (Crucial for Fast Scrolling):**
```typescript
// animationFrameService.ts (lines 92-113)
private sortTaskList(taskList: TaskList) {
    if (taskList.sorted) {
        return;
    }

    // Direction-aware sorting: build rows in scroll direction
    const sortDirection = this.scrollGoingDown ? 1 : -1;

    taskList.list.sort((a, b) => {
        // 1. Deferred tasks execute last
        if (a.deferred !== b.deferred) {
            return a.deferred ? -1 : 1;
        }
        // 2. Sort by row index (considering scroll direction)
        //    If scrolling down, render bottom rows first (visible first)
        if (a.index !== b.index) {
            return sortDirection * (b.index - a.index);
        }
        // 3. Within same row, execute by creation order (left-to-right)
        return b.createOrder - a.createOrder;
    });
    taskList.sorted = true;
}

public setScrollTop(scrollTop: number): void {
    this.scrollGoingDown = scrollTop >= this.lastScrollTop;
    if (scrollTop === 0) {
        this.scrollGoingDown = true;
    }
    this.lastScrollTop = scrollTop;
}
```

**Frame Execution with Time Budgeting:**
```typescript
// animationFrameService.ts (lines 121-192)
private executeFrame(millis: number): void {
    const frameStart = Date.now();
    let duration = 0;
    const noMaxMillis = millis <= 0;  // -1 = flush all, >= 0 = time budget

    const scrollFeature = ctrlsSvc.getScrollFeature();

    while (noMaxMillis || duration < millis) {
        // First priority: finish any pending scroll positioning
        const gridBodyDidSomething = scrollFeature.scrollGridIfNeeded();

        if (!gridBodyDidSomething) {
            // Execute p1 tasks (row backgrounds)
            if (p1Tasks.length) {
                this.sortTaskList(p1);
                task = p1Tasks.pop()!.task;
            }
            // Then p2 tasks (cell content)
            else if (p2Tasks.length) {
                this.sortTaskList(p2);
                task = p2Tasks.pop()!.task;
            }
            // Then f1 tasks (framework renderers)
            else if (f1Tasks.length) {
                // Framework tasks loop within time budget
                ...
            }
            // Finally destroy old rows
            else if (destroyTasks.length) {
                task = destroyTasks.pop()!;
            } else {
                break;
            }
            task();
        }
        duration = Date.now() - frameStart;
    }

    // If work remains, request another frame
    if (p1Tasks.length || p2Tasks.length || f1Tasks.length || destroyTasks.length) {
        this.requestFrame();
    } else {
        this.ticking = false;
    }
}
```

**Smart Frame Scheduling:**
```typescript
// animationFrameService.ts (lines 211-216)
private requestFrame(): void {
    const callback = this.executeFrame.bind(this, 60);  // 60ms budget (below 16.67ms frame)
    _requestAnimationFrame(this.beans, callback);
}

public flushAllFrames(): void {
    if (!this.active) {
        return;
    }
    this.executeFrame(-1);  // Execute all tasks immediately (no time limit)
}
```

**Integration with Row Renderer:**
```typescript
// rowRenderer.ts (lines 1536-1540)
const useAnimationFrameForCreate = afterScroll && !this.printLayout && !!this.beans.animationFrameSvc?.active;

const res = new RowCtrl(rowNode, this.beans, animate, useAnimationFrameForCreate, this.printLayout);
```

**Why This Works:**
- Directional rendering: users see relevant rows (below when scrolling down) first
- Time-boxed execution: prevents frame drops by spreading work across frames
- Priority levels: ensures critical UI elements render before less critical ones
- Smart scheduling: doesn't over-commit work that can't fit in frame budget

---

## 4. DOM REUSE & ROW RECYCLING

### The Recycling Pattern

**Row Reuse on Model Update:**
```typescript
// rowRenderer.ts (lines 969-990)
private getRowsToRecycle(): RowCtrlByRowNodeIdMap {
    // Remove stub nodes first (can't be reused)
    const stubNodeIndexes: string[] = [];
    for (const index of Object.keys(this.rowCtrlsByRowIndex)) {
        const rowCtrl = this.rowCtrlsByRowIndex[index as any];
        const stubNode = rowCtrl.rowNode.id == null;
        if (stubNode) {
            stubNodeIndexes.push(index);
        }
    }
    this.removeRowCtrls(stubNodeIndexes);

    // Reindex existing rows by ID for reuse
    const ctrlsByIdMap: RowCtrlByRowNodeIdMap = {};
    for (const rowCtrl of Object.values(this.rowCtrlsByRowIndex)) {
        const rowNode = rowCtrl.rowNode;
        ctrlsByIdMap[rowNode.id!] = rowCtrl;
    }
    this.rowCtrlsByRowIndex = {};

    return ctrlsByIdMap;  // Return for potential reuse
}
```

**Row Creation or Reuse During Recycling:**
```typescript
// rowRenderer.ts (lines 1240-1282)
private createOrUpdateRowCtrl(
    rowIndex: number,
    rowsToRecycle: { [key: string]: RowCtrl | null } | null | undefined,
    animate: boolean,
    afterScroll: boolean
): void {
    let rowNode: RowNode | undefined;
    let rowCtrl: RowCtrl | null = this.rowCtrlsByRowIndex[rowIndex];

    // Try to get from recycled pool first
    if (!rowCtrl) {
        rowNode = this.rowModel.getRow(rowIndex);
        if (_exists(rowNode) && _exists(rowsToRecycle) && rowsToRecycle[rowNode.id!] && 
            rowNode.alreadyRendered) {
            // Reuse existing row control!
            rowCtrl = rowsToRecycle[rowNode.id!];
            rowsToRecycle[rowNode.id!] = null;  // Mark as used
        }
    }

    const creatingNewRowCtrl = !rowCtrl;

    if (creatingNewRowCtrl) {
        if (!rowNode) {
            rowNode = this.rowModel.getRow(rowIndex);
        }
        if (_exists(rowNode)) {
            // Create new row control
            rowCtrl = this.createRowCon(rowNode, animate, afterScroll);
        } else {
            return;
        }
    }

    if (rowNode) {
        rowNode.alreadyRendered = true;
    }

    this.rowCtrlsByRowIndex[rowIndex] = rowCtrl!;
}
```

**Two-Phase Destruction (Smooth Animations):**
```typescript
// rowRenderer.ts (lines 1284-1326)
private destroyRowCtrls(rowCtrlsMap: RowCtrlIdMap | null | undefined, animate: boolean): void {
    const executeInAWhileFuncs: (() => void)[] = [];
    if (rowCtrlsMap) {
        for (const rowCtrl of Object.values(rowCtrlsMap)) {
            if (!rowCtrl) {
                continue;
            }

            // Try to cache for detail rows (master-detail pattern)
            if (this.cachedRowCtrls && rowCtrl.isCacheable()) {
                this.cachedRowCtrls.addRow(rowCtrl);
                continue;
            }

            // Phase 1: Trigger fade-out animation
            rowCtrl.destroyFirstPass(!animate);
            
            if (animate) {
                const instanceId = rowCtrl.instanceId;
                // Keep as "zombie" row during animation
                this.zombieRowCtrls[instanceId] = rowCtrl;
                executeInAWhileFuncs.push(() => {
                    // Phase 2: Remove from DOM after animation completes
                    rowCtrl.destroySecondPass();
                    delete this.zombieRowCtrls[instanceId];
                });
            } else {
                rowCtrl.destroySecondPass();
            }
        }
    }
    
    // Execute phase 2 after animation timeout
    if (animate) {
        executeInAWhileFuncs.push(() => {
            if (this.isAlive()) {
                this.updateAllRowCtrls();
                this.dispatchDisplayedRowsChanged();
            }
        });
        window.setTimeout(() => {
            for (const func of executeInAWhileFuncs) {
                func();
            }
        }, ROW_ANIMATION_TIMEOUT);  // 400ms
    }
}
```

**Row Cache for Detail Rows:**
```typescript
// rowRenderer.ts (lines 1593-1660)
class RowCtrlCache {
    private entriesMap: RowCtrlByRowNodeIdMap = {};
    private readonly entriesList: RowCtrl[] = [];
    private readonly maxCount: number;

    constructor(maxCount: number) {
        this.maxCount = maxCount;
    }

    public addRow(rowCtrl: RowCtrl): void {
        this.entriesMap[rowCtrl.rowNode.id!] = rowCtrl;
        this.entriesList.push(rowCtrl);
        rowCtrl.setCached(true);

        // LRU eviction: remove oldest when cache full
        if (this.entriesList.length > this.maxCount) {
            const rowCtrlToDestroy = this.entriesList[0];
            rowCtrlToDestroy.destroyFirstPass();
            rowCtrlToDestroy.destroySecondPass();
            this.removeFromCache(rowCtrlToDestroy);
        }
    }

    public getRow(rowNode: RowNode): RowCtrl | null {
        if (rowNode?.id == null) {
            return null;
        }
        const res = this.entriesMap[rowNode.id];
        if (!res) {
            return null;
        }
        this.removeFromCache(res);
        res.setCached(false);
        return res;
    }
}
```

**Zombie Row Management (Visible During Animations):**
```typescript
// rowRenderer.ts (lines 211-222)
private updateAllRowCtrls(): void {
    const liveList = Object.values(this.rowCtrlsByRowIndex);
    const zombieList = Object.values(this.zombieRowCtrls);  // Include zombie rows in render
    const cachedList = this.cachedRowCtrls?.getEntries() ?? [];

    if (zombieList.length > 0 || cachedList.length > 0) {
        // Only spread if we need to (performance optimization)
        this.allRowCtrls = [...liveList, ...zombieList, ...cachedList];
    } else {
        this.allRowCtrls = liveList;
    }
}
```

**Why This Works:**
- Reuses DOM elements instead of creating/destroying (expensive operations)
- Two-phase destruction allows smooth fade-out animations
- Zombie rows remain visible during animation, preventing flashing
- Row cache keeps frequently accessed rows ready (detail rows)
- LRU eviction ensures memory doesn't grow unbounded

---

## 5. ROW HEIGHT ESTIMATION & LAZY MEASUREMENT

**Viewport Height Calculation with Caching:**
```typescript
// gridBodyScrollFeature.ts (lines 441-475)
public getVScrollPosition(): VerticalScrollPosition {
    if (!this.isVerticalPositionInvalidated) {
        const { lastOffsetHeight, lastScrollTop } = this;
        // Return cached value to avoid DOM reads
        return {
            top: lastScrollTop,
            bottom: lastScrollTop + lastOffsetHeight,
        };
    }

    this.isVerticalPositionInvalidated = false;

    const { scrollTop, offsetHeight } = this.eBodyViewport;
    this.lastScrollTop = scrollTop;
    this.lastOffsetHeight = offsetHeight;

    return {
        top: scrollTop,
        bottom: scrollTop + offsetHeight,
    };
}

// Approximate version to avoid forcing reflows
public getApproximateVScollPosition(): VerticalScrollPosition {
    if (this.lastScrollTop >= 0 && this.lastOffsetHeight >= 0) {
        return {
            top: this.scrollTop,
            bottom: this.scrollTop + this.lastOffsetHeight,
        };
    }
    return this.getVScrollPosition();
}
```

**Lazy Height Validation:**
```typescript
// rowRenderer.ts (lines 1455-1480)
private ensureAllRowsInRangeHaveHeightsCalculated(topPixel: number, bottomPixel: number): boolean {
    const pinnedRowHeightsChanged = this.pinnedRowModel?.ensureRowHeightsValid();
    const stickyHeightsChanged = this.stickyRowFeature?.ensureRowHeightsValid();
    const { pageBounds, rowModel } = this;
    
    // Lazy height calculation - only for visible range
    const rowModelHeightsChanged = rowModel.ensureRowHeightsValid(
        topPixel,
        bottomPixel,
        pageBounds.getFirstRow(),
        pageBounds.getLastRow()
    );
    
    if (rowModelHeightsChanged || stickyHeightsChanged) {
        this.eventSvc.dispatchEvent({
            type: 'recalculateRowBounds',
        });
    }

    if (stickyHeightsChanged || rowModelHeightsChanged || pinnedRowHeightsChanged) {
        this.updateContainerHeights();
        return true;
    }
    return false;
}
```

**Height Recalculation Loop (Handles Dynamic Heights):**
```typescript
// rowRenderer.ts (lines 1357-1384)
let rowHeightsChanged = false;
let firstPixel: number;
let lastPixel: number;

do {
    const paginationOffset = pageBounds.getPixelOffset();
    const { pageFirstPixel, pageLastPixel } = pageBounds.getCurrentPagePixelRange();
    
    // Calculate range with buffer
    firstPixel = Math.max(bodyTopPixel + paginationOffset - bufferPixels, pageFirstPixel) + divStretchOffset;
    lastPixel = Math.min(bodyBottomPixel + paginationOffset + bufferPixels, pageLastPixel) + divStretchOffset;

    // If dynamic heights changed, recalculate
    rowHeightsChanged = this.ensureAllRowsInRangeHaveHeightsCalculated(firstPixel, lastPixel);
} while (rowHeightsChanged);  // Loop until stable
```

**Why This Works:**
- Caching viewport dimensions avoids repeated DOM reads
- Lazy measurement only calculates heights for visible rows
- Loop handles custom row heights (autoHeight, getRowHeight)
- Prevents infinite loops with attempt counter

---

## 6. ADDITIONAL PERFORMANCE OPTIMIZATIONS

### Viewport Size Change Handling
```typescript
// viewportSizeFeature.ts (lines 38-84)
private listenForResize(): void {
    const listener = () => {
        // Use requestAnimationFrame to prevent infinite loops
        // ResizeObserver can trigger cascading resize events
        _requestAnimationFrame(beans, () => {
            this.onCenterViewportResized();
        });
    };

    centerContainerCtrl.registerViewportResizeListener(listener);
    gridBodyCtrl.registerBodyViewportResizeListener(listener);
}
```

### Row Rendering Optimization After Scroll
```typescript
// rowRenderer.ts (lines 1007-1012, 1018-1067)
private onBodyScroll(e: BodyScrollEvent) {
    if (e.direction !== 'vertical') {
        return;
    }
    this.redraw({ afterScroll: true });  // Special handling flag
}

public redraw(params: { afterScroll?: boolean } = {}) {
    const { focusSvc, animationFrameSvc } = this.beans;
    const { afterScroll } = params;

    // Only use animation frames when scrolling (smooth experience)
    const useAnimationFrameForCreate = afterScroll && !this.printLayout && 
        !!this.beans.animationFrameSvc?.active;
    
    // Prevent animation frame creation except during scroll
}
```

### Column Viewport Service (Horizontal Virtualization)
```typescript
// From rowContainerCtrl.ts (lines 406-408, 432-437)
public onDisplayedColumnsChanged(): void {
    this.forContainers(['center'], () => this.onHorizontalViewportChanged());
}

public onHorizontalViewportChanged(afterScroll: boolean = false): void {
    const scrollWidth = this.getCenterWidth();
    const scrollPosition = this.getCenterViewportScrollLeft();
    
    // Notify column controller to virtualize columns too
    this.beans.colViewport.setScrollPosition(scrollWidth, scrollPosition, afterScroll);
}
```

---

## 7. KEY GRID OPTIONS FOR CONFIGURATION

```typescript
rowBuffer: 10,                              // Rows to render above/below viewport
debounceVerticalScrollbar: false,          // Enable debounce (optional)
suppressRowVirtualisation: false,          // Disable virtual scrolling
suppressAnimationFrame: false,             // Disable frame scheduling
suppressMaxRenderedRowRestriction: false,  // Allow unlimited rows (risky)
animateRows: true,                         // Enable row animations
suppressColumnVirtualisation: false,       // Disable column virtualization
domLayout: 'normal',                       // Use virtual scrolling layout
embedFullWidthRows: false,                 // Don't embed full-width rows
```

---

## IMPLEMENTATION RECOMMENDATIONS FOR SQL STUDIO

### 1. **Adopt the Frame Scheduling Pattern**
Create a similar `AnimationFrameService` that:
- Schedules row creation in priority levels (backgrounds → content → framework)
- Sorts tasks by scroll direction
- Budgets time within 60ms frames

### 2. **Implement Row Recycling**
- Keep a map of `rowIndex → RowCtrl` 
- On scroll, reuse existing row controls instead of destroying/creating
- Two-phase destruction for animations

### 3. **Buffer Strategy**
- Default: 10 rows worth of buffer pixels
- Above and below visible viewport
- Adjust based on row height

### 4. **Smart Debouncing**
```typescript
const SCROLL_DEBOUNCE_TIMEOUT = 100;
const SCROLL_END_TIMEOUT = 150;

// Debounce scroll renderer only if needed (option)
const onScroll = enableDebounce 
    ? _debounce(this, handleScroll, SCROLL_DEBOUNCE_TIMEOUT)
    : handleScroll;
```

### 5. **Prevent Multiple Scroll Sources**
Track which element initiated scroll, only that element updates position for next 150ms.

### 6. **Cache Viewport Dimensions**
- Don't read `offsetHeight`/`scrollTop` every scroll event
- Cache and invalidate on relevant changes only

### 7. **Lazy Height Measurement**
- Only calculate heights for visible + buffer rows
- Support dynamic heights with re-calculation loop

### 8. **Directional Rendering**
- When scrolling down, render lower rows first (visible)
- When scrolling up, render upper rows first
- Reduces user perception of lag

---

## Conclusion

ag-Grid prevents white chunks/flashing through a sophisticated combination of:
1. **Event debouncing** - Reduces redundant renders
2. **Smart buffering** - Pre-renders rows beyond viewport
3. **Frame scheduling** - Prioritizes visible content
4. **DOM recycling** - Reuses expensive DOM operations
5. **Lazy measurement** - Only calculates what's needed
6. **Directional rendering** - Shows most important content first
7. **Scroll source tracking** - Prevents conflicting updates

The key insight: **Prevention through prediction** - render what's likely needed before the user sees it, while managing DOM operations efficiently.

