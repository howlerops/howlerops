/**
 * Hook to automatically detect and manage query editor mode (single vs multi-DB)
 * Auto-switches based on number of connections
 */

import { useState, useEffect } from 'react';
import { GetConnectionCount } from '../../wailsjs/go/main/App';

export type QueryMode = 'single' | 'multi';

export interface UseQueryModeReturn {
  mode: QueryMode;
  canToggle: boolean;
  toggleMode: () => void;
  connectionCount: number;
  isMultiDB: boolean;
}

export function useQueryMode(initialMode?: 'auto' | QueryMode): UseQueryModeReturn {
  const [mode, setMode] = useState<QueryMode>('single');
  const [connectionCount, setConnectionCount] = useState(0);
  const [canToggle, setCanToggle] = useState(false);

  // Fetch connection count and determine mode
  useEffect(() => {
    const fetchConnectionCount = async () => {
      try {
        const count = await GetConnectionCount();
        setConnectionCount(count);

        // Auto-detect mode based on connection count
        if (initialMode === 'auto' || !initialMode) {
          const newMode: QueryMode = count > 1 ? 'multi' : 'single';
          setMode(newMode);
        } else {
          setMode(initialMode);
        }

        // Can only toggle if we have 2+ connections
        setCanToggle(count > 1);
      } catch (error) {
        console.error('Failed to get connection count:', error);
        setConnectionCount(0);
        setMode('single');
        setCanToggle(false);
      }
    };

    fetchConnectionCount();

    // Poll for connection changes every 5 seconds
    const interval = setInterval(fetchConnectionCount, 5000);

    return () => clearInterval(interval);
  }, [initialMode]);

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
  const [enabled, setEnabled] = useState(false);

  useEffect(() => {
    const checkMultiDB = async () => {
      try {
        const count = await GetConnectionCount();
        setEnabled(count > 1);
      } catch (error) {
        setEnabled(false);
      }
    };

    checkMultiDB();
    const interval = setInterval(checkMultiDB, 5000);

    return () => clearInterval(interval);
  }, []);

  return enabled;
}

