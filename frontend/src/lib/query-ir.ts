/**
 * Query Intermediate Representation (IR) for Visual Query Builder
 * Dialect-agnostic representation of SQL queries
 */

import { FilterOperator } from '@/workers/types'

export interface TableRef {
  schema: string
  table: string
  alias?: string
  connection?: string // Connection ID or name for multi-DB queries
}

export interface SelectItem {
  column: string
  alias?: string
  aggregate?: 'count' | 'sum' | 'avg' | 'min' | 'max'
}

export interface Join {
  type: 'inner' | 'left' | 'right' | 'full'
  table: TableRef
  on: Expr
}

export interface OrderBy {
  column: string
  direction: 'asc' | 'desc'
}

export interface Predicate {
  column: string
  operator: FilterOperator
  value: unknown
  not?: boolean
}

export interface Group {
  operator: 'AND' | 'OR'
  conditions: Expr[]
  not?: boolean
}

export interface Exists {
  subquery: QueryIR
  not?: boolean
}

export type Expr = Predicate | Group | Exists

export interface QueryIR {
  from: TableRef
  joins?: Join[]
  select: SelectItem[]
  where?: Expr
  orderBy?: OrderBy[]
  limit?: number
  offset?: number
}

export type SQLDialect = 'postgres' | 'mysql' | 'sqlite' | 'mssql'

export interface SQLGeneratorOptions {
  dialect: SQLDialect
  parameterized?: boolean
  indent?: string
}

export class SQLGenerator {
  private options: SQLGeneratorOptions

  constructor(options: SQLGeneratorOptions) {
    this.options = options
  }

  /**
   * Generate SQL from QueryIR
   */
  generateSQL(ir: QueryIR): string {
    const parts: string[] = []
    
    // SELECT clause
    const selectClause = this.generateSelect(ir.select)
    parts.push(selectClause)
    
    // FROM clause
    const fromClause = this.generateFrom(ir.from)
    parts.push(fromClause)
    
    // JOIN clauses
    if (ir.joins?.length) {
      const joinClauses = ir.joins.map(join => this.generateJoin(join))
      parts.push(...joinClauses)
    }
    
    // WHERE clause
    if (ir.where) {
      const whereClause = this.generateWhere(ir.where)
      parts.push(whereClause)
    }
    
    // ORDER BY clause
    if (ir.orderBy?.length) {
      const orderClause = this.generateOrderBy(ir.orderBy)
      parts.push(orderClause)
    }
    
    // LIMIT clause
    if (ir.limit) {
      const limitClause = this.generateLimit(ir.limit, ir.offset)
      parts.push(limitClause)
    }
    
    return parts.join('\n')
  }

  private generateSelect(select: SelectItem[]): string {
    if (select.length === 0) {
      return 'SELECT *'
    }
    
    const items = select.map(item => {
      let column = this.quoteIdentifier(item.column)
      
      if (item.aggregate) {
        column = `${item.aggregate.toUpperCase()}(${column})`
      }
      
      if (item.alias) {
        column += ` AS ${this.quoteIdentifier(item.alias)}`
      }
      
      return column
    })
    
    return `SELECT ${items.join(', ')}`
  }

  private generateFrom(from: TableRef): string {
    let table = this.quoteIdentifier(from.schema) + '.' + this.quoteIdentifier(from.table)
    
    if (from.alias) {
      table += ` AS ${this.quoteIdentifier(from.alias)}`
    }
    
    return `FROM ${table}`
  }

  private generateJoin(join: Join): string {
    const joinType = join.type.toUpperCase()
    const table = this.quoteIdentifier(join.table.schema) + '.' + this.quoteIdentifier(join.table.table)
    const alias = join.table.alias ? ` AS ${this.quoteIdentifier(join.table.alias)}` : ''
    const onClause = this.generateExpr(join.on)
    
    return `${joinType} JOIN ${table}${alias} ON ${onClause}`
  }

  private generateWhere(where: Expr): string {
    return `WHERE ${this.generateExpr(where)}`
  }

  private generateOrderBy(orderBy: OrderBy[]): string {
    const items = orderBy.map(item => {
      const column = this.quoteIdentifier(item.column)
      const direction = item.direction.toUpperCase()
      return `${column} ${direction}`
    })
    
    return `ORDER BY ${items.join(', ')}`
  }

  private generateLimit(limit: number, offset?: number): string {
    switch (this.options.dialect) {
      case 'postgres':
      case 'sqlite':
        if (offset) {
          return `LIMIT ${limit} OFFSET ${offset}`
        }
        return `LIMIT ${limit}`
      
      case 'mysql':
        if (offset) {
          return `LIMIT ${offset}, ${limit}`
        }
        return `LIMIT ${limit}`
      
      case 'mssql':
        // SQL Server uses TOP in SELECT, not LIMIT
        return `-- LIMIT ${limit} OFFSET ${offset || 0} (handled in SELECT)`
      
      default:
        return `LIMIT ${limit}`
    }
  }

  private generateExpr(expr: Expr): string {
    if ('column' in expr) {
      // Predicate
      return this.generatePredicate(expr)
    } else if ('operator' in expr) {
      // Group
      return this.generateGroup(expr)
    } else if ('subquery' in expr) {
      // Exists
      return this.generateExists(expr)
    }
    
    throw new Error('Invalid expression type')
  }

  private generatePredicate(predicate: Predicate): string {
    const column = this.quoteIdentifier(predicate.column)
    const operator = this.mapOperator(predicate.operator)
    const value = this.formatValue(predicate.value, predicate.operator)
    
    let condition = `${column} ${operator} ${value}`
    
    if (predicate.not) {
      condition = `NOT (${condition})`
    }
    
    return condition
  }

  private generateGroup(group: Group): string {
    const conditions = group.conditions.map(condition => this.generateExpr(condition))
    const operator = group.operator
    const combined = conditions.join(` ${operator} `)
    
    let result = `(${combined})`
    
    if (group.not) {
      result = `NOT ${result}`
    }
    
    return result
  }

  private generateExists(exists: Exists): string {
    const subquery = this.generateSQL(exists.subquery)
    let result = `EXISTS (${subquery})`
    
    if (exists.not) {
      result = `NOT ${result}`
    }
    
    return result
  }

  private mapOperator(operator: FilterOperator): string {
    switch (operator) {
      case FilterOperator.EQUALS:
        return '='
      case FilterOperator.NOT_EQUALS:
        return '!='
      case FilterOperator.GREATER_THAN:
        return '>'
      case FilterOperator.GREATER_THAN_OR_EQUALS:
        return '>='
      case FilterOperator.LESS_THAN:
        return '<'
      case FilterOperator.LESS_THAN_OR_EQUALS:
        return '<='
      case FilterOperator.CONTAINS:
        return this.options.dialect === 'postgres' ? 'ILIKE' : 'LIKE'
      case FilterOperator.NOT_CONTAINS:
        return this.options.dialect === 'postgres' ? 'NOT ILIKE' : 'NOT LIKE'
      case FilterOperator.STARTS_WITH:
        return this.options.dialect === 'postgres' ? 'ILIKE' : 'LIKE'
      case FilterOperator.ENDS_WITH:
        return this.options.dialect === 'postgres' ? 'ILIKE' : 'LIKE'
      case FilterOperator.IN:
        return 'IN'
      case FilterOperator.NOT_IN:
        return 'NOT IN'
      case FilterOperator.IS_NULL:
        return 'IS NULL'
      case FilterOperator.IS_NOT_NULL:
        return 'IS NOT NULL'
      case FilterOperator.REGEX:
        return this.options.dialect === 'postgres' ? '~' : 'REGEXP'
      case FilterOperator.BETWEEN:
        return 'BETWEEN'
      default:
        throw new Error(`Unsupported operator: ${operator}`)
    }
  }

  private formatValue(value: unknown, operator: FilterOperator): string {
    if (value === null || value === undefined) {
      return 'NULL'
    }
    
    if (Array.isArray(value)) {
      if (operator === FilterOperator.IN || operator === FilterOperator.NOT_IN) {
        const formatted = value.map(v => this.formatLiteral(v)).join(', ')
        return `(${formatted})`
      }
      if (operator === FilterOperator.BETWEEN && value.length === 2) {
        return `${this.formatLiteral(value[0])} AND ${this.formatLiteral(value[1])}`
      }
    }
    
    if (operator === FilterOperator.CONTAINS || operator === FilterOperator.NOT_CONTAINS) {
      return this.formatLiteral(`%${value}%`)
    }
    
    if (operator === FilterOperator.STARTS_WITH) {
      return this.formatLiteral(`${value}%`)
    }
    
    if (operator === FilterOperator.ENDS_WITH) {
      return this.formatLiteral(`%${value}`)
    }
    
    return this.formatLiteral(value)
  }

  private formatLiteral(value: unknown): string {
    if (typeof value === 'string') {
      return `'${value.replace(/'/g, "''")}'`
    }
    
    if (typeof value === 'number') {
      return String(value)
    }
    
    if (typeof value === 'boolean') {
      return this.options.dialect === 'mssql' ? (value ? '1' : '0') : String(value)
    }
    
    if (value instanceof Date) {
      const iso = value.toISOString()
      if (this.options.dialect === 'mssql') {
        return `'${iso}'`
      }
      return `'${iso}'`
    }
    
    return `'${String(value)}'`
  }

  private quoteIdentifier(identifier: string): string {
    switch (this.options.dialect) {
      case 'postgres':
        return `"${identifier}"`
      case 'mysql':
        return `\`${identifier}\``
      case 'sqlite':
        return `"${identifier}"`
      case 'mssql':
        return `[${identifier}]`
      default:
        return `"${identifier}"`
    }
  }
}

/**
 * Create a SQL generator for the specified dialect
 */
export function createSQLGenerator(dialect: SQLDialect, options: Partial<SQLGeneratorOptions> = {}): SQLGenerator {
  return new SQLGenerator({
    dialect,
    parameterized: false,
    indent: '  ',
    ...options
  })
}

/**
 * Generate SQL from QueryIR for a specific dialect
 */
export function generateSQL(ir: QueryIR, dialect: SQLDialect, options: Partial<SQLGeneratorOptions> = {}): string {
  const generator = createSQLGenerator(dialect, options)
  return generator.generateSQL(ir)
}
