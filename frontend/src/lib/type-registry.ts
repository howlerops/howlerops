/**
 * Type Registry for Visual Query Builder
 * Maps database column metadata to canonical field types and operators
 */

import { FilterOperator } from '@/workers/types'

export type FieldType = 
  | 'text' 
  | 'number' 
  | 'boolean' 
  | 'date' 
  | 'datetime' 
  | 'enum' 
  | 'uuid' 
  | 'json' 
  | 'array'

export interface ColumnMetadata {
  name: string
  dataType?: string
  characterMaximumLength?: number
  numericPrecision?: number
  numericScale?: number
  isNullable?: string
  columnDefault?: string | null
  isPrimaryKey?: boolean
  isForeignKey?: boolean
  enumValues?: string[]
  foreignKeyTable?: string
  foreignKeyColumn?: string
}

export interface TypeConfig {
  fieldType: FieldType
  operators: FilterOperator[]
  validator: (value: unknown) => boolean
  serializer: (value: unknown) => string
  defaultValue: unknown
}

export class TypeRegistry {
  private static instance: TypeRegistry
  private typeMap = new Map<string, TypeConfig>()

  private constructor() {
    this.initializeTypes()
  }

  static getInstance(): TypeRegistry {
    if (!TypeRegistry.instance) {
      TypeRegistry.instance = new TypeRegistry()
    }
    return TypeRegistry.instance
  }

  private initializeTypes() {
    // Text type
    this.typeMap.set('text', {
      fieldType: 'text',
      operators: [
        FilterOperator.EQUALS,
        FilterOperator.NOT_EQUALS,
        FilterOperator.CONTAINS,
        FilterOperator.NOT_CONTAINS,
        FilterOperator.STARTS_WITH,
        FilterOperator.ENDS_WITH,
        FilterOperator.IN,
        FilterOperator.NOT_IN,
        FilterOperator.IS_NULL,
        FilterOperator.IS_NOT_NULL,
        FilterOperator.REGEX
      ],
      validator: (value) => typeof value === 'string',
      serializer: (value) => String(value),
      defaultValue: ''
    })

    // Number type
    this.typeMap.set('number', {
      fieldType: 'number',
      operators: [
        FilterOperator.EQUALS,
        FilterOperator.NOT_EQUALS,
        FilterOperator.GREATER_THAN,
        FilterOperator.GREATER_THAN_OR_EQUALS,
        FilterOperator.LESS_THAN,
        FilterOperator.LESS_THAN_OR_EQUALS,
        FilterOperator.BETWEEN,
        FilterOperator.IN,
        FilterOperator.NOT_IN,
        FilterOperator.IS_NULL,
        FilterOperator.IS_NOT_NULL
      ],
      validator: (value) => typeof value === 'number' && !isNaN(value),
      serializer: (value) => String(value),
      defaultValue: 0
    })

    // Boolean type
    this.typeMap.set('boolean', {
      fieldType: 'boolean',
      operators: [
        FilterOperator.EQUALS,
        FilterOperator.NOT_EQUALS,
        FilterOperator.IS_NULL,
        FilterOperator.IS_NOT_NULL
      ],
      validator: (value) => typeof value === 'boolean',
      serializer: (value) => String(value),
      defaultValue: false
    })

    // Date type
    this.typeMap.set('date', {
      fieldType: 'date',
      operators: [
        FilterOperator.EQUALS,
        FilterOperator.NOT_EQUALS,
        FilterOperator.GREATER_THAN,
        FilterOperator.GREATER_THAN_OR_EQUALS,
        FilterOperator.LESS_THAN,
        FilterOperator.LESS_THAN_OR_EQUALS,
        FilterOperator.BETWEEN,
        FilterOperator.IS_NULL,
        FilterOperator.IS_NOT_NULL
      ],
      validator: (value) => value instanceof Date || (typeof value === 'string' && !isNaN(Date.parse(value))),
      serializer: (value) => value instanceof Date ? value.toISOString().split('T')[0] : String(value),
      defaultValue: new Date().toISOString().split('T')[0]
    })

    // DateTime type
    this.typeMap.set('datetime', {
      fieldType: 'datetime',
      operators: [
        FilterOperator.EQUALS,
        FilterOperator.NOT_EQUALS,
        FilterOperator.GREATER_THAN,
        FilterOperator.GREATER_THAN_OR_EQUALS,
        FilterOperator.LESS_THAN,
        FilterOperator.LESS_THAN_OR_EQUALS,
        FilterOperator.BETWEEN,
        FilterOperator.IS_NULL,
        FilterOperator.IS_NOT_NULL
      ],
      validator: (value) => value instanceof Date || (typeof value === 'string' && !isNaN(Date.parse(value))),
      serializer: (value) => value instanceof Date ? value.toISOString() : String(value),
      defaultValue: new Date().toISOString()
    })

    // Enum type
    this.typeMap.set('enum', {
      fieldType: 'enum',
      operators: [
        FilterOperator.EQUALS,
        FilterOperator.NOT_EQUALS,
        FilterOperator.IN,
        FilterOperator.NOT_IN,
        FilterOperator.IS_NULL,
        FilterOperator.IS_NOT_NULL
      ],
      validator: (value) => typeof value === 'string',
      serializer: (value) => String(value),
      defaultValue: ''
    })

    // UUID type
    this.typeMap.set('uuid', {
      fieldType: 'uuid',
      operators: [
        FilterOperator.EQUALS,
        FilterOperator.NOT_EQUALS,
        FilterOperator.IN,
        FilterOperator.NOT_IN,
        FilterOperator.IS_NULL,
        FilterOperator.IS_NOT_NULL
      ],
      validator: (value) => typeof value === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value),
      serializer: (value) => String(value),
      defaultValue: ''
    })

    // JSON type
    this.typeMap.set('json', {
      fieldType: 'json',
      operators: [
        FilterOperator.EQUALS,
        FilterOperator.NOT_EQUALS,
        FilterOperator.CONTAINS,
        FilterOperator.IS_NULL,
        FilterOperator.IS_NOT_NULL
      ],
      validator: (value) => {
        try {
          JSON.parse(typeof value === 'string' ? value : JSON.stringify(value))
          return true
        } catch {
          return false
        }
      },
      serializer: (value) => typeof value === 'string' ? value : JSON.stringify(value),
      defaultValue: '{}'
    })

    // Array type
    this.typeMap.set('array', {
      fieldType: 'array',
      operators: [
        FilterOperator.CONTAINS,
        FilterOperator.NOT_CONTAINS,
        FilterOperator.IS_NULL,
        FilterOperator.IS_NOT_NULL
      ],
      validator: (value) => Array.isArray(value),
      serializer: (value) => JSON.stringify(value),
      defaultValue: []
    })
  }

  /**
   * Infer field type from database column metadata
   */
  inferFieldType(column: ColumnMetadata): FieldType {
    const dataType = column.dataType?.toLowerCase() || ''
    
    // Boolean types
    if (dataType.includes('bool') || dataType.includes('bit')) {
      return 'boolean'
    }
    
    // Numeric types
    if (dataType.includes('int') || dataType.includes('decimal') || 
        dataType.includes('numeric') || dataType.includes('float') || 
        dataType.includes('double') || dataType.includes('real')) {
      return 'number'
    }
    
    // Date/Time types
    if (dataType.includes('date') && !dataType.includes('datetime')) {
      return 'date'
    }
    if (dataType.includes('datetime') || dataType.includes('timestamp')) {
      return 'datetime'
    }
    
    // UUID types
    if (dataType.includes('uuid') || dataType.includes('guid')) {
      return 'uuid'
    }
    
    // JSON types
    if (dataType.includes('json')) {
      return 'json'
    }
    
    // Array types
    if (dataType.includes('array') || dataType.includes('[]')) {
      return 'array'
    }
    
    // Enum types
    if (dataType.includes('enum') || column.enumValues?.length) {
      return 'enum'
    }
    
    // Default to text
    return 'text'
  }

  /**
   * Get type configuration for a field type
   */
  getTypeConfig(fieldType: FieldType): TypeConfig {
    const config = this.typeMap.get(fieldType)
    if (!config) {
      throw new Error(`Unknown field type: ${fieldType}`)
    }
    return config
  }

  /**
   * Get type configuration for a column
   */
  getColumnTypeConfig(column: ColumnMetadata): TypeConfig {
    const fieldType = this.inferFieldType(column)
    return this.getTypeConfig(fieldType)
  }

  /**
   * Get available operators for a field type
   */
  getOperators(fieldType: FieldType): FilterOperator[] {
    return this.getTypeConfig(fieldType).operators
  }

  /**
   * Validate a value for a field type
   */
  validateValue(fieldType: FieldType, value: unknown): boolean {
    return this.getTypeConfig(fieldType).validator(value)
  }

  /**
   * Serialize a value for a field type
   */
  serializeValue(fieldType: FieldType, value: unknown): string {
    return this.getTypeConfig(fieldType).serializer(value)
  }

  /**
   * Get default value for a field type
   */
  getDefaultValue(fieldType: FieldType): unknown {
    return this.getTypeConfig(fieldType).defaultValue
  }
}

export const typeRegistry = TypeRegistry.getInstance()
