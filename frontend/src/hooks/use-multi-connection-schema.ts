/**
 * Hook for managing multi-connection schema data
 * 
 * Provides cached schema information for multiple database connections
 * and detects schema conflicts
 */

import { useState, useEffect, useCallback } from 'react';
import { GetMultiConnectionSchema, ListConnections } from '../../wailsjs/go/main/App';

export interface Connection {
  id: string;
  type: string;
  database: string;
  name?: string;
  alias?: string;
  active: boolean;
}

export interface TableInfo {
  schema: string;
  name: string;
  type: string;
  comment: string;
  rowCount: number;
  sizeBytes: number;
}

export interface ConnectionSchema {
  connectionId: string;
  name: string;
  type: string;
  schemas: string[];
  tables: TableInfo[];
}

export interface SchemaConflict {
  tableName: string;
  connections: {
    connectionId: string;
    tableName: string;
    schema: string;
  }[];
  resolution: string;
}

export interface CombinedSchema {
  connections: Record<string, ConnectionSchema>;
  conflicts: SchemaConflict[];
}

interface UseMultiConnectionSchemaReturn {
  connections: Connection[];
  schemas: CombinedSchema | null;
  isLoading: boolean;
  error: string | null;
  refreshConnections: () => Promise<void>;
  loadSchemas: (connectionIds: string[]) => Promise<void>;
  getTablesForConnection: (connectionId: string) => TableInfo[];
  getSchemaConflicts: () => SchemaConflict[];
}

/**
 * Hook for managing multi-connection schema data
 */
export function useMultiConnectionSchema(): UseMultiConnectionSchemaReturn {
  const [connections, setConnections] = useState<Connection[]>([]);
  const [schemas, setSchemas] = useState<CombinedSchema | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load connections on mount
  useEffect(() => {
    refreshConnections();
  }, []);

  /**
   * Refresh the list of available connections
   */
  const refreshConnections = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const conns = await ListConnections();
      setConnections(conns || []);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : String(err);
      setError(`Failed to load connections: ${errorMessage}`);
      console.error('Failed to load connections:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Load schemas for specified connections
   */
  const loadSchemas = useCallback(async (connectionIds: string[]) => {
    if (connectionIds.length === 0) {
      setSchemas(null);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const combinedSchema = await GetMultiConnectionSchema(connectionIds);
      setSchemas(combinedSchema);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : String(err);
      setError(`Failed to load schemas: ${errorMessage}`);
      console.error('Failed to load schemas:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Get all tables for a specific connection
   */
  const getTablesForConnection = useCallback(
    (connectionId: string): TableInfo[] => {
      if (!schemas || !schemas.connections[connectionId]) {
        return [];
      }

      return schemas.connections[connectionId].tables || [];
    },
    [schemas]
  );

  /**
   * Get all schema conflicts
   */
  const getSchemaConflicts = useCallback((): SchemaConflict[] => {
    if (!schemas) {
      return [];
    }

    return schemas.conflicts || [];
  }, [schemas]);

  return {
    connections,
    schemas,
    isLoading,
    error,
    refreshConnections,
    loadSchemas,
    getTablesForConnection,
    getSchemaConflicts,
  };
}

export default useMultiConnectionSchema;

