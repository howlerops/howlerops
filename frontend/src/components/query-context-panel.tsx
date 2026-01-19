/**
 * Query Context Panel Component
 *
 * Displays relevant context for queries including:
 * - Relevant tables and schemas
 * - Similar past queries
 * - Performance hints
 */

import { AlertTriangle, Clock, Database, Info,Table2, TrendingUp } from 'lucide-react';
import React, { useCallback,useEffect, useState } from 'react';

import { api } from '@/lib/api-client';

interface QueryContext {
  relevantTables: TableContext[];
  similarQueries: SimilarQuery[];
  performanceHints: PerformanceHint[];
}

interface TableContext {
  name: string;
  schema: string;
  relevance: number;
  rowCount?: number;
  columns?: string[];
}

interface SimilarQuery {
  query: string;
  similarity: number;
  avgDuration?: number;
  successRate: number;
}

interface PerformanceHint {
  type: 'warning' | 'info' | 'suggestion';
  message: string;
  impact?: string;
}

interface QueryContextPanelProps {
  connectionId?: string;
  query?: string;
}

/**
 * Extract table names referenced in a SQL query.
 * Handles common patterns: FROM, JOIN, INTO, UPDATE, TABLE
 */
function extractTableReferences(sql: string): string[] {
  const normalized = sql.toLowerCase();
  const tables = new Set<string>();

  // Pattern matches: FROM table, JOIN table, INTO table, UPDATE table, TABLE table
  const patterns = [
    /\bfrom\s+([a-z_][a-z0-9_]*(?:\.[a-z_][a-z0-9_]*)?)/gi,
    /\bjoin\s+([a-z_][a-z0-9_]*(?:\.[a-z_][a-z0-9_]*)?)/gi,
    /\binto\s+([a-z_][a-z0-9_]*(?:\.[a-z_][a-z0-9_]*)?)/gi,
    /\bupdate\s+([a-z_][a-z0-9_]*(?:\.[a-z_][a-z0-9_]*)?)/gi,
    /\btable\s+([a-z_][a-z0-9_]*(?:\.[a-z_][a-z0-9_]*)?)/gi,
  ];

  for (const pattern of patterns) {
    let match;
    while ((match = pattern.exec(normalized)) !== null) {
      // Extract just the table name (last part if schema.table)
      const fullRef = match[1];
      const tableName = fullRef.includes('.') ? fullRef.split('.').pop()! : fullRef;
      tables.add(tableName);
    }
  }

  return Array.from(tables);
}

/**
 * Generate performance hints based on query analysis.
 */
function analyzeQueryPerformance(sql: string): PerformanceHint[] {
  const hints: PerformanceHint[] = [];
  const normalized = sql.toLowerCase();

  // SELECT * warning
  if (/select\s+\*/.test(normalized)) {
    hints.push({
      type: 'suggestion',
      message: 'Consider selecting specific columns instead of SELECT * for better performance',
      impact: 'Medium'
    });
  }

  // Missing WHERE clause on large operations
  if ((normalized.includes('update') || normalized.includes('delete')) && !normalized.includes('where')) {
    hints.push({
      type: 'warning',
      message: 'UPDATE/DELETE without WHERE clause will affect all rows',
      impact: 'High'
    });
  }

  // LIKE with leading wildcard
  if (/like\s+['"]%/.test(normalized)) {
    hints.push({
      type: 'suggestion',
      message: 'LIKE patterns starting with % cannot use indexes efficiently',
      impact: 'Medium'
    });
  }

  // ORDER BY without LIMIT
  if (normalized.includes('order by') && !normalized.includes('limit')) {
    hints.push({
      type: 'info',
      message: 'Consider adding LIMIT when using ORDER BY on large tables'
    });
  }

  // Subquery in WHERE
  if (/where\s+.*\(\s*select/i.test(normalized)) {
    hints.push({
      type: 'suggestion',
      message: 'Subqueries in WHERE may be rewritten as JOINs for better performance',
      impact: 'Medium'
    });
  }

  // DISTINCT can be expensive
  if (normalized.includes('distinct')) {
    hints.push({
      type: 'info',
      message: 'DISTINCT requires sorting/hashing all results - ensure it\'s necessary'
    });
  }

  return hints;
}

export const QueryContextPanel: React.FC<QueryContextPanelProps> = ({ connectionId, query }) => {
  const [context, setContext] = useState<QueryContext | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const loadContext = useCallback(async () => {
    if (!connectionId || !query) {
      setContext(null);
      return;
    }

    setIsLoading(true);
    try {
      // Extract table references from the query
      const referencedTables = extractTableReferences(query);

      // Fetch actual table data from backend
      const tablesResult = await api.schema.tables(connectionId);

      const relevantTables: TableContext[] = [];

      if (tablesResult.success && tablesResult.data) {
        // Match referenced tables with actual tables
        for (const table of tablesResult.data) {
          const tableLower = table.name.toLowerCase();
          const isDirectMatch = referencedTables.some(ref => ref === tableLower);

          if (isDirectMatch) {
            // Fetch column info for directly referenced tables
            let columns: string[] = [];
            try {
              const structureResult = await api.schema.columns(
                connectionId,
                table.schema || 'public',
                table.name
              );
              if (structureResult.success && structureResult.data) {
                columns = structureResult.data.map(col => col.name);
              }
            } catch {
              // Column fetch failed, continue without columns
            }

            relevantTables.push({
              name: table.name,
              schema: table.schema || 'public',
              relevance: 1.0,
              rowCount: table.rowCount,
              columns
            });
          }
        }

        // Sort by relevance
        relevantTables.sort((a, b) => b.relevance - a.relevance);
      }

      // Generate performance hints from query analysis
      const performanceHints = analyzeQueryPerformance(query);

      // Similar queries: For now, return empty (would need query history storage)
      // This could be enhanced with local storage of past queries
      const similarQueries: SimilarQuery[] = [];

      setContext({
        relevantTables,
        similarQueries,
        performanceHints
      });
    } catch (err) {
      console.error('Failed to load context:', err);
      // On error, still show performance hints from local analysis
      setContext({
        relevantTables: [],
        similarQueries: [],
        performanceHints: analyzeQueryPerformance(query || '')
      });
    } finally {
      setIsLoading(false);
    }
  }, [connectionId, query]);

  // Load context when query or connection changes
  useEffect(() => {
    if (query && query.length > 10 && connectionId) {
       
      loadContext();
    } else {
      setContext(null);
    }
  }, [query, connectionId, loadContext]);

  // Render loading state
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-gray-400">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  // Render empty state
  if (!context) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-gray-500 p-8 text-center">
        <Info size={48} className="mb-4 opacity-50" />
        <p className="text-sm">Start typing a query to see relevant context</p>
      </div>
    );
  }

  // Render relevance indicator
  const renderRelevance = (relevance: number) => {
    const color = relevance >= 0.8 ? 'bg-primary/10' : relevance >= 0.6 ? 'bg-accent/10' : 'bg-gray-500';
    return (
      <div className="flex items-center gap-2">
        <div className="w-16 h-2 bg-gray-700 rounded-full overflow-hidden">
          <div className={`h-full ${color}`} style={{ width: `${relevance * 100}%` }} />
        </div>
        <span className="text-xs text-gray-400">{Math.round(relevance * 100)}%</span>
      </div>
    );
  };

  return (
    <div className="flex flex-col h-full overflow-auto bg-gray-900 border-l border-gray-700">
      <div className="p-4 border-b border-gray-700">
        <h3 className="text-lg font-semibold flex items-center gap-2">
          <Database size={20} />
          Query Context
        </h3>
      </div>

      <div className="flex-1 overflow-auto p-4 space-y-6">
        {/* Relevant Tables */}
        {context.relevantTables.length > 0 && (
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-sm font-semibold text-gray-300">
              <Table2 size={16} />
              <span>Relevant Tables</span>
            </div>
            <div className="space-y-2">
              {context.relevantTables.map((table, idx) => (
                <div key={idx} className="p-3 bg-gray-800 rounded-lg border border-gray-700">
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-mono text-sm text-primary">
                      {table.schema}.{table.name}
                    </span>
                    {renderRelevance(table.relevance)}
                  </div>
                  {table.rowCount != null && table.rowCount > 0 && (
                    <div className="text-xs text-gray-500 mb-2">{table.rowCount.toLocaleString()} rows</div>
                  )}
                  {table.columns && table.columns.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {table.columns.map((col, colIdx) => (
                        <span key={colIdx} className="px-2 py-1 bg-gray-700 rounded text-xs text-gray-400">
                          {col}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Similar Queries - only shown when we have history */}
        {context.similarQueries.length > 0 && (
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-sm font-semibold text-gray-300">
              <Clock size={16} />
              <span>Similar Past Queries</span>
            </div>
            <div className="space-y-2">
              {context.similarQueries.map((similar, idx) => (
                <div key={idx} className="p-3 bg-gray-800 rounded-lg border border-gray-700">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs text-gray-500">Similarity</span>
                    {renderRelevance(similar.similarity)}
                  </div>
                  <code className="text-xs text-gray-400 font-mono block mb-2 break-all">{similar.query}</code>
                  <div className="flex items-center gap-3 text-xs text-gray-500">
                    {similar.avgDuration && (
                      <div className="flex items-center gap-1">
                        <Clock size={12} />
                        <span>{similar.avgDuration}ms avg</span>
                      </div>
                    )}
                    <div className="flex items-center gap-1">
                      <TrendingUp size={12} />
                      <span>{Math.round(similar.successRate * 100)}% success</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Performance Hints */}
        {context.performanceHints.length > 0 && (
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-sm font-semibold text-gray-300">
              <TrendingUp size={16} />
              <span>Performance Hints</span>
            </div>
            <div className="space-y-2">
              {context.performanceHints.map((hint, idx) => (
                <div
                  key={idx}
                  className={`p-3 rounded-lg border ${
                    hint.type === 'warning'
                      ? 'bg-accent/10 border-accent/50'
                      : hint.type === 'suggestion'
                      ? 'bg-primary/10 border-primary/50'
                      : 'bg-gray-800 border-gray-700'
                  }`}
                >
                  <div className="flex items-start gap-2">
                    {hint.type === 'warning' && <AlertTriangle size={16} className="text-accent-foreground mt-0.5" />}
                    {hint.type === 'suggestion' && <TrendingUp size={16} className="text-primary mt-0.5" />}
                    {hint.type === 'info' && <Info size={16} className="text-gray-400 mt-0.5" />}
                    <div className="flex-1">
                      <p className="text-sm text-gray-300">{hint.message}</p>
                      {hint.impact && (
                        <span className="text-xs text-gray-500 mt-1 inline-block">Impact: {hint.impact}</span>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Empty state when no context available */}
        {context.relevantTables.length === 0 && context.performanceHints.length === 0 && (
          <div className="flex flex-col items-center justify-center py-8 text-gray-500 text-center">
            <Info size={32} className="mb-3 opacity-50" />
            <p className="text-sm">No tables detected in query</p>
            <p className="text-xs mt-1 opacity-75">Try adding FROM, JOIN, or UPDATE clauses</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default QueryContextPanel;

