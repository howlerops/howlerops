/**
 * Tests for AI Schema Context Builder
 */

import { AISchemaContextBuilder } from '../ai-schema-context'
import type { DatabaseConnection } from '@/store/connection-store'
import type { SchemaNode } from '@/hooks/use-schema-introspection'

// Type for connection info
type ConnectionInfo = Pick<DatabaseConnection, 'id' | 'name' | 'database' | 'isConnected'>

describe('AISchemaContextBuilder', () => {
  const mockSingleDBConnection: ConnectionInfo = {
    id: 'conn1',
    name: 'ProductionDB',
    database: 'ecommerce',
    isConnected: true
  }

  const mockMultiDBConnections: ConnectionInfo[] = [
    {
      id: 'conn1',
      name: 'Production',
      database: 'prod_db',
      isConnected: true
    },
    {
      id: 'conn2',
      name: 'Analytics',
      database: 'analytics_db',
      isConnected: true
    },
    {
      id: 'conn3',
      name: 'Staging',
      database: 'staging_db',
      isConnected: false
    }
  ]

  const mockSchemas: SchemaNode[] = [
    {
      name: 'public',
      type: 'schema' as const,
      children: [
        {
          name: 'users',
          type: 'table' as const,
          children: [
            { name: 'id', type: 'column' as const, dataType: 'integer', primaryKey: true },
            { name: 'email', type: 'column' as const, dataType: 'varchar', nullable: false },
            { name: 'created_at', type: 'column' as const, dataType: 'timestamp', nullable: false }
          ]
        },
        {
          name: 'orders',
          type: 'table' as const,
          children: [
            { name: 'id', type: 'column' as const, dataType: 'integer', primaryKey: true },
            { name: 'user_id', type: 'column' as const, dataType: 'integer', nullable: false },
            { name: 'total', type: 'column' as const, dataType: 'decimal', nullable: false },
            { name: 'status', type: 'column' as const, dataType: 'varchar', nullable: false }
          ]
        }
      ]
    }
  ]

  const mockAnalyticsSchemas: SchemaNode[] = [
    {
      name: 'metrics',
      type: 'schema' as const,
      children: [
        {
          name: 'daily_stats',
          type: 'table' as const,
          children: [
            { name: 'date', type: 'column' as const, dataType: 'date', primaryKey: true },
            { name: 'revenue', type: 'column' as const, dataType: 'decimal', nullable: false },
            { name: 'orders_count', type: 'column' as const, dataType: 'integer', nullable: false }
          ]
        }
      ]
    }
  ]

  describe('buildSingleDatabaseContext', () => {
    it('should build context for single database mode', () => {
      const context = AISchemaContextBuilder.buildSingleDatabaseContext(
        mockSingleDBConnection,
        mockSchemas
      )

      expect(context.mode).toBe('single')
      expect(context.databases).toHaveLength(1)
      expect(context.databases[0].connectionName).toBe('ProductionDB')
      expect(context.databases[0].schemas).toHaveLength(1)
      expect(context.databases[0].schemas[0].tables).toHaveLength(2)
    })

    it('should generate single database SQL examples', () => {
      const context = AISchemaContextBuilder.buildSingleDatabaseContext(
        mockSingleDBConnection,
        mockSchemas
      )

      expect(context.syntaxExamples).toBeDefined()
      expect(context.syntaxExamples![0]).toContain('SELECT * FROM users')
    })
  })

  describe('buildMultiDatabaseContext', () => {
    it('should build context for multiple connected databases', () => {
      const schemasMap = new Map([
        ['conn1', mockSchemas],
        ['Production', mockSchemas],
        ['conn2', mockAnalyticsSchemas],
        ['Analytics', mockAnalyticsSchemas]
      ])

      const context = AISchemaContextBuilder.buildMultiDatabaseContext(
        mockMultiDBConnections,
        schemasMap,
        'conn1'
      )

      expect(context.mode).toBe('multi')
      expect(context.databases).toHaveLength(2) // Only connected databases
      expect(context.activeConnectionId).toBe('conn1')
    })

    it('should generate multi-database SQL examples with @ syntax', () => {
      const schemasMap = new Map([
        ['conn1', mockSchemas],
        ['Production', mockSchemas],
        ['conn2', mockAnalyticsSchemas],
        ['Analytics', mockAnalyticsSchemas]
      ])

      const context = AISchemaContextBuilder.buildMultiDatabaseContext(
        mockMultiDBConnections,
        schemasMap,
        'conn1'
      )

      const examples = context.syntaxExamples || []
      expect(examples.some(ex => ex.includes('@Production.users'))).toBe(true)
      expect(examples.some(ex => ex.includes('@Analytics.metrics.daily_stats'))).toBe(true)
    })

    it('should filter out disconnected databases', () => {
      const schemasMap = new Map([
        ['conn1', mockSchemas],
        ['conn3', mockSchemas] // Staging is disconnected
      ])

      const context = AISchemaContextBuilder.buildMultiDatabaseContext(
        mockMultiDBConnections,
        schemasMap
      )

      expect(context.databases).toHaveLength(1)
      expect(context.databases[0].connectionId).toBe('conn1')
    })
  })

  describe('generateAIPrompt', () => {
    it('should generate prompt for single database mode', () => {
      const context = AISchemaContextBuilder.buildSingleDatabaseContext(
        mockSingleDBConnection,
        mockSchemas
      )

      const prompt = AISchemaContextBuilder.generateAIPrompt(
        'Show me all users',
        context
      )

      expect(prompt).toContain('Database: ecommerce')
      expect(prompt).toContain('users:')
      expect(prompt).toContain('- id (integer, PK)')
      expect(prompt).toContain('User Request: Show me all users')
    })

    it('should generate prompt for multi-database mode', () => {
      const schemasMap = new Map([
        ['conn1', mockSchemas],
        ['Production', mockSchemas],
        ['conn2', mockAnalyticsSchemas],
        ['Analytics', mockAnalyticsSchemas]
      ])

      const context = AISchemaContextBuilder.buildMultiDatabaseContext(
        mockMultiDBConnections,
        schemasMap
      )

      const prompt = AISchemaContextBuilder.generateAIPrompt(
        'Join users with daily stats',
        context
      )

      expect(prompt).toContain('Multi-Database Mode Active')
      expect(prompt).toContain('@connection_name.schema.table')
      expect(prompt).toContain('@Production.users')
      expect(prompt).toContain('@Analytics.metrics.daily_stats')
      expect(prompt).toContain('User Request: Join users with daily stats')
    })
  })

  describe('generateCompactSchemaContext', () => {
    it('should generate compact context for single database', () => {
      const context = AISchemaContextBuilder.buildSingleDatabaseContext(
        mockSingleDBConnection,
        mockSchemas
      )

      const compact = AISchemaContextBuilder.generateCompactSchemaContext(context)

      expect(compact).toContain('DB: ecommerce')
      expect(compact).toContain('users(id,email,created_at)')
      expect(compact).toContain('orders(id,user_id,total,status)')
    })

    it('should generate compact context for multi-database', () => {
      const schemasMap = new Map([
        ['conn1', mockSchemas],
        ['Production', mockSchemas],
        ['conn2', mockAnalyticsSchemas],
        ['Analytics', mockAnalyticsSchemas]
      ])

      const context = AISchemaContextBuilder.buildMultiDatabaseContext(
        mockMultiDBConnections,
        schemasMap
      )

      const compact = AISchemaContextBuilder.generateCompactSchemaContext(context)

      expect(compact).toContain('Multi-DB Mode')
      expect(compact).toContain('@Production.users')
      expect(compact).toContain('@Analytics.metrics.daily_stats')
    })

    it('should limit columns in compact mode', () => {
      const largeTableSchema = [
        {
          name: 'public',
          type: 'schema' as const,
          children: [
            {
              name: 'large_table',
              type: 'table' as const,
              children: Array.from({ length: 20 }, (_, i) => ({
                name: `col${i}`,
                type: 'column' as const,
                dataType: 'varchar'
              }))
            }
          ]
        }
      ]

      const context = AISchemaContextBuilder.buildSingleDatabaseContext(
        mockSingleDBConnection,
        largeTableSchema
      )

      const compact = AISchemaContextBuilder.generateCompactSchemaContext(context)

      // Should only show first 5 columns in single DB mode
      expect(compact).toContain('col0,col1,col2,col3,col4,...')
      expect(compact).not.toContain('col10')
    })
  })
})