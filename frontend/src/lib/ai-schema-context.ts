/**
 * AI Schema Context Builder for Multi-Database Support
 *
 * This module builds comprehensive schema context for AI SQL generation
 * that includes information from multiple connected databases.
 */

import { DatabaseConnection } from '@/store/connection-store'
import { SchemaNode } from '@/hooks/use-schema-introspection'

// Type alias for compatibility
type ConnectionInfo = Pick<DatabaseConnection, 'id' | 'name' | 'database' | 'isConnected'>

export interface DatabaseSchemaContext {
  connectionId: string
  connectionName: string
  database: string
  schemas: SchemaInfo[]
}

export interface SchemaInfo {
  name: string
  tables: TableInfo[]
}

export interface TableInfo {
  name: string
  columns: ColumnInfo[]
  primaryKeys: string[]
  foreignKeys: ForeignKeyInfo[]
}

export interface ColumnInfo {
  name: string
  dataType: string
  nullable: boolean
  primaryKey?: boolean
  unique?: boolean
  defaultValue?: string
}

export interface ForeignKeyInfo {
  column: string
  referencedTable: string
  referencedColumn: string
  referencedSchema?: string
}

export interface MultiDatabaseContext {
  mode: 'single' | 'multi'
  activeConnectionId?: string
  databases: DatabaseSchemaContext[]
  syntaxExamples?: string[]
}

/**
 * Builds schema context for AI SQL generation in multi-database mode
 */
export class AISchemaContextBuilder {
  /**
   * Build context for single database mode
   */
  static buildSingleDatabaseContext(
    connection: ConnectionInfo,
    schemas: SchemaNode[]
  ): MultiDatabaseContext {
    const dbContext: DatabaseSchemaContext = {
      connectionId: connection.id,
      connectionName: connection.name,
      database: connection.database,
      schemas: this.convertSchemaNodesToSchemaInfo(schemas)
    }

    return {
      mode: 'single',
      activeConnectionId: connection.id,
      databases: [dbContext],
      syntaxExamples: this.generateSingleDBExamples(dbContext)
    }
  }

  /**
   * Build context for multi-database mode
   */
  static buildMultiDatabaseContext(
    connections: ConnectionInfo[],
    schemasMap: Map<string, SchemaNode[]>,
    activeConnectionId?: string
  ): MultiDatabaseContext {
    const databases: DatabaseSchemaContext[] = []

    // Process each connected database
    for (const conn of connections) {
      if (!conn.isConnected) continue

      const schemas = schemasMap.get(conn.id) || schemasMap.get(conn.name) || []
      if (schemas.length === 0) continue

      databases.push({
        connectionId: conn.id,
        connectionName: conn.name,
        database: conn.database,
        schemas: this.convertSchemaNodesToSchemaInfo(schemas)
      })
    }

    return {
      mode: 'multi',
      activeConnectionId,
      databases,
      syntaxExamples: this.generateMultiDBExamples(databases)
    }
  }

  /**
   * Convert SchemaNode structure to SchemaInfo for AI context
   */
  private static convertSchemaNodesToSchemaInfo(schemas: SchemaNode[]): SchemaInfo[] {
    return schemas.map(schema => ({
      name: schema.name,
      tables: (schema.children || []).map(table => ({
        name: table.name,
        columns: (table.children || []).map(col => ({
          name: col.name,
          dataType: (col.metadata as any)?.dataType || 'unknown',
          nullable: (col.metadata as any)?.nullable || false,
          primaryKey: (col.metadata as any)?.primaryKey,
          unique: (col.metadata as any)?.unique,
          defaultValue: (col.metadata as any)?.defaultValue
        })),
        primaryKeys: (table.children || [])
          .filter(col => (col.metadata as any)?.primaryKey)
          .map(col => col.name),
        foreignKeys: [] // TODO: Extract foreign key information if available
      }))
    }))
  }

  /**
   * Generate SQL syntax examples for single database mode
   */
  private static generateSingleDBExamples(context: DatabaseSchemaContext): string[] {
    const examples: string[] = []

    if (context.schemas.length > 0 && context.schemas[0].tables.length > 0) {
      const firstTable = context.schemas[0].tables[0]
      const tableName = context.schemas[0].name === 'public'
        ? firstTable.name
        : `${context.schemas[0].name}.${firstTable.name}`

      examples.push(`SELECT * FROM ${tableName} LIMIT 10`)

      if (firstTable.columns.length > 0) {
        const columns = firstTable.columns.slice(0, 3).map(c => c.name).join(', ')
        examples.push(`SELECT ${columns} FROM ${tableName}`)
      }
    }

    return examples
  }

  /**
   * Generate SQL syntax examples for multi-database mode
   */
  private static generateMultiDBExamples(databases: DatabaseSchemaContext[]): string[] {
    const examples: string[] = []

    // Add multi-DB syntax explanation
    examples.push('-- Multi-database query syntax: @connection_name.schema.table')

    // Generate examples for each database
    for (const db of databases.slice(0, 2)) { // Limit to first 2 databases for brevity
      if (db.schemas.length > 0 && db.schemas[0].tables.length > 0) {
        const firstTable = db.schemas[0].tables[0]
        const tablePath = db.schemas[0].name === 'public'
          ? `@${db.connectionName}.${firstTable.name}`
          : `@${db.connectionName}.${db.schemas[0].name}.${firstTable.name}`

        examples.push(`SELECT * FROM ${tablePath} LIMIT 10`)
      }
    }

    // Add cross-database JOIN example if we have at least 2 databases
    if (databases.length >= 2) {
      const db1 = databases[0]
      const db2 = databases[1]

      if (db1.schemas.length > 0 && db1.schemas[0].tables.length > 0 &&
          db2.schemas.length > 0 && db2.schemas[0].tables.length > 0) {
        const table1 = db1.schemas[0].tables[0]
        const table2 = db2.schemas[0].tables[0]

        const path1 = db1.schemas[0].name === 'public'
          ? `@${db1.connectionName}.${table1.name}`
          : `@${db1.connectionName}.${db1.schemas[0].name}.${table1.name}`

        const path2 = db2.schemas[0].name === 'public'
          ? `@${db2.connectionName}.${table2.name}`
          : `@${db2.connectionName}.${db2.schemas[0].name}.${table2.name}`

        examples.push(`-- Cross-database JOIN example:`)
        examples.push(`SELECT t1.*, t2.* FROM ${path1} t1`)
        examples.push(`JOIN ${path2} t2 ON t1.id = t2.foreign_id`)
      }
    }

    return examples
  }

  /**
   * Generate a text prompt for the AI with schema context
   */
  static generateAIPrompt(
    userPrompt: string,
    context: MultiDatabaseContext
  ): string {
    let prompt = `Database Schema Context:\n\n`

    if (context.mode === 'single') {
      // Single database mode
      const db = context.databases[0]
      prompt += `Database: ${db.database}\n`
      prompt += `Connection: ${db.connectionName}\n\n`

      prompt += `Available Tables:\n`
      for (const schema of db.schemas) {
        for (const table of schema.tables) {
          const tableName = schema.name === 'public'
            ? table.name
            : `${schema.name}.${table.name}`

          prompt += `- ${tableName}:\n`
          for (const col of table.columns.slice(0, 10)) { // Limit columns for brevity
            prompt += `  - ${col.name} (${col.dataType}${col.nullable ? ', nullable' : ''}${col.primaryKey ? ', PK' : ''})\n`
          }
          if (table.columns.length > 10) {
            prompt += `  ... and ${table.columns.length - 10} more columns\n`
          }
        }
      }
    } else {
      // Multi-database mode
      prompt += `Multi-Database Mode Active\n`
      prompt += `Syntax: Use @connection_name.schema.table to reference tables from different databases\n\n`

      for (const db of context.databases) {
        prompt += `\nDatabase: ${db.connectionName} (${db.database})\n`
        prompt += `Available Tables:\n`

        for (const schema of db.schemas) {
          for (const table of schema.tables) {
            const tablePath = schema.name === 'public'
              ? `@${db.connectionName}.${table.name}`
              : `@${db.connectionName}.${schema.name}.${table.name}`

            prompt += `- ${tablePath}:\n`
            for (const col of table.columns.slice(0, 5)) { // Fewer columns in multi-DB mode
              prompt += `  - ${col.name} (${col.dataType}${col.primaryKey ? ', PK' : ''})\n`
            }
            if (table.columns.length > 5) {
              prompt += `  ... and ${table.columns.length - 5} more columns\n`
            }
          }
        }
      }

      prompt += `\nExamples:\n`
      for (const example of context.syntaxExamples || []) {
        prompt += `${example}\n`
      }
    }

    prompt += `\n\nUser Request: ${userPrompt}\n`
    prompt += `\nGenerate a SQL query that fulfills the user's request.`

    if (context.mode === 'multi') {
      prompt += ` Use the @connection_name.table syntax for tables from different databases.`
    }

    prompt += ` Return only the SQL query without any explanation or markdown formatting.`

    return prompt
  }

  /**
   * Generate a simplified schema summary for token-efficient AI requests
   */
  static generateCompactSchemaContext(context: MultiDatabaseContext): string {
    let summary = ''

    if (context.mode === 'single') {
      const db = context.databases[0]
      summary += `DB: ${db.database}\n`

      const tables: string[] = []
      for (const schema of db.schemas) {
        for (const table of schema.tables) {
          const name = schema.name === 'public' ? table.name : `${schema.name}.${table.name}`
          const cols = table.columns.map(c => c.name).slice(0, 5).join(',')
          tables.push(`${name}(${cols}${table.columns.length > 5 ? ',...' : ''})`)
        }
      }
      summary += `Tables: ${tables.join('; ')}`
    } else {
      summary += `Multi-DB Mode (@conn.table syntax)\n`

      for (const db of context.databases) {
        const tables: string[] = []
        for (const schema of db.schemas) {
          for (const table of schema.tables) {
            const name = schema.name === 'public'
              ? `@${db.connectionName}.${table.name}`
              : `@${db.connectionName}.${schema.name}.${table.name}`
            const cols = table.columns.map(c => c.name).slice(0, 3).join(',')
            tables.push(`${name}(${cols}${table.columns.length > 3 ? ',...' : ''})`)
          }
        }
        if (tables.length > 0) {
          summary += `${db.connectionName}: ${tables.join('; ')}\n`
        }
      }
    }

    return summary
  }
}