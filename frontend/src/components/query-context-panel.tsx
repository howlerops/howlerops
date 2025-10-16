/**
 * Query Context Panel Component
 * 
 * Displays relevant context for queries including:
 * - Relevant tables and schemas
 * - Similar past queries
 * - Performance hints
 */

import React, { useState, useEffect, useCallback } from 'react';
import { Database, Clock, TrendingUp, AlertTriangle, Table2, Info } from 'lucide-react';

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

export const QueryContextPanel: React.FC<QueryContextPanelProps> = ({ connectionId, query }) => {
  const [context, setContext] = useState<QueryContext | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const loadContext = useCallback(async () => {
    setIsLoading(true);
    try {
      // TODO: Call backend API to get query context
      // For now, use mock data
      setTimeout(() => {
        setContext({
          relevantTables: [
            {
              name: 'users',
              schema: 'public',
              relevance: 0.95,
              rowCount: 15420,
              columns: ['id', 'name', 'email', 'created_at'],
            },
            {
              name: 'orders',
              schema: 'public',
              relevance: 0.82,
              rowCount: 45231,
              columns: ['id', 'user_id', 'total', 'status', 'created_at'],
            },
          ],
          similarQueries: [
            {
              query: 'SELECT u.name, COUNT(o.id) FROM users u JOIN orders o ON u.id = o.user_id GROUP BY u.name',
              similarity: 0.89,
              avgDuration: 125,
              successRate: 1.0,
            },
            {
              query: 'SELECT * FROM users WHERE created_at > NOW() - INTERVAL \'30 days\'',
              similarity: 0.75,
              avgDuration: 45,
              successRate: 1.0,
            },
          ],
          performanceHints: [
            {
              type: 'suggestion',
              message: 'Consider adding an index on users.created_at for faster filtering',
              impact: 'High',
            },
            {
              type: 'info',
              message: 'This query typically returns ~500 rows and takes ~100ms',
            },
          ],
        });
        setIsLoading(false);
      }, 500);
    } catch (err) {
      console.error('Failed to load context:', err);
      setIsLoading(false);
    }
  }, []);

  // Load context when query or connection changes
  useEffect(() => {
    if (query && query.length > 10 && connectionId) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
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
                {table.rowCount && (
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

        {/* Similar Queries */}
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

        {/* Performance Hints */}
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
                    ? 'bg-accent/10/10 border-accent/50'
                    : hint.type === 'suggestion'
                    ? 'bg-primary/10/10 border-primary/50'
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
      </div>
    </div>
  );
};

export default QueryContextPanel;

