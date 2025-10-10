import { useCallback, useRef, useEffect, useMemo } from 'react';
import { requestAnimationFrame, cancelAnimationFrame } from '../utils/table';

interface UsePerformanceOptions {
  enableThrottling?: boolean;
  throttleMs?: number;
  enableDebouncing?: boolean;
  debounceMs?: number;
  enableRAF?: boolean;
}

export const usePerformance = (options: UsePerformanceOptions = {}) => {
  const {
    enableThrottling = true,
    throttleMs = 16, // ~60fps
    enableDebouncing = true,
    debounceMs = 300,
    enableRAF = true,
  } = options;

  // Performance metrics
  const metricsRef = useRef({
    renderCount: 0,
    lastRenderTime: 0,
    averageRenderTime: 0,
    peakRenderTime: 0,
    totalRenderTime: 0,
  });

  // RAF queue for smooth animations
  const rafQueueRef = useRef<Array<() => void>>([]);
  const rafIdRef = useRef<number>();

  const processRAFQueue = useCallback(() => {
    const startTime = performance.now();
    const queue = rafQueueRef.current.splice(0);

    queue.forEach(callback => {
      try {
        callback();
      } catch (error) {
        console.error('RAF callback error:', error);
      }
    });

    const endTime = performance.now();
    const renderTime = endTime - startTime;

    // Update metrics
    const metrics = metricsRef.current;
    metrics.renderCount++;
    metrics.lastRenderTime = renderTime;
    metrics.totalRenderTime += renderTime;
    metrics.averageRenderTime = metrics.totalRenderTime / metrics.renderCount;
    metrics.peakRenderTime = Math.max(metrics.peakRenderTime, renderTime);

    // Schedule next frame if there are more items in queue
    if (rafQueueRef.current.length > 0) {
      rafIdRef.current = requestAnimationFrame(processRAFQueue);
    } else {
      rafIdRef.current = undefined;
    }
  }, []);

  const scheduleRAF = useCallback((callback: () => void) => {
    if (!enableRAF) {
      callback();
      return;
    }

    rafQueueRef.current.push(callback);

    if (!rafIdRef.current) {
      rafIdRef.current = requestAnimationFrame(processRAFQueue);
    }
  }, [enableRAF, processRAFQueue]);

  const throttle = useCallback(<T extends (...args: unknown[]) => unknown>(
    func: T,
    limit: number = throttleMs
  ): (...args: Parameters<T>) => void => {
    if (!enableThrottling) {
      return func;
    }

    let inThrottle: boolean;

    return (...args: Parameters<T>) => {
      if (!inThrottle) {
        scheduleRAF(() => func(...args));
        inThrottle = true;
        setTimeout(() => (inThrottle = false), limit);
      }
    };
  }, [enableThrottling, throttleMs, scheduleRAF]);

  const debounce = useCallback(<T extends (...args: unknown[]) => unknown>(
    func: T,
    wait: number = debounceMs
  ): (...args: Parameters<T>) => void => {
    if (!enableDebouncing) {
      return func;
    }

    let timeout: NodeJS.Timeout;

    return (...args: Parameters<T>) => {
      clearTimeout(timeout);
      timeout = setTimeout(() => {
        scheduleRAF(() => func(...args));
      }, wait);
    };
  }, [enableDebouncing, debounceMs, scheduleRAF]);

  // Memoized stable references
  const stableThrottle = useMemo(() => throttle, [throttle]);
  const stableDebounce = useMemo(() => debounce, [debounce]);

  // Memory management
  useEffect(() => {
    return () => {
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }
      rafQueueRef.current = [];
    };
  }, []);

  // Performance monitoring
  const getMetrics = useCallback(() => ({ ...metricsRef.current }), []);

  const resetMetrics = useCallback(() => {
    metricsRef.current = {
      renderCount: 0,
      lastRenderTime: 0,
      averageRenderTime: 0,
      peakRenderTime: 0,
      totalRenderTime: 0,
    };
  }, []);

  return {
    throttle: stableThrottle,
    debounce: stableDebounce,
    scheduleRAF,
    getMetrics,
    resetMetrics,
  };
};

// Hook for measuring component render performance
export const useRenderPerformance = (componentName: string, enabled = false) => {
  const renderStartRef = useRef<number>();
  const renderCountRef = useRef(0);

  useEffect(() => {
    if (!enabled) return;

    renderStartRef.current = performance.now();
    renderCountRef.current++;

    return () => {
      if (renderStartRef.current) {
        const renderTime = performance.now() - renderStartRef.current;
        console.log(`${componentName} render #${renderCountRef.current}: ${renderTime.toFixed(2)}ms`);
      }
    };
  });

  return {
    renderCount: renderCountRef.current,
  };
};

// Hook for virtual scrolling optimization
export const useVirtualScrollingOptimization = (
  containerRef: React.RefObject<HTMLElement>
) => {
  const scrollPositionRef = useRef(0);
  const visibleRangeRef = useRef({ start: 0, end: 0 });


  const updateVisibleRange = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;

    const { scrollTop } = container;
    scrollPositionRef.current = scrollTop;

    // Calculate new visible range (this would be used by the virtual list)
    // In practice, this would be integrated with the virtualizer
  }, [containerRef]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const handleScroll = () => {
      requestAnimationFrame(updateVisibleRange);
    };

    container.addEventListener('scroll', handleScroll, { passive: true });

    return () => {
      container.removeEventListener('scroll', handleScroll);
    };
  }, [containerRef, updateVisibleRange]);

  return {
    visibleRange: visibleRangeRef.current,
    scrollPosition: scrollPositionRef.current,
  };
};