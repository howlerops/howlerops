# Implementation Guide: ag-Grid Performance Patterns

## Overview

This guide provides step-by-step instructions to implement the key performance patterns ag-Grid uses to prevent white chunks and flashing during fast scrolling.

## Phase 1: Foundation (Debouncing & Source Tracking)

### Step 1.1: Create Debounce Utility
```typescript
// utils/debounce.ts
export function debounce<T extends (...args: any[]) => any>(
    func: T,
    delay: number,
    context?: any
): (...args: Parameters<T>) => void {
    let timeout: NodeJS.Timeout;
    
    return function(...args: Parameters<T>) {
        clearTimeout(timeout);
        timeout = setTimeout(() => {
            func.apply(context, args);
        }, delay);
    };
}
```

### Step 1.2: Implement Scroll Source Tracking
```typescript
// GridScroller.ts
private lastScrollSource: HTMLElement | null = null;
private scrollEndTimeout: NodeJS.Timeout | null = null;

private isControllingScroll(source: HTMLElement): boolean {
    if (this.lastScrollSource === null) {
        this.lastScrollSource = source;
        return true;
    }
    return this.lastScrollSource === source;
}

private resetScrollSource(): void {
    this.lastScrollSource = null;
}

private onScroll(source: HTMLElement): void {
    if (!this.isControllingScroll(source)) {
        return;
    }
    
    // Handle scroll...
    
    // Schedule reset after 150ms
    if (this.scrollEndTimeout) {
        clearTimeout(this.scrollEndTimeout);
    }
    this.scrollEndTimeout = setTimeout(
        () => this.resetScrollSource(),
        150
    );
}
```

## Phase 2: Buffering & Viewport Calculation

### Step 2.1: Define Row Buffer Configuration
```typescript
// GridConfig.ts
interface VirtualScrollConfig {
    rowBuffer: number;        // Default: 10
    rowHeight: number;        // Average row height
    maxRenderedRows: number;  // Default: 500 (safety cap)
    suppressVirtualization: boolean; // Default: false
}

const DEFAULT_CONFIG: VirtualScrollConfig = {
    rowBuffer: 10,
    rowHeight: 35,
    maxRenderedRows: 500,
    suppressVirtualization: false,
};
```

### Step 2.2: Implement Viewport Calculation
```typescript
// VirtualScrollCalculator.ts
export class VirtualScrollCalculator {
    constructor(private config: VirtualScrollConfig) {}
    
    calculateVisibleRange(
        scrollTop: number,
        viewportHeight: number,
        totalHeight: number
    ): { start: number; end: number; bufferStart: number; bufferEnd: number } {
        const bufferPixels = this.config.rowBuffer * this.config.rowHeight;
        
        const bufferStart = Math.max(0, scrollTop - bufferPixels);
        const bufferEnd = Math.min(totalHeight, scrollTop + viewportHeight + bufferPixels);
        
        const start = this.pixelsToRowIndex(bufferStart);
        const end = this.pixelsToRowIndex(bufferEnd);
        
        // Apply safety cap
        if (end - start > this.config.maxRenderedRows) {
            return {
                start,
                end: start + this.config.maxRenderedRows,
                bufferStart,
                bufferEnd,
            };
        }
        
        return { start, end, bufferStart, bufferEnd };
    }
    
    private pixelsToRowIndex(pixels: number): number {
        return Math.floor(pixels / this.config.rowHeight);
    }
}
```

## Phase 3: Frame Scheduling

### Step 3.1: Create Animation Frame Service
```typescript
// AnimationFrameService.ts
interface TaskItem {
    fn: () => void;
    index: number;        // Row index for sorting
    createOrder: number;  // Creation sequence
}

export class AnimationFrameService {
    private p1Tasks: TaskItem[] = [];  // Backgrounds
    private p2Tasks: TaskItem[] = [];  // Content
    private p3Tasks: TaskItem[] = [];  // Framework
    private destroyTasks: (() => void)[] = [];
    
    private ticking = false;
    private taskCounter = 0;
    private scrollDirection: 'up' | 'down' = 'down';
    private lastScrollTop = 0;
    
    setScrollTop(scrollTop: number): void {
        this.scrollDirection = scrollTop > this.lastScrollTop ? 'down' : 'up';
        this.lastScrollTop = scrollTop;
    }
    
    addTask(fn: () => void, priority: 'p1' | 'p2' | 'p3', index: number): void {
        const task: TaskItem = {
            fn,
            index,
            createOrder: ++this.taskCounter,
        };
        
        const list = {
            p1: this.p1Tasks,
            p2: this.p2Tasks,
            p3: this.p3Tasks,
        }[priority];
        
        list.push(task);
        this.schedule();
    }
    
    addDestroyTask(fn: () => void): void {
        this.destroyTasks.push(fn);
        this.schedule();
    }
    
    private schedule(): void {
        if (!this.ticking) {
            this.ticking = true;
            requestAnimationFrame((time) => this.executeFrame(time));
        }
    }
    
    private executeFrame(now: number): void {
        const frameStart = now;
        const frameBudget = 60; // ms
        
        while (Date.now() - frameStart < frameBudget) {
            // Execute p1 tasks
            if (this.p1Tasks.length > 0) {
                this.sortTasks(this.p1Tasks);
                this.p1Tasks.pop()!.fn();
                continue;
            }
            
            // Execute p2 tasks
            if (this.p2Tasks.length > 0) {
                this.sortTasks(this.p2Tasks);
                this.p2Tasks.pop()!.fn();
                continue;
            }
            
            // Execute p3 tasks
            if (this.p3Tasks.length > 0) {
                this.sortTasks(this.p3Tasks);
                this.p3Tasks.pop()!.fn();
                continue;
            }
            
            // Execute destroy tasks
            if (this.destroyTasks.length > 0) {
                this.destroyTasks.pop()!();
                continue;
            }
            
            break;
        }
        
        // Schedule next frame if work remains
        if (
            this.p1Tasks.length > 0 ||
            this.p2Tasks.length > 0 ||
            this.p3Tasks.length > 0 ||
            this.destroyTasks.length > 0
        ) {
            requestAnimationFrame((time) => this.executeFrame(time));
        } else {
            this.ticking = false;
        }
    }
    
    private sortTasks(tasks: TaskItem[]): void {
        const direction = this.scrollDirection === 'down' ? -1 : 1;
        
        tasks.sort((a, b) => {
            // Sort by index (considering direction)
            if (a.index !== b.index) {
                return direction * (b.index - a.index);
            }
            // Tiebreak by creation order
            return b.createOrder - a.createOrder;
        });
    }
}
```

## Phase 4: Row Recycling

### Step 4.1: Implement Row Controller
```typescript
// RowController.ts
export interface Row {
    id: string;
    data: any;
    element?: HTMLElement;
}

export class RowController {
    private rowsByIndex: Map<number, Row> = new Map();
    private rowsById: Map<string, Row> = new Map();
    private zombieRows: Set<Row> = new Set();
    
    updateRows(
        newRows: Row[],
        startIndex: number,
        animate: boolean = false
    ): void {
        const rowsToKeep = new Set<Row>();
        
        for (let i = 0; i < newRows.length; i++) {
            const newRow = newRows[i];
            const rowIndex = startIndex + i;
            
            // Try to reuse existing row
            let row = this.rowsByIndex.get(rowIndex);
            if (row && row.id !== newRow.id) {
                row = this.rowsById.get(newRow.id);
            }
            
            if (row && row.id === newRow.id) {
                // Reuse existing row
                row.data = newRow.data;
                this.rowsByIndex.set(rowIndex, row);
                rowsToKeep.add(row);
            } else {
                // Create new row
                row = newRow;
                this.rowsByIndex.set(rowIndex, row);
                this.rowsById.set(row.id, row);
                rowsToKeep.add(row);
                
                // Render new row
                this.renderRow(row, rowIndex, animate);
            }
        }
        
        // Remove rows no longer needed
        this.removeUnusedRows(rowsToKeep, animate);
    }
    
    private removeUnusedRows(rowsToKeep: Set<Row>, animate: boolean): void {
        const toRemove: number[] = [];
        
        for (const [index, row] of this.rowsByIndex) {
            if (!rowsToKeep.has(row)) {
                toRemove.push(index);
            }
        }
        
        for (const index of toRemove) {
            const row = this.rowsByIndex.get(index)!;
            this.destroyRow(row, animate);
            this.rowsByIndex.delete(index);
        }
    }
    
    private renderRow(row: Row, index: number, animate: boolean): void {
        // Implement row rendering
    }
    
    private destroyRow(row: Row, animate: boolean): void {
        if (animate) {
            // Phase 1: Start animation
            if (row.element) {
                row.element.style.opacity = '0';
                this.zombieRows.add(row);
                
                // Phase 2: Remove after animation completes
                setTimeout(() => {
                    if (row.element) {
                        row.element.remove();
                    }
                    this.zombieRows.delete(row);
                    this.rowsById.delete(row.id);
                }, 400);
            }
        } else {
            // Immediate removal
            if (row.element) {
                row.element.remove();
            }
            this.rowsById.delete(row.id);
        }
    }
}
```

## Phase 5: Integration

### Step 5.1: Create Virtual Grid Component
```typescript
// VirtualGrid.tsx
import React, { useRef, useEffect, useState, useCallback } from 'react';

export interface VirtualGridProps<T> {
    items: T[];
    itemHeight: number;
    renderItem: (item: T, index: number) => React.ReactNode;
    rowBuffer?: number;
    onScroll?: (scrollTop: number) => void;
}

export const VirtualGrid = React.forwardRef<
    HTMLDivElement,
    VirtualGridProps<any>
>(
    (
        {
            items,
            itemHeight,
            renderItem,
            rowBuffer = 10,
            onScroll,
        },
        ref
    ) => {
        const containerRef = useRef<HTMLDivElement>(null);
        const [scrollTop, setScrollTop] = useState(0);
        const [visibleRange, setVisibleRange] = useState({ start: 0, end: 0 });
        
        const animFrameSvc = useRef(new AnimationFrameService()).current;
        
        const handleScroll = useCallback(
            debounce((e: React.UIEvent<HTMLDivElement>) => {
                const target = e.currentTarget;
                const newScrollTop = target.scrollTop;
                
                setScrollTop(newScrollTop);
                animFrameSvc.setScrollTop(newScrollTop);
                
                const viewportHeight = target.clientHeight;
                const bufferPixels = rowBuffer * itemHeight;
                
                const start = Math.max(
                    0,
                    Math.floor((newScrollTop - bufferPixels) / itemHeight)
                );
                
                const end = Math.min(
                    items.length,
                    Math.ceil((newScrollTop + viewportHeight + bufferPixels) / itemHeight)
                );
                
                setVisibleRange({ start, end });
                onScroll?.(newScrollTop);
            }, 100),
            [items.length, itemHeight, rowBuffer, onScroll]
        );
        
        const totalHeight = items.length * itemHeight;
        const visibleItems = items.slice(visibleRange.start, visibleRange.end);
        const offsetY = visibleRange.start * itemHeight;
        
        return (
            <div
                ref={ref || containerRef}
                style={{
                    height: '100%',
                    overflow: 'auto',
                    position: 'relative',
                }}
                onScroll={handleScroll}
            >
                <div style={{ height: totalHeight, position: 'relative' }}>
                    <div style={{ transform: `translateY(${offsetY}px)` }}>
                        {visibleItems.map((item, idx) => (
                            <div
                                key={visibleRange.start + idx}
                                style={{ height: itemHeight, overflow: 'hidden' }}
                            >
                                {renderItem(item, visibleRange.start + idx)}
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        );
    }
);

VirtualGrid.displayName = 'VirtualGrid';
```

### Step 5.2: Usage Example
```typescript
// App.tsx
const [data, setData] = useState<any[]>([]);

const renderRow = (item: any, index: number) => (
    <div className="table-row">
        <div className="table-cell">{item.id}</div>
        <div className="table-cell">{item.name}</div>
        <div className="table-cell">{item.value}</div>
    </div>
);

return (
    <VirtualGrid
        items={data}
        itemHeight={35}
        renderItem={renderRow}
        rowBuffer={10}
        onScroll={(scrollTop) => console.log('Scrolled to:', scrollTop)}
    />
);
```

## Phase 6: Optimization Tuning

### Configuration Recommendations

For different use cases:

**Heavy Data (10k+ rows):**
```typescript
rowBuffer: 15,
rowHeight: 32,
maxRenderedRows: 1000,
debounceMs: 100,
```

**Lightweight (< 1k rows):**
```typescript
rowBuffer: 5,
rowHeight: 28,
maxRenderedRows: 300,
debounceMs: 50,
```

**Dynamic Heights (Variable row size):**
```typescript
rowBuffer: 20,  // More buffer for safety
estimatedRowHeight: 40,
enableHeightMeasurement: true,
```

## Testing Checklist

- [ ] Monitor scroll events per second (should decrease with debounce)
- [ ] Verify DOM node count stays constant (recycling working)
- [ ] Check frame time stays under 16.67ms (60fps)
- [ ] Test fast scrolling - no white chunks
- [ ] Test slow scrolling - smooth animations
- [ ] Verify row recycling by ID (not index)
- [ ] Check memory usage over time (should be stable)
- [ ] Test with large datasets (10k+ rows)
- [ ] Test with dynamic heights
- [ ] Verify scroll end detection (150ms timeout)

## Performance Metrics to Track

```typescript
// PerformanceMonitor.ts
export class PerformanceMonitor {
    private scrollEvents = 0;
    private rowsCreated = 0;
    private startTime = Date.now();
    
    recordScroll(): void {
        this.scrollEvents++;
    }
    
    recordRowCreation(): void {
        this.rowsCreated++;
    }
    
    getMetrics() {
        const elapsed = Date.now() - this.startTime;
        return {
            scrollEventsPerSecond: (this.scrollEvents / elapsed) * 1000,
            rowsCreatedPerSecond: (this.rowsCreated / elapsed) * 1000,
            totalScrollEvents: this.scrollEvents,
            totalRowsCreated: this.rowsCreated,
        };
    }
}
```

## Common Issues & Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| White chunks during scroll | Buffer too small | Increase rowBuffer |
| Flickering on row removal | No animation phase | Add two-phase destruction |
| Lag when scrolling | Large frame budget exceeded | Reduce frame budget or rows |
| High memory usage | Rows not being destroyed | Verify recycling & cleanup |
| Scroll jumping | Multiple scroll sources | Implement source tracking |

## Next Steps

1. Start with Phase 1 (Debouncing)
2. Add Phase 2 (Buffering)
3. Implement Phase 3 (Frame Scheduling) - most critical for performance
4. Add Phase 4 (Row Recycling)
5. Test thoroughly before Phase 5 integration
6. Monitor and tune in Phase 6

