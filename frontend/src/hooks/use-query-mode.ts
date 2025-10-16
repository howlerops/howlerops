/**
 * Hook to automatically detect and manage query editor mode (single vs multi-DB)
 * Auto-switches based on number of connections
 */

import { useState, useEffect } from 'react';
import { useConnectionStore } from '@/store/connection-store';

export type QueryMode = 'single' | 'multi';

export interface UseQueryModeReturn {
  mode: QueryMode;
  canToggle: boolean;
  toggleMode: () => void;
  connectionCount: number;
  isMultiDB: boolean;
}

export function useQueryMode(initialMode?: 'auto' | QueryMode): UseQueryModeReturn {
  const { connections } = useConnectionStore();
  const connectionCount = connections.length;
  const canToggle = connectionCount > 1;

  // Calculate initial mode based on connection count
  const getInitialMode = (): QueryMode => {
    if (initialMode === 'auto' || !initialMode) {
      return connectionCount > 1 ? 'multi' : 'single';
    }
    return initialMode as QueryMode;
  };

  const [mode, setMode] = useState<QueryMode>(getInitialMode);

  // Auto-adjust mode when connection count changes
  useEffect(() => {
    if (initialMode === 'auto' || !initialMode) {
      const newMode: QueryMode = connectionCount > 1 ? 'multi' : 'single';
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setMode(newMode);
    }
  }, [connectionCount, initialMode]);

  const toggleMode = () => {
    if (canToggle) {
      setMode((prev) => (prev === 'single' ? 'multi' : 'single'));
    }
  };

  return {
    mode,
    canToggle,
    toggleMode,
    connectionCount,
    isMultiDB: mode === 'multi',
  };
}

// Hook to check if multi-DB features should be enabled
export function useMultiDBEnabled(): boolean {
  const { connections } = useConnectionStore();
  return connections.length > 1;
}

