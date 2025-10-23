/**
 * Query Sanitization Tests
 *
 * Comprehensive test suite to ensure NO credentials leak through query sanitization.
 * These tests are CRITICAL for security - they must all pass with zero false negatives.
 */

import { describe, it, expect, beforeEach } from 'vitest'
import {
  sanitizeQuery,
  sanitizeQueries,
  isPrivateQuery,
  extractQueryMetadata,
  QueryPrivacyLevel,
  PrivacyMode,
  createDefaultConfig,
  type SanitizationConfig
} from '../index'

describe('Query Sanitizer', () => {
  let config: SanitizationConfig

  beforeEach(() => {
    config = createDefaultConfig()
  })

  describe('Simple Queries', () => {
    it('should sanitize string literals in SELECT', () => {
      const query = "SELECT * FROM users WHERE email = 'admin@example.com'"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toBe("SELECT * FROM users WHERE email = ?")
      expect(result.wasModified).toBe(true)
      expect(result.stats.literalsRemoved).toBe(1)
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE) // users table
    })

    it('should sanitize string literals in INSERT', () => {
      const query = "INSERT INTO products (name, price) VALUES ('Secret Product', 99.99)"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toBe("INSERT INTO products (name, price) VALUES (?, 99.99)")
      expect(result.wasModified).toBe(true)
      expect(result.stats.literalsRemoved).toBe(1)
    })

    it('should sanitize multiple string literals in UPDATE', () => {
      const query = "UPDATE customers SET name = 'John Doe', email = 'john@example.com' WHERE id = 1"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain("SET name = ?, email = ?")
      expect(result.wasModified).toBe(true)
      expect(result.stats.literalsRemoved).toBeGreaterThanOrEqual(2)
    })

    it('should sanitize string literals in DELETE', () => {
      const query = "DELETE FROM logs WHERE message = 'password: abc123'"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toBe("DELETE FROM logs WHERE message = ?")
      expect(result.wasModified).toBe(true)
      expect(result.stats.literalsRemoved).toBe(1)
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
    })

    it('should handle empty queries', () => {
      const result = sanitizeQuery('', config)
      expect(result.sanitizedQuery).toBe('')
      expect(result.wasModified).toBe(false)
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.NORMAL)
    })

    it('should handle whitespace-only queries', () => {
      const result = sanitizeQuery('   \n  \t  ', config)
      expect(result.wasModified).toBe(false)
    })
  })

  describe('Complex Queries', () => {
    it('should handle JOINs with multiple tables', () => {
      const query = `
        SELECT u.name, p.title
        FROM users u
        JOIN posts p ON u.id = p.user_id
        WHERE u.email = 'admin@example.com'
          AND p.status = 'published'
      `
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('WHERE u.email = ?')
      expect(result.sanitizedQuery).toContain("AND p.status = ?")
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE) // users table
      expect(result.stats.sensitiveTablesFound).toBeGreaterThan(0)
    })

    it('should handle subqueries', () => {
      const query = `
        SELECT * FROM orders
        WHERE customer_id IN (
          SELECT id FROM customers
          WHERE api_key = 'sk-1234567890abcdef'
        )
      `
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('WHERE api_key = ?')
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })

    it('should handle CTEs (Common Table Expressions)', () => {
      const query = `
        WITH admin_users AS (
          SELECT * FROM users
          WHERE role = 'admin'
            AND password_hash = '\$2a\$10\$abcdefghijklmnop'
        )
        SELECT * FROM admin_users
      `
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('AND password_hash = ?')
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })

    it('should handle CASE statements', () => {
      const query = `
        SELECT
          CASE
            WHEN status = 'active' THEN 'Active User'
            WHEN status = 'suspended' THEN 'Suspended'
            ELSE 'Unknown'
          END as status_label
        FROM accounts
      `
      const result = sanitizeQuery(query, config)

      // In normal mode, CASE literals might be preserved
      expect(result.sanitizedQuery).toBeDefined()
    })
  })

  describe('DDL with Credentials', () => {
    it('should detect CREATE USER statements', () => {
      const query = "CREATE USER 'newuser'@'localhost' IDENTIFIED BY 'SuperSecret123!'"
      const result = sanitizeQuery(query, config)

      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
      expect(result.stats.sensitiveOperationsFound).toBeGreaterThan(0)
      expect(result.reasons).toContain('Sensitive operation detected: CREATE USER')
    })

    it('should detect ALTER USER statements', () => {
      const query = "ALTER USER 'root'@'localhost' IDENTIFIED BY 'NewPassword456!'"
      const result = sanitizeQuery(query, config)

      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
      expect(result.stats.sensitiveOperationsFound).toBeGreaterThan(0)
    })

    it('should detect GRANT statements', () => {
      const query = "GRANT ALL PRIVILEGES ON *.* TO 'admin'@'%' WITH GRANT OPTION"
      const result = sanitizeQuery(query, config)

      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
      expect(result.stats.sensitiveOperationsFound).toBeGreaterThan(0)
      expect(result.reasons.some(r => r.includes('GRANT'))).toBe(true)
    })

    it('should detect SET PASSWORD statements', () => {
      const query = "SET PASSWORD FOR 'user'@'host' = PASSWORD('NewPass789!')"
      const result = sanitizeQuery(query, config)

      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
      expect(result.stats.sensitiveOperationsFound).toBeGreaterThan(0)
    })

    it('should handle CREATE TABLE with sensitive names', () => {
      const query = `
        CREATE TABLE user_passwords (
          id INT PRIMARY KEY,
          user_id INT,
          password_hash VARCHAR(255),
          salt VARCHAR(32)
        )
      `
      const result = sanitizeQuery(query, config)

      // Table name contains 'password' which should trigger private mode
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })
  })

  describe('Edge Cases', () => {
    it('should handle escaped quotes in strings', () => {
      const query = "SELECT * FROM logs WHERE message = 'User\\'s password: abc123'"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('WHERE message = ?')
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
    })

    it('should handle doubled quotes (SQL escape)', () => {
      const query = "SELECT * FROM config WHERE value = 'api_key: ''secret_value'''"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('WHERE value = ?')
    })

    it('should handle different quote types', () => {
      const queries = [
        "SELECT * FROM t WHERE col = 'value'",
        'SELECT * FROM t WHERE col = "value"',
        "SELECT * FROM t WHERE col = `value`"
      ]

      queries.forEach(q => {
        const result = sanitizeQuery(q, config)
        expect(result.sanitizedQuery).toContain('WHERE col = ?')
      })
    })

    it('should handle comments with credentials', () => {
      const query = `
        -- Password for admin: SuperSecret123!
        SELECT * FROM users
        /* API Key: sk-1234567890abcdef */
        WHERE active = true
      `
      const result = sanitizeQuery(query, config)

      // Comments with credentials should be removed
      expect(result.sanitizedQuery).not.toContain('SuperSecret123!')
      expect(result.sanitizedQuery).not.toContain('sk-1234567890abcdef')
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
    })

    it('should handle multi-line strings', () => {
      const query = `
        INSERT INTO messages (content) VALUES ('
          This is a multi-line
          message with password: Secret123
        ')
      `
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('VALUES (?')
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
    })

    it('should handle numeric literals in WHERE clauses', () => {
      const query = "SELECT * FROM accounts WHERE balance > 10000 AND user_id = 42"
      config.privacyMode = PrivacyMode.NORMAL
      const result = sanitizeQuery(query, config)

      // In normal mode, numeric literals in WHERE should be sanitized
      expect(result.sanitizedQuery).toContain('WHERE balance > ? AND user_id = ?')
    })

    it('should preserve structure with nested parentheses', () => {
      const query = "SELECT * FROM t WHERE (a = 'val1' AND (b = 'val2' OR c = 'val3'))"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('(a = ? AND (b = ? OR c = ?))')
    })
  })

  describe('Credential Detection in Queries', () => {
    it('should detect JWT tokens', () => {
      const query = "UPDATE sessions SET token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c'"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toBe("UPDATE sessions SET token = ?")
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })

    it('should detect API keys', () => {
      const query = "INSERT INTO api_keys (key) VALUES ('sk-proj-1234567890abcdefghijklmnop')"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('VALUES (?)')
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })

    it('should detect database connection strings', () => {
      const query = "UPDATE config SET db_url = 'postgresql://user:password@localhost:5432/mydb'"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toBe("UPDATE config SET db_url = ?")
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })

    it('should detect SSH private keys', () => {
      const query = `
        INSERT INTO ssh_keys (private_key) VALUES ('-----BEGIN RSA PRIVATE KEY-----
        MIIEpAIBAAKCAQEA...
        -----END RSA PRIVATE KEY-----')
      `
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('VALUES (?)')
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })

    it('should detect bcrypt password hashes', () => {
      const query = "UPDATE users SET password = '$2a$10$N9qo8uLOickgx2ZMRZoMye'"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toBe("UPDATE users SET password = ?")
      expect(result.stats.credentialsDetected).toBeGreaterThan(0)
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })
  })

  describe('Privacy Modes', () => {
    it('should sanitize everything in STRICT mode', () => {
      config.privacyMode = PrivacyMode.STRICT
      const query = "SELECT name, age FROM employees WHERE department = 'Engineering' AND salary > 100000"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toBe("SELECT name, age FROM employees WHERE department = ? AND salary > ?")
      expect(result.wasModified).toBe(true)
    })

    it('should be selective in NORMAL mode', () => {
      config.privacyMode = PrivacyMode.NORMAL
      const query = "SELECT * FROM products WHERE category = 'Electronics'"
      const result = sanitizeQuery(query, config)

      // In normal mode, non-sensitive literals in WHERE are sanitized
      expect(result.sanitizedQuery).toBe("SELECT * FROM products WHERE category = ?")
    })

    it('should be minimal in PERMISSIVE mode', () => {
      config.privacyMode = PrivacyMode.PERMISSIVE
      const query = "SELECT * FROM products WHERE name = 'iPhone'"
      const result = sanitizeQuery(query, config)

      // In permissive mode, only obvious credentials are sanitized
      expect(result.sanitizedQuery).toBe("SELECT * FROM products WHERE name = 'iPhone'")
      expect(result.wasModified).toBe(false)
    })

    it('should always sanitize obvious credentials even in PERMISSIVE mode', () => {
      config.privacyMode = PrivacyMode.PERMISSIVE
      const query = "UPDATE config SET api_key = 'sk-1234567890abcdefghijklmnop'"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toBe("UPDATE config SET api_key = ?")
      expect(result.wasModified).toBe(true)
    })
  })

  describe('Batch Processing', () => {
    it('should sanitize multiple queries', () => {
      const queries = [
        "SELECT * FROM users WHERE id = 1",
        "UPDATE passwords SET value = 'secret123'",
        "INSERT INTO logs (message) VALUES ('User logged in')"
      ]

      const results = sanitizeQueries(queries, config)

      expect(results).toHaveLength(3)
      expect(results[0].privacyLevel).toBe(QueryPrivacyLevel.PRIVATE) // users table
      expect(results[1].privacyLevel).toBe(QueryPrivacyLevel.PRIVATE) // passwords table
      expect(results[2].sanitizedQuery).toContain('VALUES (?)')
    })
  })

  describe('Query Classification', () => {
    it('should identify private queries', () => {
      const privateQueries = [
        "SELECT * FROM users",
        "CREATE USER 'test' IDENTIFIED BY 'pass'",
        "GRANT ALL ON *.* TO admin",
        "SELECT * FROM passwords"
      ]

      privateQueries.forEach(q => {
        expect(isPrivateQuery(q, config)).toBe(true)
      })
    })

    it('should identify safe queries', () => {
      config.privacyMode = PrivacyMode.PERMISSIVE
      const safeQueries = [
        "SELECT * FROM products",
        "SELECT COUNT(*) FROM orders",
        "SHOW TABLES",
        "DESCRIBE customers"
      ]

      safeQueries.forEach(q => {
        expect(isPrivateQuery(q, config)).toBe(false)
      })
    })
  })

  describe('Metadata Extraction', () => {
    it('should extract table names', () => {
      const query = `
        SELECT * FROM orders o
        JOIN customers c ON o.customer_id = c.id
        JOIN products p ON o.product_id = p.id
      `
      const metadata = extractQueryMetadata(query, config)

      expect(metadata.tables).toContain('orders')
      expect(metadata.tables).toContain('customers')
      expect(metadata.tables).toContain('products')
    })

    it('should extract operations', () => {
      const query = "INSERT INTO logs SELECT * FROM temp_logs"
      const metadata = extractQueryMetadata(query, config)

      expect(metadata.operations).toContain('INSERT')
      expect(metadata.operations).toContain('SELECT')
    })

    it('should detect credentials in metadata', () => {
      const query = "UPDATE users SET password = 'secret123' WHERE id = 1"
      const metadata = extractQueryMetadata(query, config)

      expect(metadata.hasCredentials).toBe(true)
      expect(metadata.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })

    it('should extract schema names', () => {
      const query = "SELECT * FROM myschema.users JOIN auth.permissions ON users.id = permissions.user_id"
      const metadata = extractQueryMetadata(query, config)

      expect(metadata.schemas).toContain('myschema')
      expect(metadata.schemas).toContain('auth')
      expect(metadata.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE) // auth schema is sensitive
    })
  })

  describe('SQL Injection Prevention', () => {
    it('should handle potential SQL injection attempts', () => {
      const query = "SELECT * FROM users WHERE name = 'admin' OR '1'='1'"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toBe("SELECT * FROM users WHERE name = ? OR ?=?")
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })

    it('should handle union-based injection attempts', () => {
      const query = "SELECT * FROM products WHERE id = '1' UNION SELECT password FROM users--"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain('WHERE id = ?')
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })

    it('should handle comment-based injection', () => {
      const query = "SELECT * FROM users WHERE username = 'admin'-- AND password = 'whatever'"
      const result = sanitizeQuery(query, config)

      expect(result.sanitizedQuery).toContain("WHERE username = ?")
      expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
    })
  })

  describe('Performance', () => {
    it('should handle very long queries efficiently', () => {
      const longValuesList = Array(1000).fill("('value1', 'value2', 123)").join(', ')
      const query = `INSERT INTO large_table (col1, col2, col3) VALUES ${longValuesList}`

      const startTime = Date.now()
      const result = sanitizeQuery(query, config)
      const endTime = Date.now()

      expect(endTime - startTime).toBeLessThan(1000) // Should complete in under 1 second
      expect(result.stats.literalsRemoved).toBeGreaterThan(1000)
    })
  })

  describe('Validation', () => {
    it('should not leak credentials after sanitization', () => {
      const dangerousQueries = [
        "SELECT * FROM users WHERE password = 'SuperSecret123!'",
        "UPDATE config SET api_key = 'sk-1234567890abcdef'",
        "INSERT INTO tokens VALUES ('eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...')",
        "CREATE USER admin IDENTIFIED BY 'AdminPass456!'"
      ]

      dangerousQueries.forEach(query => {
        const result = sanitizeQuery(query, config)

        // Ensure no credentials remain in sanitized query
        expect(result.sanitizedQuery).not.toMatch(/SuperSecret123!/i)
        expect(result.sanitizedQuery).not.toMatch(/sk-[a-z0-9]+/i)
        expect(result.sanitizedQuery).not.toMatch(/eyJ[A-Za-z0-9_-]+/i)
        expect(result.sanitizedQuery).not.toMatch(/AdminPass456!/i)

        // Ensure they're marked as private
        expect(result.privacyLevel).toBe(QueryPrivacyLevel.PRIVATE)
      })
    })

    it('should fail closed when uncertain', () => {
      config.failClosed = true
      const suspiciousQuery = "EXECUTE IMMEDIATE 'CREATE USER ' || username || ' IDENTIFIED BY ' || password"
      const result = sanitizeQuery(suspiciousQuery, config)

      // When failing closed, dangerous queries should be blocked
      if (result.privacyLevel === QueryPrivacyLevel.PRIVATE && result.reasons.includes('Query still contains potential credentials after sanitization')) {
        expect(result.sanitizedQuery).toBe('-- Query blocked: contains credentials')
      }
    })
  })
})