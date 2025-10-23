/**
 * Connection Sanitization Tests
 *
 * Tests for stripping credentials from database connection objects.
 * CRITICAL: Ensures no passwords or private keys are ever synced to cloud.
 */

import { describe, it, expect } from 'vitest'
import {
  sanitizeConnection,
  sanitizeConnections,
  prepareConnectionsForSync,
  hasRequiredCredentials,
  restoreCredentials,
  validateSanitization,
  type SanitizedConnection
} from '../connection-sanitizer'
import type { DatabaseConnection } from '@/store/connection-store'
import { createDefaultConfig } from '../config'

describe('Connection Sanitizer', () => {
  const config = createDefaultConfig()

  // Helper to create a test connection
  const createTestConnection = (overrides?: Partial<DatabaseConnection>): DatabaseConnection => ({
    id: 'test-id',
    name: 'Test Connection',
    type: 'postgresql',
    host: 'localhost',
    port: 5432,
    database: 'testdb',
    username: 'testuser',
    password: 'TestPassword123!',
    isConnected: false,
    ...overrides
  })

  describe('Basic Sanitization', () => {
    it('should remove password from connection', () => {
      const connection = createTestConnection()
      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.password).toBeUndefined()
      expect(result.sanitizedConnection.passwordRequired).toBe(true)
      expect(result.wasModified).toBe(true)
      expect(result.removedFields).toContain('password')
    })

    it('should preserve non-sensitive fields', () => {
      const connection = createTestConnection()
      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.id).toBe(connection.id)
      expect(result.sanitizedConnection.name).toBe(connection.name)
      expect(result.sanitizedConnection.type).toBe(connection.type)
      expect(result.sanitizedConnection.host).toBe(connection.host)
      expect(result.sanitizedConnection.port).toBe(connection.port)
      expect(result.sanitizedConnection.database).toBe(connection.database)
      expect(result.sanitizedConnection.username).toBe(connection.username)
    })

    it('should handle connections without passwords', () => {
      const connection = createTestConnection({ password: undefined })
      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.passwordRequired).toBe(false)
      expect(result.wasModified).toBe(false)
      expect(result.removedFields).toHaveLength(0)
    })

    it('should add sanitization metadata', () => {
      const connection = createTestConnection()
      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.sanitizedAt).toBeInstanceOf(Date)
      expect(result.sanitizedConnection.sanitizationHash).toBeDefined()
      expect(result.sanitizedConnection.sanitizationFlags).toBeDefined()
      expect(result.sanitizedConnection.sanitizationFlags.passwordRemoved).toBe(true)
    })
  })

  describe('SSH Tunnel Sanitization', () => {
    it('should remove SSH password', () => {
      const connection = createTestConnection({
        useTunnel: true,
        sshTunnel: {
          host: 'ssh.example.com',
          port: 22,
          user: 'sshuser',
          authMethod: 'Password',
          password: 'SSHPassword123!',
          strictHostKeyChecking: false,
          timeoutSeconds: 30,
          keepAliveIntervalSeconds: 10
        }
      })

      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.sshTunnel?.password).toBeUndefined()
      expect(result.sanitizedConnection.sshTunnel?.passwordRequired).toBe(true)
      expect(result.removedFields).toContain('sshTunnel.password')
      expect(result.sanitizedConnection.sanitizationFlags.sshPasswordRemoved).toBe(true)
    })

    it('should remove SSH private key content but keep path', () => {
      const connection = createTestConnection({
        useTunnel: true,
        sshTunnel: {
          host: 'ssh.example.com',
          port: 22,
          user: 'sshuser',
          authMethod: 'PrivateKey',
          privateKey: '-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----',
          privateKeyPath: '/home/user/.ssh/id_rsa',
          strictHostKeyChecking: false,
          timeoutSeconds: 30,
          keepAliveIntervalSeconds: 10
        }
      })

      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.sshTunnel?.privateKey).toBeUndefined()
      expect(result.sanitizedConnection.sshTunnel?.privateKeyPath).toBe('/home/user/.ssh/id_rsa')
      expect(result.sanitizedConnection.sshTunnel?.privateKeyRequired).toBe(true)
      expect(result.removedFields).toContain('sshTunnel.privateKey')
      expect(result.sanitizedConnection.sanitizationFlags.sshPrivateKeyRemoved).toBe(true)
    })

    it('should preserve other SSH config', () => {
      const connection = createTestConnection({
        useTunnel: true,
        sshTunnel: {
          host: 'ssh.example.com',
          port: 2222,
          user: 'sshuser',
          authMethod: 'Password',
          password: 'secret',
          knownHostsPath: '/home/user/.ssh/known_hosts',
          strictHostKeyChecking: true,
          timeoutSeconds: 60,
          keepAliveIntervalSeconds: 15
        }
      })

      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.sshTunnel?.host).toBe('ssh.example.com')
      expect(result.sanitizedConnection.sshTunnel?.port).toBe(2222)
      expect(result.sanitizedConnection.sshTunnel?.user).toBe('sshuser')
      expect(result.sanitizedConnection.sshTunnel?.knownHostsPath).toBe('/home/user/.ssh/known_hosts')
      expect(result.sanitizedConnection.sshTunnel?.strictHostKeyChecking).toBe(true)
    })
  })

  describe('Parameter Sanitization', () => {
    it('should remove parameters with credential-like names', () => {
      const connection = createTestConnection({
        parameters: {
          sslmode: 'require',
          password: 'HiddenPassword',
          api_key: 'secret-key-123',
          authToken: 'bearer-xyz',
          normalParam: 'value'
        }
      })

      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.parameters?.password).toBeUndefined()
      expect(result.sanitizedConnection.parameters?.api_key).toBeUndefined()
      expect(result.sanitizedConnection.parameters?.authToken).toBeUndefined()
      expect(result.sanitizedConnection.parameters?.normalParam).toBe('value')
      expect(result.sanitizedConnection.parameters?.sslmode).toBe('require')
      expect(result.removedFields).toContain('parameters.password')
      expect(result.removedFields).toContain('parameters.api_key')
    })

    it('should detect credentials in parameter values', () => {
      const connection = createTestConnection({
        parameters: {
          customHeader: 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U',
          connectionString: 'postgresql://user:pass@localhost/db',
          normalParam: 'safe-value'
        }
      })

      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.parameters?.customHeader).toBeUndefined()
      expect(result.sanitizedConnection.parameters?.connectionString).toBeUndefined()
      expect(result.sanitizedConnection.parameters?.normalParam).toBe('safe-value')
      expect(result.unexpectedCredentials.length).toBeGreaterThan(0)
    })
  })

  describe('VPC Config Sanitization', () => {
    it('should sanitize VPC custom config', () => {
      const connection = createTestConnection({
        useVpc: true,
        vpcConfig: {
          vpcId: 'vpc-123456',
          subnetId: 'subnet-789012',
          securityGroupIds: ['sg-abc123'],
          customConfig: {
            endpoint: 'https://api.example.com',
            apiKey: 'sk-1234567890abcdef',
            region: 'us-west-2'
          }
        }
      })

      const result = sanitizeConnection(connection, config)

      expect(result.sanitizedConnection.vpcConfig?.customConfig?.apiKey).toBeUndefined()
      expect(result.sanitizedConnection.vpcConfig?.customConfig?.region).toBe('us-west-2')
      expect(result.sanitizedConnection.vpcConfig?.vpcId).toBe('vpc-123456')
      expect(result.unexpectedCredentials.some(c => c.field.includes('apiKey'))).toBe(true)
    })
  })

  describe('Unexpected Credential Detection', () => {
    it('should detect credentials in unexpected fields', () => {
      const connection = createTestConnection({
        name: 'sk-1234567890abcdef',  // API key as connection name
        database: 'mydb_SuperSecret123',  // Password in database name
        host: 'postgresql://user:pass@localhost'  // Connection string as host
      })

      const result = sanitizeConnection(connection, config)

      expect(result.unexpectedCredentials.length).toBeGreaterThan(0)
      expect(result.isSafeToSync).toBe(false)
      expect(result.safetyIssues.length).toBeGreaterThan(0)
    })

    it('should redact high-confidence credentials in unexpected places', () => {
      const connection = createTestConnection({
        name: 'My Connection with key sk-1234567890abcdefghijklmnop'
      })

      const result = sanitizeConnection(connection, config)

      // High confidence credentials should be redacted
      if (result.unexpectedCredentials.some(c => c.confidence > 0.8)) {
        expect(result.sanitizedConnection.name).toContain('[REDACTED]')
      }
    })
  })

  describe('Batch Processing', () => {
    it('should sanitize multiple connections', () => {
      const connections = [
        createTestConnection({ id: '1', password: 'pass1' }),
        createTestConnection({ id: '2', password: 'pass2' }),
        createTestConnection({ id: '3', password: undefined })
      ]

      const results = sanitizeConnections(connections, config)

      expect(results).toHaveLength(3)
      expect(results[0].wasModified).toBe(true)
      expect(results[1].wasModified).toBe(true)
      expect(results[2].wasModified).toBe(false)
    })
  })

  describe('Sync Preparation', () => {
    it('should separate safe and unsafe connections', () => {
      const connections = [
        createTestConnection({ id: '1', name: 'Safe Connection' }),
        createTestConnection({
          id: '2',
          name: 'Connection with API key sk-1234567890abcdef'
        }),
        createTestConnection({ id: '3', name: 'Normal Connection' })
      ]

      const { safeConnections, unsafeConnections } = prepareConnectionsForSync(connections, config)

      // The connection with API key in name should be unsafe
      expect(unsafeConnections.length).toBeGreaterThanOrEqual(1)
      expect(unsafeConnections.some(u => u.connection.id === '2')).toBe(true)
    })

    it('should sanitize all safe connections', () => {
      const connections = [
        createTestConnection({ id: '1', password: 'pass1' }),
        createTestConnection({ id: '2', password: 'pass2' })
      ]

      const { safeConnections } = prepareConnectionsForSync(connections, config)

      safeConnections.forEach(conn => {
        expect(conn.password).toBeUndefined()
        expect(conn.passwordRequired).toBeDefined()
        expect(conn.sanitizedAt).toBeInstanceOf(Date)
      })
    })
  })

  describe('Credential Requirements', () => {
    it('should detect missing credentials in sanitized connections', () => {
      const connection = createTestConnection()
      const result = sanitizeConnection(connection, config)

      expect(hasRequiredCredentials(result.sanitizedConnection)).toBe(false)
    })

    it('should detect present credentials in unsanitized connections', () => {
      const connection = createTestConnection()
      expect(hasRequiredCredentials(connection)).toBe(true)
    })

    it('should handle SQLite connections (no password required)', () => {
      const connection = createTestConnection({
        type: 'sqlite',
        password: undefined
      })

      expect(hasRequiredCredentials(connection)).toBe(true)
    })

    it('should check SSH tunnel credentials', () => {
      const connection = createTestConnection({
        password: 'dbpass',
        useTunnel: true,
        sshTunnel: {
          host: 'ssh.example.com',
          port: 22,
          user: 'sshuser',
          authMethod: 'Password',
          strictHostKeyChecking: false,
          timeoutSeconds: 30,
          keepAliveIntervalSeconds: 10
          // Missing password
        }
      })

      expect(hasRequiredCredentials(connection)).toBe(false)
    })
  })

  describe('Credential Restoration', () => {
    it('should restore credentials to sanitized connection', () => {
      const original = createTestConnection({
        password: 'TestPass123',
        useTunnel: true,
        sshTunnel: {
          host: 'ssh.example.com',
          port: 22,
          user: 'sshuser',
          authMethod: 'Password',
          password: 'SSHPass456',
          strictHostKeyChecking: false,
          timeoutSeconds: 30,
          keepAliveIntervalSeconds: 10
        }
      })

      const sanitized = sanitizeConnection(original, config).sanitizedConnection
      const restored = restoreCredentials(sanitized, {
        password: 'TestPass123',
        sshPassword: 'SSHPass456'
      })

      expect(restored.password).toBe('TestPass123')
      expect(restored.sshTunnel?.password).toBe('SSHPass456')
      expect(restored.passwordRequired).toBeUndefined()
      expect(restored.sanitizationFlags).toBeUndefined()
    })
  })

  describe('Validation', () => {
    it('should validate proper sanitization', () => {
      const connection = createTestConnection()
      const sanitized = sanitizeConnection(connection, config).sanitizedConnection

      const validation = validateSanitization(sanitized, config)
      expect(validation.isValid).toBe(true)
      expect(validation.issues).toHaveLength(0)
    })

    it('should detect incomplete sanitization', () => {
      const connection = createTestConnection({
        password: 'StillHere123!',
        sshTunnel: {
          host: 'ssh.example.com',
          port: 22,
          user: 'sshuser',
          authMethod: 'Password',
          password: 'AlsoStillHere456!',
          strictHostKeyChecking: false,
          timeoutSeconds: 30,
          keepAliveIntervalSeconds: 10
        }
      })

      const validation = validateSanitization(connection, config)
      expect(validation.isValid).toBe(false)
      expect(validation.issues).toContain('Password field still present')
      expect(validation.issues).toContain('SSH password still present')
    })

    it('should detect credentials in unexpected places', () => {
      const partialSanitized = {
        id: 'test',
        name: 'Test',
        type: 'postgresql',
        database: 'db',
        customField: 'sk-1234567890abcdefghijklmnop'  // API key in custom field
      }

      const validation = validateSanitization(partialSanitized, config)
      expect(validation.isValid).toBe(false)
      expect(validation.issues.some(i => i.includes('Credential found'))).toBe(true)
    })
  })

  describe('Edge Cases', () => {
    it('should handle null/undefined connections gracefully', () => {
      expect(() => sanitizeConnection(null as any, config)).not.toThrow()
      expect(() => sanitizeConnection(undefined as any, config)).not.toThrow()
    })

    it('should handle deeply nested credentials', () => {
      const connection = createTestConnection({
        parameters: {
          nested: JSON.stringify({
            deep: {
              secret: 'sk-1234567890abcdef'
            }
          })
        }
      })

      const result = sanitizeConnection(connection, config)
      expect(result.unexpectedCredentials.length).toBeGreaterThan(0)
    })

    it('should preserve connection functionality after sanitization', () => {
      const connection = createTestConnection({
        id: 'unique-id',
        sessionId: 'session-123',
        isConnected: true,
        lastUsed: new Date()
      })

      const result = sanitizeConnection(connection, config)

      // These fields should be preserved for functionality
      expect(result.sanitizedConnection.id).toBe('unique-id')
      expect(result.sanitizedConnection.sessionId).toBe('session-123')
      expect(result.sanitizedConnection.isConnected).toBe(true)
    })
  })

  describe('Security Guarantees', () => {
    it('should NEVER leave passwords in sanitized output', () => {
      const passwords = [
        'SimplePass',
        'Complex!Pass123',
        'SuperSecret@2024',
        'Admin$ecure#99'
      ]

      passwords.forEach(pwd => {
        const conn = createTestConnection({ password: pwd })
        const result = sanitizeConnection(conn, config)

        // Check the entire sanitized object as a string
        const sanitizedStr = JSON.stringify(result.sanitizedConnection)
        expect(sanitizedStr).not.toContain(pwd)
      })
    })

    it('should NEVER leave SSH credentials in sanitized output', () => {
      const sshSecrets = {
        password: 'SSHSecret123!',
        privateKey: '-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----'
      }

      const conn = createTestConnection({
        useTunnel: true,
        sshTunnel: {
          host: 'ssh.example.com',
          port: 22,
          user: 'user',
          authMethod: 'Password',
          password: sshSecrets.password,
          privateKey: sshSecrets.privateKey,
          strictHostKeyChecking: false,
          timeoutSeconds: 30,
          keepAliveIntervalSeconds: 10
        }
      })

      const result = sanitizeConnection(conn, config)
      const sanitizedStr = JSON.stringify(result.sanitizedConnection)

      expect(sanitizedStr).not.toContain(sshSecrets.password)
      expect(sanitizedStr).not.toContain(sshSecrets.privateKey)
      expect(sanitizedStr).not.toContain('BEGIN RSA PRIVATE KEY')
    })

    it('should NEVER leave API keys in sanitized output', () => {
      const apiKeys = [
        'sk-1234567890abcdef',
        'api-key-xyz123',
        'token-abcdef123456'
      ]

      apiKeys.forEach(key => {
        const conn = createTestConnection({
          parameters: {
            apiKey: key,
            normalParam: 'safe'
          }
        })

        const result = sanitizeConnection(conn, config)
        const sanitizedStr = JSON.stringify(result.sanitizedConnection)

        expect(sanitizedStr).not.toContain(key)
      })
    })
  })
})