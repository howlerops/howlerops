/**
 * Credential Detection Tests
 *
 * Tests for detecting various types of credentials and sensitive data.
 * CRITICAL: These tests must ensure ZERO false negatives for security.
 */

import { describe, expect,it } from 'vitest'

import { createDefaultConfig } from '../config'
import {
  CredentialType,
  deepScanForCredentials,
  detectCredentials,
  detectCredentialsInBatch,
  mightBeCredential,
  redactCredentials} from '../credential-detector'

describe('Credential Detector', () => {
  const config = createDefaultConfig()

  describe('Password Detection', () => {
    it('should detect complex passwords', () => {
      const passwords = [
        'SuperSecret123!',
        'P@ssw0rd2024',
        'Admin$ecure#99',
        'MyP@55w0rd!',
        'C0mpl3x!Pass'
      ]

      passwords.forEach(pwd => {
        const result = detectCredentials(pwd, config)
        expect(result.isCredential).toBe(true)
        expect([CredentialType.PASSWORD, CredentialType.UNKNOWN]).toContain(result.type)
        expect(result.confidence).toBeGreaterThan(0.5)
      })
    })

    it('should detect simple passwords with context', () => {
      const result = detectCredentials('password: admin123', config)
      expect(result.isCredential).toBe(true)
      expect(result.confidence).toBeGreaterThan(0.5)
    })

    it('should not flag common words as passwords', () => {
      const words = ['hello', 'world', 'test', 'example', 'data']
      words.forEach(word => {
        const result = detectCredentials(word, config)
        expect(result.isCredential).toBe(false)
      })
    })
  })

  describe('API Key Detection', () => {
    it('should detect OpenAI API keys', () => {
      const keys = [
        'sk-proj-1234567890abcdefghijklmnopqrstuvwxyz',
        'sk-1234567890abcdefghijklmnopqrstuvwxyz1234567890'
      ]

      keys.forEach(key => {
        const result = detectCredentials(key, config)
        expect(result.isCredential).toBe(true)
        expect(result.type).toBe(CredentialType.API_KEY)
        expect(result.confidence).toBeGreaterThan(0.8)
      })
    })

    it('should detect AWS access keys', () => {
      const result = detectCredentials('AKIAIOSFODNN7EXAMPLE', config)
      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.API_KEY)
    })

    it('should detect generic API keys', () => {
      const keys = [
        'api-1234567890abcdefghijklmnopqrstuvwxyz',
        'key-abcdef1234567890abcdef1234567890',
        'token-xyz123abc456def789ghi012jkl345'
      ]

      keys.forEach(key => {
        const result = detectCredentials(key, config)
        expect(result.isCredential).toBe(true)
        expect(result.type).toBe(CredentialType.API_KEY)
      })
    })

    it('should detect Bearer tokens', () => {
      const result = detectCredentials('Bearer 1234567890abcdefghijklmnopqrstuvwxyz', config)
      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.API_KEY)
    })
  })

  describe('JWT Token Detection', () => {
    it('should detect valid JWT tokens', () => {
      const jwt = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c'
      const result = detectCredentials(jwt, config)

      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.JWT_TOKEN)
      expect(result.confidence).toBe(1.0)
    })

    it('should detect JWT with Bearer prefix', () => {
      const jwt = 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U'
      const result = detectCredentials(jwt, config)

      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.JWT_TOKEN)
    })
  })

  describe('SSH Private Key Detection', () => {
    it('should detect RSA private keys', () => {
      const key = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----`
      const result = detectCredentials(key, config)

      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.SSH_PRIVATE_KEY)
      expect(result.confidence).toBe(1.0)
    })

    it('should detect OpenSSH private keys', () => {
      const key = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
-----END OPENSSH PRIVATE KEY-----`
      const result = detectCredentials(key, config)

      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.SSH_PRIVATE_KEY)
    })

    it('should detect encrypted private keys', () => {
      const key = '-----BEGIN ENCRYPTED PRIVATE KEY-----\nMIIE...\n-----END ENCRYPTED PRIVATE KEY-----'
      const result = detectCredentials(key, config)

      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.SSH_PRIVATE_KEY)
    })
  })

  describe('Connection String Detection', () => {
    it('should detect PostgreSQL connection strings', () => {
      const connStrings = [
        'postgresql://user:password@localhost:5432/mydb',
        'postgres://admin:secret123@db.example.com/production'
      ]

      connStrings.forEach(str => {
        const result = detectCredentials(str, config)
        expect(result.isCredential).toBe(true)
        expect(result.type).toBe(CredentialType.CONNECTION_STRING)
        expect(result.confidence).toBe(1.0)
      })
    })

    it('should detect MongoDB connection strings', () => {
      const connStrings = [
        'mongodb://user:pass@localhost:27017/mydb',
        'mongodb+srv://admin:secret@cluster.mongodb.net/test'
      ]

      connStrings.forEach(str => {
        const result = detectCredentials(str, config)
        expect(result.isCredential).toBe(true)
        expect(result.type).toBe(CredentialType.CONNECTION_STRING)
      })
    })

    it('should detect SQL Server connection strings', () => {
      const result = detectCredentials('Data Source=server;User ID=sa;Password=MyPass123', config)
      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.CONNECTION_STRING)
    })
  })

  describe('Hash Detection', () => {
    it('should detect MD5 hashes', () => {
      const result = detectCredentials('5d41402abc4b2a76b9719d911017c592', config)
      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.HASH)
      expect(result.reason).toContain('MD5')
    })

    it('should detect SHA256 hashes', () => {
      const result = detectCredentials('e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', config)
      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.HASH)
      expect(result.reason).toContain('SHA256')
    })

    it('should detect bcrypt hashes', () => {
      const result = detectCredentials('$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', config)
      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.HASH)
      expect(result.reason).toContain('bcrypt')
    })
  })

  describe('Certificate Detection', () => {
    it('should detect certificates', () => {
      const cert = `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKLdQVPy90WjMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
-----END CERTIFICATE-----`
      const result = detectCredentials(cert, config)

      expect(result.isCredential).toBe(true)
      expect(result.type).toBe(CredentialType.CERTIFICATE)
    })
  })

  describe('High Entropy Detection', () => {
    it('should detect high-entropy strings', () => {
      const highEntropyStrings = [
        'aB3dE7gH1jK4mN6pQ9rS2tU5vW8xY0z',
        'X9y8Z7w6V5u4T3s2R1q0P9o8N7m6L5k',
        'f4G7j9K2m5N8p1Q4r7S0t3U6v9W2x5Y'
      ]

      highEntropyStrings.forEach(str => {
        const result = detectCredentials(str, config)
        expect(result.isCredential).toBe(true)
        expect(result.confidence).toBeGreaterThan(0.5)
      })
    })

    it('should not flag UUIDs as credentials', () => {
      const uuids = [
        '550e8400-e29b-41d4-a716-446655440000',
        '6ba7b810-9dad-11d1-80b4-00c04fd430c8'
      ]

      uuids.forEach(uuid => {
        const result = detectCredentials(uuid, config)
        expect(result.isCredential).toBe(false)
      })
    })

    it('should not flag dates as credentials', () => {
      const dates = [
        '2024-01-15',
        '2024-01-15T10:30:00Z',
        '15/01/2024'
      ]

      dates.forEach(date => {
        const result = detectCredentials(date, config)
        expect(result.isCredential).toBe(false)
      })
    })
  })

  describe('Batch Detection', () => {
    it('should detect credentials in batch', () => {
      const inputs = [
        'normal text',
        'sk-1234567890abcdef',
        'password123',
        'hello world',
        'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U'
      ]

      const results = detectCredentialsInBatch(inputs, config)

      expect(results[0].isCredential).toBe(false)
      expect(results[1].isCredential).toBe(true)
      expect(results[1].type).toBe(CredentialType.API_KEY)
      expect(results[2].isCredential).toBe(false) // Simple password without complexity
      expect(results[3].isCredential).toBe(false)
      expect(results[4].isCredential).toBe(true)
      expect(results[4].type).toBe(CredentialType.JWT_TOKEN)
    })
  })

  describe('Deep Scanning', () => {
    it('should deep scan objects for credentials', () => {
      const obj = {
        name: 'John Doe',
        email: 'john@example.com',
        config: {
          apiKey: 'sk-1234567890abcdefghijklmnop',
          database: {
            host: 'localhost',
            password: 'SuperSecret123!',
            connectionString: 'postgresql://user:pass@localhost/db'
          }
        },
        tokens: [
          'normal-token',
          'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U'
        ]
      }

      const results = deepScanForCredentials(obj, config)

      expect(results.length).toBeGreaterThan(0)
      expect(results.some(r => r.path.includes('apiKey'))).toBe(true)
      expect(results.some(r => r.path.includes('password'))).toBe(true)
      expect(results.some(r => r.path.includes('connectionString'))).toBe(true)
      expect(results.some(r => r.path.includes('tokens[1]'))).toBe(true)
    })

    it('should handle circular references', () => {
      const obj: any = { name: 'test' }
      obj.circular = obj // Create circular reference

      expect(() => deepScanForCredentials(obj, config)).not.toThrow()
    })

    it('should detect credentials in sensitive field names', () => {
      const obj = {
        password: '',  // Even empty password fields should be flagged
        api_key: null,
        secret_token: undefined,
        auth: 'basic-auth-string'
      }

      const results = deepScanForCredentials(obj, config)
      // Should flag auth field even though the value might not look like a credential
      expect(results.some(r => r.path.includes('auth'))).toBe(true)
    })
  })

  describe('Quick Checks', () => {
    it('should quickly identify potential credentials', () => {
      expect(mightBeCredential('sk-1234567890abcdef')).toBe(true)
      expect(mightBeCredential('eyJhbGciOiJIUzI1NiI')).toBe(true)
      expect(mightBeCredential('-----BEGIN RSA')).toBe(true)
      expect(mightBeCredential('hello world')).toBe(false)
      expect(mightBeCredential('test')).toBe(false)
    })
  })

  describe('Redaction', () => {
    it('should redact credentials in strings', () => {
      const input = 'My API key is sk-1234567890abcdef and password is SuperSecret123!'
      const { redacted, detections } = redactCredentials(input, config)

      expect(redacted).not.toContain('sk-1234567890abcdef')
      expect(redacted).not.toContain('SuperSecret123!')
      expect(detections.length).toBeGreaterThan(0)
    })

    it('should preserve structure when redacting', () => {
      const input = 'token=eyJhbGciOiJIUzI1NiI.payload.signature'
      const { redacted } = redactCredentials(input, config)

      expect(redacted).toContain('token=')
      expect(redacted).not.toContain('eyJhbGciOiJIUzI1NiI')
    })

    it('should handle multiple credentials', () => {
      const input = 'user:pass@host, api_key=abc123def456, token=xyz789'
      const { redacted, detections } = redactCredentials(input, config)

      expect(detections.length).toBeGreaterThanOrEqual(1)
      expect(redacted).not.toContain('abc123def456')
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty inputs', () => {
      const result = detectCredentials('', config)
      expect(result.isCredential).toBe(false)
    })

    it('should handle null/undefined gracefully', () => {
      const result = detectCredentials(null as any, config)
      expect(result.isCredential).toBe(false)
    })

    it('should handle very long strings', () => {
      const longString = 'a'.repeat(1000)
      const result = detectCredentials(longString, config)
      expect(result.isCredential).toBe(true) // Exceeds safe length
      expect(result.reason).toContain('exceeds safe length')
    })

    it('should detect environment variable references', () => {
      const envVars = [
        '${DB_PASSWORD}',
        '$SECRET_KEY',
        '${API_TOKEN}',
        '${AUTH_TOKEN}'
      ]

      envVars.forEach(env => {
        const result = detectCredentials(env, config)
        expect(result.isCredential).toBe(true)
        expect(result.reason).toContain('Environment variable')
      })
    })
  })

  describe('False Positive Prevention', () => {
    it('should not flag common non-credentials', () => {
      const nonCredentials = [
        'localhost',
        '127.0.0.1',
        'example.com',
        'john.doe@example.com',
        'user123',
        'test-data',
        'sample-text',
        'hello-world-123'
      ]

      nonCredentials.forEach(text => {
        const result = detectCredentials(text, config)
        expect(result.isCredential).toBe(false)
      })
    })

    it('should not flag version numbers', () => {
      const versions = ['v1.2.3', '2.0.0', 'v2.1.0-beta']
      versions.forEach(ver => {
        const result = detectCredentials(ver, config)
        expect(result.isCredential).toBe(false)
      })
    })

    it('should not flag order/reference numbers', () => {
      const refs = ['ORD-12345', 'REF-2024-001', 'INV-9876']
      refs.forEach(ref => {
        const result = detectCredentials(ref, config)
        expect(result.isCredential).toBe(false)
      })
    })
  })
})