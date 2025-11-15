# ag-Grid Source Files Reference

Complete mapping of key ag-Grid files and their purposes for preventing white chunks/flashing during fast scrolling.

## Core Scroll Handling

### GridBodyScrollFeature
**File:** `packages/ag-grid-community/src/gridBodyComp/gridBodyScrollFeature.ts`

Key responsibilities:
- Scroll event debouncing (100ms, optional)
- Scroll end detection (150ms timeout)
- Multi-source scroll synchronization
- Scroll position validation
- Prevents elastic scroll on iOS

**Key Methods:**
- `addVerticalScrollListeners()` - Line 156-169
- `onVScroll()` - Line 250-286
- `fireScrollEvent()` - Line 306-326
- `getVScrollPosition()` - Line 441-461
- `getApproximateVScollPosition()` - Line 467-475

**Key Constants:**
- SCROLL_DEBOUNCE_TIMEOUT = 100ms
- SCROLL_END_TIMEOUT = 150ms

---

## Row Rendering & Virtualization

### RowRenderer
**File:** `packages/ag-grid-community/src/rendering/rowRenderer.ts`

Key responsibilities:
- Virtual scrolling implementation
- Row recycling management
- Buffer calculation
- Row height measurement
- DOM reuse strategy

**Key Methods:**
- `workOutFirstAndLastRowsToRender()` - Line 1339-1432 (Main virtualization logic)
- `recycleRows()` - Line 1140-1179 (DOM reuse)
- `createOrUpdateRowCtrl()` - Line 1240-1282 (Row reuse vs creation)
- `destroyRowCtrls()` - Line 1284-1326 (Two-phase destruction)
- `ensureAllRowsInRangeHaveHeightsCalculated()` - Line 1455-1480 (Lazy height)
- `redraw()` - Line 1018-1067 (After-scroll rendering)
- `getRowBufferInPixels()` - Line 1332-1337

**Key Constants:**
- ROW_ANIMATION_TIMEOUT = 400ms (Fade-out animation)

**Row Cache (RowCtrlCache):**
- Lines 1593-1660
- LRU cache for detail rows
- Configurable max size

---

## Animation Frame Service (Frame Scheduling)

### AnimationFrameService
**File:** `packages/ag-grid-community/src/misc/animationFrameService.ts`

Key responsibilities:
- Work scheduling across frames
- Priority-based task execution
- Time budgeting (60ms per frame)
- Directional task sorting
- Scroll position processing

**Key Methods:**
- `executeFrame()` - Line 121-192 (Core execution loop)
- `sortTaskList()` - Line 92-113 (Directional sorting)
- `setScrollTop()` - Line 43-53 (Scroll direction detection)
- `createTask()` - Line 70-85 (Task scheduling)
- `schedule()` - Line 201-209 (Request next frame)
- `flushAllFrames()` - Line 194-199 (Immediate execution)

**Task Priority Levels:**
1. p1 - Row backgrounds
2. p2 - Cell renderers & hover
3. f1 - Framework renderers
4. destroyTasks - Cleanup

**Key Constants:**
- Frame budget: 60ms (in requestFrame, line 214)

---

## Row Container Control

### RowContainerCtrl
**File:** `packages/ag-grid-community/src/gridBodyComp/rowContainer/rowContainerCtrl.ts`

Key responsibilities:
- Manages row container components
- Horizontal virtualization
- Scroll partnership coordination
- Container visibility management

**Key Methods:**
- `onDisplayedRowsChanged()` - Line 504-529 (Updates visible rows)
- `onHorizontalViewportChanged()` - Line 432-437 (Column viewport sync)

---

## Utility Functions (Debounce & Throttle)

### Function Utilities
**File:** `packages/ag-grid-community/src/agStack/utils/function.ts`

**Debounce Implementation:** Lines 76-97
```typescript
export function _debounce<TArgs extends any[], TContext>(
    bean: { isAlive(): boolean },
    func: (this: TContext, ...args: TArgs) => void,
    delay: number
): (this: TContext, ...args: TArgs) => void
```

**Throttle Implementation:** Lines 104-120
```typescript
export function _throttle(func: (...args: any[]) => void, wait: number)
```

**Batch Call Implementation:** Lines 38-68
- Batches functions for setTimeout or requestAnimationFrame

---

## Viewport Management

### ViewportSizeFeature
**File:** `packages/ag-grid-community/src/gridBodyComp/viewportSizeFeature.ts`

Key responsibilities:
- Viewport dimension tracking
- Resize event handling with RAF debouncing
- Scroll visibility detection
- Height change detection

**Key Methods:**
- `onCenterViewportResized()` - Line 64-84 (RAF-debounced resize)
- `checkViewportAndScrolls()` - Line 88-99 (Comprehensive viewport check)
- `checkBodyHeight()` - Line 105-115 (Height change detection)

---

## Row Control

### RowCtrl
**File:** `packages/ag-grid-community/src/rendering/row/rowCtrl.ts`

Key responsibilities:
- Individual row lifecycle management
- Cell control management
- Animation handling
- Focus management

**Key Pattern:**
- Two-phase destruction (destroyFirstPass, destroySecondPass)
- Animation tracking (fadeInAnimation, slideInAnimation)

---

## Grid Options & Configuration

### Grid Options Default
**File:** `packages/ag-grid-community/src/gridOptionsDefault.ts`

**Key Configuration Options:**
- Line 62: `rowBuffer: 10` - Default overscan
- Line 99: `suppressMaxRenderedRowRestriction: false` - Row cap safety
- Line 100: `suppressRowVirtualisation: false` - Enable virtualization
- Line 142: `debounceVerticalScrollbar: false` - Optional debounce
- Line 146: `suppressAnimationFrame: false` - Frame scheduling

---

## File Dependency Flow

```
GridBodyScrollFeature
  ├── Listens to scroll events
  ├── Debounces (optional)
  └── Calls RowRenderer.redraw()

RowRenderer
  ├── Calls workOutFirstAndLastRowsToRender()
  ├── Calculates buffer range
  ├── Calls recycleRows()
  └── Creates/reuses RowCtrl instances

AnimationFrameService
  ├── Queues RowCtrl creation tasks
  ├── Sorts by scroll direction
  ├── Executes with time budget
  └── Schedules next frame if needed

ViewportSizeFeature
  ├── Monitors viewport changes
  ├── Debounces with RAF
  └── Triggers height recalculation
```

---

## Key Data Structures

### Row Control Maps
```typescript
// From RowRenderer
private rowCtrlsByRowIndex: Record<number, RowCtrl> = {};    // Current: index → ctrl
private rowsToRecycle: Record<string, RowCtrl> = {};          // Recyclable: id → ctrl
private zombieRowCtrls: Record<RowCtrlInstanceId, RowCtrl> = {}; // Animating: id → ctrl
private cachedRowCtrls: RowCtrlCache;                         // Cached: LRU cache
```

### Task List
```typescript
// From AnimationFrameService
private readonly p1: TaskList = { list: [], sorted: false };
private readonly p2: TaskList = { list: [], sorted: false };
private readonly f1: TaskList = { list: [], sorted: false };
private readonly destroyTasks: (() => void)[] = [];
```

### Task Item
```typescript
interface TaskItem {
    task: () => void;
    index: number;           // Row index for sorting
    createOrder: number;     // Creation order for tiebreaker
    deferred: boolean;       // Execute last if true
}
```

---

## Testing & Debugging Hooks

### Events to Monitor
- `bodyScroll` - Scroll event fired (gridBodyScrollFeature.ts:306)
- `bodyScrollEnd` - Scroll ended (gridBodyScrollFeature.ts:321)
- `displayedRowsChanged` - Visible rows updated (rowRenderer.ts:1181)
- `viewportChanged` - Viewport range changed (rowRenderer.ts:1426)

### Metrics to Track
1. **Scroll events per second** - Should be reduced by debounce
2. **Row creation rate** - Should decrease with recycling
3. **DOM nodes count** - Should remain constant (recycling)
4. **Frame time** - Should stay under 16.67ms for 60fps
5. **Scroll buffer size** - Visible range ± buffer

---

## Integration Checklist

- [ ] Implement gridBodyScrollFeature scroll handling
- [ ] Add debounce to scroll events (100ms)
- [ ] Implement scroll end detection (150ms)
- [ ] Create AnimationFrameService with priorities
- [ ] Implement row recycling by ID (not index)
- [ ] Add directional task sorting
- [ ] Implement time budgeting in frame execution
- [ ] Add two-phase destruction with animations
- [ ] Cache viewport dimensions
- [ ] Implement lazy height measurement
- [ ] Add row buffer (10 rows default)
- [ ] Set max row cap (500 rows safety)

---

## Related Issues in ag-Grid

The following ag-Grid GitHub issues document these performance patterns:
- AG-3274: Row DOM sync during tab switches
- AG-7018: First data rendered event timing
- ResizeObserver cascade issues (viewportSizeFeature.ts comment)

