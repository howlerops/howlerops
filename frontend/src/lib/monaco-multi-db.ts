/**
 * Monaco Editor Multi-Database Language Support
 *
 * Provides syntax highlighting, completion, and validation for multi-database queries
 * with @connection_alias.schema.table syntax
 */

import * as monaco from 'monaco-editor';

export interface Connection {
  id: string;
  alias?: string;
  name: string;
  type: string;
  database: string;
  sessionId?: string;
  isConnected?: boolean;
}

export interface SchemaNode {
  id?: string;  // Optional for compatibility
  name: string;
  type: 'database' | 'schema' | 'table' | 'column';  // Include 'database' for compatibility
  children?: SchemaNode[];
  dataType?: string;
  nullable?: boolean;
  primaryKey?: boolean;
  metadata?: any;
  sessionId?: string;
  expanded?: boolean;  // Optional for compatibility
}

export interface Column {
  name: string;
  dataType: string;
  nullable?: boolean;
  primaryKey?: boolean;
}

export type ColumnLoader = (sessionId: string, schema: string, table: string) => Promise<Column[]>;

/**
 * Multi-DB Monaco Provider Manager
 * Manages the lifecycle of Monaco language providers for multi-database support
 */
export class MultiDBMonacoProvider {
  private editor: monaco.editor.IStandaloneCodeEditor;
  private connections: Connection[];
  private schemas: Map<string, SchemaNode[]>;
  private columnLoader?: ColumnLoader;
  private columnCache: Map<string, Column[]> = new Map();
  private disposables: monaco.IDisposable[] = [];
  private mode: 'single' | 'multi' = 'single';

  constructor(
    editor: monaco.editor.IStandaloneCodeEditor,
    connections: Connection[] = [],
    schemas: Map<string, SchemaNode[]> = new Map(),
    columnLoader?: ColumnLoader,
    mode: 'single' | 'multi' = 'single'
  ) {
    this.editor = editor;
    this.connections = connections;
    this.schemas = schemas;
    this.columnLoader = columnLoader;
    this.mode = mode;
    this.initialize();
  }

  /**
   * Update connections and schemas, then refresh providers
   */
  update(connections: Connection[], schemas: Map<string, SchemaNode[]>) {
    this.connections = connections;
    this.schemas = schemas;
    this.refresh();
  }

  /**
   * Set the mode (single or multi database)
   */
  setMode(mode: 'single' | 'multi') {
    if (this.mode !== mode) {
      this.mode = mode;
      this.refresh();
    }
  }

  /**
   * Dispose all providers and clean up
   */
  dispose() {
    this.disposables.forEach(d => {
      try {
        d.dispose();
      } catch (e) {
        console.warn('Failed to dispose provider:', e);
      }
    });
    this.disposables = [];
    this.columnCache.clear();
  }

  /**
   * Refresh all providers with current data
   */
  private refresh() {
    // Dispose old providers
    this.disposables.forEach(d => {
      try {
        d.dispose();
      } catch (e) {
        console.warn('Failed to dispose provider:', e);
      }
    });
    this.disposables = [];
    // Re-initialize with new data
    this.registerCompletionProvider();
    if (this.mode === 'multi') {
      this.registerHoverProvider();
    }
  }

  private initialize() {
    // Only configure language once per editor instance
    if (!this.editor.getModel()) {
      return;
    }

    // Register providers
    this.registerCompletionProvider();

    // Only register hover provider in multi-DB mode
    if (this.mode === 'multi') {
      this.registerHoverProvider();
    }
  }

  private async loadColumnsForTable(
    sessionId: string,
    schema: string,
    table: string
  ): Promise<Column[]> {
    const cacheKey = `${sessionId}-${schema}-${table}`;

    // Check cache first
    if (this.columnCache.has(cacheKey)) {
      return this.columnCache.get(cacheKey)!;
    }

    // Load columns if loader is available
    if (this.columnLoader) {
      try {
        const columns = await this.columnLoader(sessionId, schema, table);
        this.columnCache.set(cacheKey, columns);
        return columns;
      } catch (error) {
        console.error(`Failed to load columns for ${schema}.${table}:`, error);
        return [];
      }
    }

    return [];
  }

  private findConnectionByIdentifier(identifier: string): Connection | undefined {
    return this.connections.find(c =>
      c.name === identifier ||
      c.id === identifier ||
      c.alias === identifier
    );
  }

  private isTypingSQLKeyword(word: string): boolean {
    if (!word || word.length < 2) return false;
    const sqlKeywords = ['SELECT', 'FROM', 'WHERE', 'JOIN', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER'];
    const upperWord = word.toUpperCase();
    return sqlKeywords.some(kw => kw.startsWith(upperWord));
  }

  private registerCompletionProvider() {
    const triggerCharacters = this.mode === 'multi' ? ['@', '.'] : ['.'];

    const provider = monaco.languages.registerCompletionItemProvider('sql', {
      triggerCharacters,
      provideCompletionItems: async (model, position, context) => {
        try {
          const word = model.getWordUntilPosition(position);
          const range = {
            startLineNumber: position.lineNumber,
            endLineNumber: position.lineNumber,
            startColumn: word.startColumn,
            endColumn: word.endColumn,
          };

          const textUntilCursor = model.getValueInRange({
            startLineNumber: position.lineNumber,
            startColumn: 1,
            endLineNumber: position.lineNumber,
            endColumn: position.column,
          });

          const trimmedBeforeCursor = textUntilCursor.replace(/\s+$/, '');

          // Check if we're typing a SQL keyword
          const currentWord = word.word.toUpperCase();
          const isTypingKeyword = this.isTypingSQLKeyword(currentWord);

        // Multi-DB mode: Handle @connection patterns
        if (this.mode === 'multi') {
          // Case 1: User typed '@' - show connections
          if (trimmedBeforeCursor.endsWith('@')) {
            return {
              suggestions: this.connections
                .filter(conn => conn.isConnected)
                .map((conn, idx) => ({
                  label: `@${conn.name || conn.id}`,
                  kind: monaco.languages.CompletionItemKind.Module,
                  detail: `${conn.type} - ${conn.database}`,
                  insertText: `@${conn.name || conn.id}`,
                  documentation: `${conn.name} (${conn.type})`,
                  sortText: `0${idx.toString().padStart(3, '0')}`,
                  range: {
                    startLineNumber: position.lineNumber,
                    endLineNumber: position.lineNumber,
                    startColumn: position.column - 1,
                    endColumn: position.column,
                  }
                }))
            };
          }

          // Case 2: User typed '@connection.schema.table.' - show columns
          const columnPattern = /@([\w-]+)\.(\w+)\.(\w+)\.(\w*)$/;
          const columnMatch = trimmedBeforeCursor.match(columnPattern);

          if (columnMatch) {
            const [, connId, schemaName, tableName, partialColumn] = columnMatch;
            const connection = this.findConnectionByIdentifier(connId);

            if (connection && connection.sessionId) {
              const columns = await this.loadColumnsForTable(
                connection.sessionId,
                schemaName,
                tableName
              );

              return {
                suggestions: columns
                  .filter(col => !partialColumn || col.name.toLowerCase().startsWith(partialColumn.toLowerCase()))
                  .map(col => ({
                    label: col.name,
                    kind: monaco.languages.CompletionItemKind.Property,
                    insertText: col.name,
                    detail: `${tableName}.${col.name} (${col.dataType || 'unknown'})`,
                    documentation: col.nullable ? 'Nullable' : 'Not null',
                    sortText: `2_${col.name}`,
                    range: {
                      startLineNumber: position.lineNumber,
                      endLineNumber: position.lineNumber,
                      startColumn: position.column - partialColumn.length,
                      endColumn: position.column,
                    }
                  }))
              };
            }
          }

          // Case 3: User typed '@connection.table.' - show columns (assuming public schema)
          const tableColumnPattern = /@([\w-]+)\.(\w+)\.(\w*)$/;
          const tableColumnMatch = trimmedBeforeCursor.match(tableColumnPattern);

          if (tableColumnMatch) {
            const [, connId, tableOrSchema, partialColumn] = tableColumnMatch;
            const connection = this.findConnectionByIdentifier(connId);

            if (connection && this.schemas.has(connection.id)) {
              const connSchemas = this.schemas.get(connection.id)!;

              // Check if this is actually a table name (not a schema)
              for (const schema of connSchemas) {
                if (schema.type === 'schema' && schema.children) {
                  const table = schema.children.find(t =>
                    t.type === 'table' && t.name === tableOrSchema
                  );

                  if (table && connection.sessionId) {
                    const columns = await this.loadColumnsForTable(
                      connection.sessionId,
                      schema.name,
                      tableOrSchema
                    );

                    return {
                      suggestions: columns
                        .filter(col => !partialColumn || col.name.toLowerCase().startsWith(partialColumn.toLowerCase()))
                        .map(col => ({
                          label: col.name,
                          kind: monaco.languages.CompletionItemKind.Property,
                          insertText: col.name,
                          detail: `${tableOrSchema}.${col.name} (${col.dataType || 'unknown'})`,
                          documentation: col.nullable ? 'Nullable' : 'Not null',
                          sortText: `2_${col.name}`,
                          range: {
                            startLineNumber: position.lineNumber,
                            endLineNumber: position.lineNumber,
                            startColumn: position.column - partialColumn.length,
                            endColumn: position.column,
                          }
                        }))
                    };
                  }
                }
              }

              // If not found as a table, treat as schema and show tables
              const schema = connSchemas.find(s => s.name === tableOrSchema);
              if (schema && schema.children) {
                return {
                  suggestions: schema.children
                    .filter(t => t.type === 'table')
                    .filter(t => !partialColumn || t.name.toLowerCase().startsWith(partialColumn.toLowerCase()))
                    .map(table => ({
                      label: table.name,
                      kind: monaco.languages.CompletionItemKind.Class,
                      detail: `@${connection.name}.${schema.name}.${table.name}`,
                      insertText: table.name,
                      documentation: `Table in ${schema.name}`,
                      sortText: `1_${table.name}`,
                      range: {
                        startLineNumber: position.lineNumber,
                        endLineNumber: position.lineNumber,
                        startColumn: position.column - partialColumn.length,
                        endColumn: position.column,
                      }
                    }))
                };
              }
            }
          }

          // Case 4: User typed '@connection.' - show schemas/tables
          const connPattern = /@([\w-]+)\.(\w*)$/;
          const connMatch = trimmedBeforeCursor.match(connPattern);

          if (connMatch) {
            const [, connId, partialTable] = connMatch;
            const connection = this.findConnectionByIdentifier(connId);

            if (connection && this.schemas.has(connection.id)) {
              const connSchemas = this.schemas.get(connection.id)!;
              const suggestions: any[] = [];

              connSchemas.forEach(schema => {
                if (schema.type === 'schema' && schema.children) {
                  schema.children
                    .filter(t => t.type === 'table')
                    .filter(t => !partialTable || t.name.toLowerCase().startsWith(partialTable.toLowerCase()))
                    .forEach(table => {
                      const fullName = schema.name === 'public'
                        ? table.name
                        : `${schema.name}.${table.name}`;

                      suggestions.push({
                        label: table.name,
                        kind: monaco.languages.CompletionItemKind.Class,
                        detail: `@${connection.name}.${schema.name}.${table.name}`,
                        insertText: fullName,
                        documentation: `Table from ${connection.name}`,
                        sortText: `1_${table.name}`,
                        range: partialTable ? {
                          startLineNumber: position.lineNumber,
                          endLineNumber: position.lineNumber,
                          startColumn: position.column - partialTable.length,
                          endColumn: position.column,
                        } : range
                      });
                    });
                }
              });

              return { suggestions };
            }
          }

          // If we didn't match any specific pattern but might be typing keywords, show defaults
          if (!trimmedBeforeCursor.includes('@') || isTypingKeyword) {
            return this.getDefaultSuggestions(range);
          }
        }

        // Single-DB mode - always show SQL keywords and tables
        if (this.mode === 'single') {
          // Check for table.column pattern
          const tablePattern = /(\w+)\.(\w*)$/;
          const tableMatch = trimmedBeforeCursor.match(tablePattern);

          if (tableMatch) {
            const [, tableName, partialColumn] = tableMatch;

            // Try to find the table and load columns
            if (this.schemas.size > 0) {
              const firstSchemaSet = Array.from(this.schemas.values())[0];
              if (firstSchemaSet) {
                for (const schemaNode of firstSchemaSet) {
                  if (schemaNode.type === 'schema' && schemaNode.children) {
                    const table = schemaNode.children.find(t =>
                      t.type === 'table' && t.name.toLowerCase() === tableName.toLowerCase()
                    );

                    if (table && this.connections.length > 0 && this.connections[0].sessionId) {
                      const columns = await this.loadColumnsForTable(
                        this.connections[0].sessionId,
                        schemaNode.name,
                        tableName
                      );

                      if (columns.length > 0) {
                        return {
                          suggestions: columns
                            .filter(col => !partialColumn || col.name.toLowerCase().startsWith(partialColumn.toLowerCase()))
                            .map(col => ({
                              label: col.name,
                              kind: monaco.languages.CompletionItemKind.Property,
                              insertText: col.name,
                              detail: `${col.dataType || 'unknown'}`,
                              documentation: col.nullable ? 'Nullable' : 'Not null',
                              sortText: `1_${col.name}`,
                              range: {
                                startLineNumber: position.lineNumber,
                                endLineNumber: position.lineNumber,
                                startColumn: position.column - partialColumn.length,
                                endColumn: position.column,
                              }
                            }))
                        };
                      }
                    }
                  }
                }
              }
            }
          }
        }

        // Default: Always show SQL keywords and available tables
        return this.getDefaultSuggestions(range, !isTypingKeyword);
        } catch (error) {
          console.warn('Error in completion provider:', error);
          // Even on error, provide basic SQL keywords
          return this.getDefaultSuggestions(range);
        }
      },
    });

    this.disposables.push(provider);
  }

  private registerHoverProvider() {
    if (this.mode !== 'multi') return;

    const provider = monaco.languages.registerHoverProvider('sql', {
      provideHover: (model, position) => {
        const word = model.getWordAtPosition(position);
        if (!word) return null;

        const line = model.getLineContent(position.lineNumber);
        const beforeWord = line.substring(0, word.startColumn - 1);

        // Check if this is a @connection reference
        if (beforeWord.endsWith('@')) {
          const connection = this.findConnectionByIdentifier(word.word);

          if (connection) {
            return {
              contents: [
                { value: `**Connection: ${connection.name}**` },
                { value: `Type: ${connection.type}` },
                { value: `Database: ${connection.database}` },
                { value: `Status: ${connection.isConnected ? '🟢 Connected' : '🔴 Disconnected'}` }
              ],
            };
          }
        }

        return null;
      },
    });

    this.disposables.push(provider);
  }

  private getDefaultSuggestions(range: any, includeKeywords: boolean = true) {
    const suggestions: any[] = [];

    // Always include SQL keywords unless explicitly disabled
    if (includeKeywords !== false) {
      const sqlKeywords = [
        'SELECT', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT', 'INNER',
        'OUTER', 'ON', 'AS', 'AND', 'OR', 'ORDER', 'BY', 'GROUP',
        'HAVING', 'LIMIT', 'OFFSET', 'UNION', 'ALL', 'DISTINCT',
        'INSERT', 'INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE',
        'CREATE', 'TABLE', 'DROP', 'ALTER', 'INDEX', 'VIEW',
        'WITH', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END', 'EXISTS',
        'NOT', 'NULL', 'IS', 'IN', 'LIKE', 'BETWEEN', 'CAST',
        'COUNT', 'SUM', 'AVG', 'MIN', 'MAX'
      ];

      suggestions.push(...sqlKeywords.map((keyword, idx) => ({
        label: keyword,
        kind: monaco.languages.CompletionItemKind.Keyword,
        insertText: keyword,
        detail: 'SQL keyword',
        range,
        sortText: `0_${idx.toString().padStart(3, '0')}_${keyword}`,
      })));
    }

    // Add tables from schemas
    if (this.schemas.size > 0) {
      const schemasToProcess = this.mode === 'single'
        ? [Array.from(this.schemas.values())[0]] // Single mode: use first schema set
        : Array.from(this.schemas.values()); // Multi mode: use all schema sets

      schemasToProcess.forEach(schemaSet => {
        if (schemaSet) {
          schemaSet.forEach(schemaNode => {
            if (schemaNode.type === 'schema' && schemaNode.children) {
              schemaNode.children.forEach(table => {
                if (table.type === 'table') {
                  suggestions.push({
                    label: table.name,
                    kind: monaco.languages.CompletionItemKind.Class,
                    insertText: table.name,
                    detail: `Table in ${schemaNode.name}`,
                    documentation: this.mode === 'multi' && this.connections.length > 1
                      ? `Use @connection.${table.name} for multi-DB queries`
                      : undefined,
                    range,
                    sortText: `1_${table.name}`,
                  });
                }
              });
            }
          });
        }
      });
    }

    // If we have no suggestions yet, at least provide basic SQL keywords
    if (suggestions.length === 0) {
      const basicKeywords = ['SELECT', 'FROM', 'WHERE', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP'];
      suggestions.push(...basicKeywords.map((keyword, idx) => ({
        label: keyword,
        kind: monaco.languages.CompletionItemKind.Keyword,
        insertText: keyword,
        detail: 'SQL keyword',
        range,
        sortText: `0_${idx.toString().padStart(3, '0')}`,
      })));
    }

    return { suggestions };
  }
}

/**
 * Legacy function for backward compatibility
 */
export function configureMultiDBLanguage(
  editor: monaco.editor.IStandaloneCodeEditor,
  connections: Connection[],
  schemas: Map<string, SchemaNode[]>,
  columnLoader?: ColumnLoader,
  mode: 'single' | 'multi' = 'single'
): MultiDBMonacoProvider {
  return new MultiDBMonacoProvider(editor, connections, schemas, columnLoader, mode);
}