# ag-Grid Performance Analysis - Complete Documentation

This directory contains a comprehensive analysis of ag-Grid's performance patterns for preventing white chunks and flashing during fast scrolling.

## Quick Start

Start here if you're short on time:
- **5 minutes:** Read `AG_GRID_QUICK_REFERENCE.md` - Key concepts and checklist
- **30 minutes:** Read `AG_GRID_FILE_REFERENCE.md` - File mapping and specific implementations
- **2 hours:** Follow `IMPLEMENTATION_GUIDE.md` - Step-by-step code implementation

## Complete Analysis

For a deep dive into the architecture:
- **Read:** `AG_GRID_PERFORMANCE_ANALYSIS.md` (24 KB, detailed with code examples)

This covers:
1. Scroll handling & debouncing (100ms strategy)
2. Buffer strategy (10 rows default overscan)
3. Frame scheduling (requestAnimationFrame with priorities)
4. DOM reuse & row recycling (ID-based reuse pattern)
5. Row height estimation & lazy measurement (caching + lazy eval)
6. Additional optimizations
7. Configuration options
8. Implementation recommendations

## Document Overview

### AG_GRID_QUICK_REFERENCE.md (5.1 KB)
Essential knowledge in minimal format:
- 6-layer solution overview
- Implementation checklist
- Key numbers (100ms, 150ms, 10 rows, 500 max, 60ms budget, 400ms animation)
- 5 critical code patterns
- Common pitfalls
- Debugging tips

**Best for:** Quick lookup, memory jogger, implementation checklist

### AG_GRID_PERFORMANCE_ANALYSIS.md (24 KB)
Complete technical analysis with full code examples:
- 7 major sections with detailed code listings
- Line-by-line references to ag-Grid source
- Explanation of every pattern and why it works
- File paths and method signatures
- Data structure layouts
- Configuration recommendations

**Best for:** Understanding the complete architecture, learning the reasoning behind each pattern

### AG_GRID_FILE_REFERENCE.md (8 KB)
Mapping of ag-Grid source files to patterns:
- Core scroll handling (GridBodyScrollFeature)
- Row rendering & virtualization (RowRenderer)
- Animation frame service (AnimationFrameService)
- Row container control (RowContainerCtrl)
- Utility functions (Debounce, Throttle)
- Viewport management (ViewportSizeFeature)
- Row control lifecycle (RowCtrl)
- Grid options & defaults
- File dependency flow diagram
- Key data structures
- Testing hooks & metrics
- Integration checklist

**Best for:** Finding specific ag-Grid source code, understanding dependencies, integration planning

### IMPLEMENTATION_GUIDE.md (16 KB)
Step-by-step implementation in TypeScript/React:
- 6 phases with code examples
- Phase 1: Debounce & source tracking
- Phase 2: Buffering & viewport calculation
- Phase 3: Frame scheduling (AnimationFrameService)
- Phase 4: Row recycling (RowController)
- Phase 5: Integration (VirtualGrid React component)
- Phase 6: Optimization tuning
- Configuration recommendations by use case
- Testing checklist
- Performance metrics tracking
- Common issues & solutions

**Best for:** Actually implementing the patterns in your codebase

## Key Concepts

### The Problem: White Chunks/Flashing During Fast Scrolling

When users scroll quickly in a table/grid:
- New rows appear blank (white chunks)
- Content flashes in and out
- Grid feels stuttery and unresponsive

### The Solution: Multi-Layer Prevention

ag-Grid prevents this with 6 complementary strategies:

1. **Debouncing (100ms)** - Reduce render cycles during rapid scrolling
2. **Buffering (10 rows)** - Pre-render content before user scrolls to it
3. **Frame Scheduling** - Spread work across frames, render visible content first
4. **DOM Recycling** - Reuse row DOM instead of creating/destroying
5. **Lazy Measurement** - Only measure visible row heights, cache dimensions
6. **Scroll Tracking** - Prevent multiple scroll sources from fighting

### The Key Insight

**Prevention through prediction** - Render what's likely needed before the user sees it, while managing DOM operations efficiently.

## Critical Numbers

| Item | Value | Why |
|------|-------|-----|
| Scroll debounce | 100ms | Reduce redundant renders during rapid scrolling |
| Scroll end timeout | 150ms | Time to consider scroll complete |
| Row buffer | 10 rows | Overscan above/below viewport |
| Max rows rendered | 500 | Safety cap to prevent browser crash |
| Frame budget | 60ms | Time available for work per frame |
| Animation duration | 400ms | Row fade-out animation time |
| Resize debounce | RAF | Prevent cascading resize events |

## Implementation Priority

1. **Critical (Must Have):**
   - Scroll debouncing
   - Row buffer calculation
   - Frame scheduling with priorities
   - Row recycling by ID

2. **Important (Should Have):**
   - Scroll end detection
   - Scroll direction tracking
   - Two-phase destruction
   - Viewport dimension caching

3. **Nice to Have (Polish):**
   - Lazy height measurement
   - LRU row cache
   - Performance monitoring
   - Directional rendering optimization

## Performance Impact

Expected improvements when fully implemented:

- **Scroll events reduced:** 70-80% (debouncing)
- **DOM operations reduced:** 85-90% (recycling)
- **Frame time:** Stays under 16.67ms (60fps)
- **Memory usage:** Constant (no unbounded DOM growth)
- **User perception:** Smooth scrolling, no white chunks

## Testing Strategy

1. Monitor scroll events per second (should decrease)
2. Count DOM nodes (should stay constant)
3. Measure frame time (should stay < 16.67ms)
4. Visual test: Fast scroll - no white chunks
5. Visual test: Slow scroll - smooth animations
6. Load test: 10k+ rows - no lag
7. Memory test: Long session - stable usage

## Related Resources

- **ag-Grid GitHub:** https://github.com/ag-grid/ag-grid
- **Key Files:**
  - gridBodyScrollFeature.ts - Scroll handling
  - rowRenderer.ts - Virtual scrolling core
  - animationFrameService.ts - Frame scheduling
  - agStack/utils/function.ts - Debounce implementation

## Next Steps

1. **Understand:** Read AG_GRID_QUICK_REFERENCE.md (5 min)
2. **Deep Dive:** Read AG_GRID_PERFORMANCE_ANALYSIS.md (1 hour)
3. **Find Code:** Use AG_GRID_FILE_REFERENCE.md (as needed)
4. **Implement:** Follow IMPLEMENTATION_GUIDE.md (2-3 hours)
5. **Test:** Use testing checklist and metrics
6. **Optimize:** Tune based on your specific use case

## Support

If you have questions:
1. Check AG_GRID_QUICK_REFERENCE.md Common Pitfalls section
2. Search AG_GRID_FILE_REFERENCE.md for specific components
3. Review IMPLEMENTATION_GUIDE.md Common Issues & Solutions
4. Cross-reference AG_GRID_PERFORMANCE_ANALYSIS.md for detailed explanations

---

**Created:** Analysis of ag-Grid v34+ architecture
**Scope:** Preventing white chunks/flashing during fast scrolling
**Coverage:** Scroll handling, buffering, frame scheduling, DOM recycling, height measurement
**Code Examples:** TypeScript/React implementation patterns

