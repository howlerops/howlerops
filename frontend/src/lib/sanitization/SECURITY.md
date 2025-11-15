# Howlerops Data Sanitization Security Module

## Overview
This module provides comprehensive data sanitization to prevent credential leakage when syncing Howlerops data to the cloud. It implements defense-in-depth with ZERO tolerance for false negatives - we'd rather over-sanitize than leak credentials.

## Critical Security Requirements

### NEVER Sync to Cloud
- Database passwords
- SSH tunnel passwords
- SSH private keys
- API keys
- Session tokens
- JWT tokens
- Connection strings with embedded credentials
- Any high-entropy strings that might be credentials

### Safe to Sync (After Sanitization)
- Connection metadata (host, port, database, username)
- Sanitized queries (literals removed)
- UI preferences
- Schema information
- Table structures

## Module Components

### 1. Configuration (`config.ts`)
- **Privacy Modes**:
  - `STRICT`: Maximum privacy, all literals removed
  - `NORMAL`: Balanced approach (default)
  - `PERMISSIVE`: Minimal sanitization (use with caution)

- **Query Privacy Levels**:
  - `PRIVATE`: Never sync, contains sensitive data
  - `NORMAL`: Sync after sanitization
  - `SHARED`: Full sync allowed

- **Configurable Rules**:
  - Excluded keywords (CREATE USER, GRANT, etc.)
  - Excluded tables (users, passwords, tokens, etc.)
  - Excluded schemas (auth, security, admin, etc.)
  - Sensitive column patterns
  - Credential detection patterns

### 2. Credential Detector (`credential-detector.ts`)
Detects various credential types with high accuracy:

- **Password Detection**: Complex passwords, password patterns
- **API Key Detection**: OpenAI, AWS, generic API keys
- **JWT Token Detection**: Standard JWT format
- **SSH Private Keys**: RSA, DSA, EC, OpenSSH formats
- **Connection Strings**: PostgreSQL, MySQL, MongoDB, etc.
- **Certificates**: X.509, PEM formats
- **Hash Detection**: MD5, SHA, bcrypt, Argon2
- **High Entropy Detection**: Random strings likely to be credentials

**Key Features**:
- Shannon entropy calculation for randomness detection
- Pattern matching with regex
- Context-aware detection
- Deep scanning of objects
- Batch processing support

### 3. Query Sanitizer (`query-sanitizer.ts`)
Removes sensitive data from SQL queries while preserving structure:

- **Sanitization Actions**:
  - Replace string literals with `?`
  - Replace numeric literals in WHERE clauses with `?`
  - Remove comments containing credentials
  - Flag queries with sensitive operations

- **Detection Capabilities**:
  - DDL with credentials (CREATE USER, GRANT, etc.)
  - Sensitive table/schema access
  - Embedded credentials in literals
  - SQL injection attempts

- **Tokenization**:
  - Full SQL tokenizer for accurate parsing
  - Preserves query structure
  - Handles all quote types and escaping
  - Supports comments and multi-line strings

### 4. Connection Sanitizer (`connection-sanitizer.ts`)
Strips credentials from database connection objects:

- **Removed Fields**:
  - Main database password
  - SSH tunnel password
  - SSH private key content (keeps path reference)
  - Credentials in parameters
  - Credentials in VPC config

- **Added Metadata**:
  - `passwordRequired` flag
  - `sshPasswordRequired` flag
  - `privateKeyRequired` flag
  - Sanitization timestamp
  - Sanitization hash

- **Safety Validation**:
  - Deep scan for missed credentials
  - Validates complete sanitization
  - Marks unsafe connections

## Usage Examples

### Basic Usage
```typescript
import { Sanitizer } from '@/lib/sanitization'

const sanitizer = new Sanitizer()

// Sanitize a query
const queryResult = sanitizer.sanitizeQuery("SELECT * FROM users WHERE password = 'secret123'")
// Result: "SELECT * FROM users WHERE password = ?"

// Sanitize a connection
const connResult = sanitizer.sanitizeConnection(connection)
// Result: Connection with password removed

// Check for credentials
const credResult = sanitizer.detectCredentials("sk-1234567890abcdef")
// Result: { isCredential: true, type: 'api_key', confidence: 0.9 }
```

### Prepare Data for Sync
```typescript
import { prepareConnectionsForSync, sanitizeQueries } from '@/lib/sanitization'

// Prepare connections
const { safeConnections, unsafeConnections } = prepareConnectionsForSync(connections)

// Sanitize queries
const sanitizedQueries = sanitizeQueries(queries)
  .filter(q => q.privacyLevel !== QueryPrivacyLevel.PRIVATE)
```

### Integration with Stores
```typescript
// In query-store.ts
import { sanitizeQuery } from '@/lib/sanitization'

function saveQuery(query: string) {
  const result = sanitizeQuery(query)
  if (result.privacyLevel === QueryPrivacyLevel.PRIVATE) {
    // Don't sync to cloud
    localStorage.setItem('private-query', query)
  } else {
    // Safe to sync
    syncToCloud(result.sanitizedQuery)
  }
}

// In connection-store.ts
import { sanitizeConnection } from '@/lib/sanitization'

function syncConnection(conn: DatabaseConnection) {
  const result = sanitizeConnection(conn)
  if (result.isSafeToSync) {
    syncToCloud(result.sanitizedConnection)
  } else {
    console.warn('Connection unsafe to sync:', result.safetyIssues)
  }
}
```

## Security Guarantees

### Zero False Negatives
The system is designed to NEVER miss credentials:
- Multiple detection layers
- Context-aware analysis
- High-entropy detection
- Pattern matching
- Fail-closed design

### Defense in Depth
Multiple security layers:
1. Field-level sanitization (remove known credential fields)
2. Pattern detection (regex for credential formats)
3. Entropy analysis (detect random strings)
4. Context analysis (sensitive table/operation detection)
5. Deep scanning (recursive object traversal)
6. Validation layer (verify sanitization completeness)

### Audit Trail
All sanitization operations include:
- Timestamp of sanitization
- Hash of original data
- List of removed fields
- Detected credentials with confidence scores
- Safety assessment

## Testing

Comprehensive test coverage including:
- **Unit Tests**: Each function tested individually
- **Integration Tests**: Component interaction testing
- **Security Tests**: Credential leakage prevention
- **Edge Cases**: SQL injection, escaping, complex queries
- **Performance Tests**: Large query handling

Run tests:
```bash
npm test src/lib/sanitization/__tests__/
```

## Configuration

### Global Configuration
```typescript
import { setGlobalConfig, PrivacyMode } from '@/lib/sanitization'

setGlobalConfig({
  privacyMode: PrivacyMode.STRICT,
  failClosed: true,
  enableLogging: true,
  maxSafeStringLength: 100
})
```

### Custom Rules
```typescript
import { mergeConfig } from '@/lib/sanitization'

const customConfig = mergeConfig({
  excludeTables: new Set(['my_sensitive_table']),
  excludeKeywords: new Set(['MY_CUSTOM_OPERATION']),
  credentialPatterns: {
    apiKey: [/^custom-key-[a-z0-9]+$/]
  }
})
```

## Performance Considerations

- **Tokenization**: Efficient single-pass SQL tokenizer
- **Caching**: Sanitization results can be cached
- **Batch Processing**: Support for bulk operations
- **Lazy Evaluation**: Deep scanning only when needed

## Future Enhancements

1. **Machine Learning**: Train model on credential patterns
2. **Contextual Analysis**: Understand query intent
3. **Reversible Sanitization**: Encrypted storage of removed data
4. **Audit Logging**: Complete sanitization audit trail
5. **Performance Optimization**: Parallel processing for large datasets

## Security Checklist

Before syncing any data:
- [ ] All queries sanitized through `sanitizeQuery()`
- [ ] All connections sanitized through `sanitizeConnection()`
- [ ] Private queries excluded from sync
- [ ] Unsafe connections blocked from sync
- [ ] Deep scan performed on final payload
- [ ] Validation confirms no credentials present
- [ ] Audit log records sanitization operations

## Emergency Procedures

If credentials are suspected to have leaked:
1. Immediately disable sync
2. Rotate all potentially exposed credentials
3. Review sanitization logs
4. Update detection patterns
5. Re-test with leaked credential format
6. Deploy fix and force re-sanitization

## Contact

For security concerns or to report credential leakage:
- File a security issue (private)
- Contact the security team directly
- Use responsible disclosure practices

---

**Remember**: When in doubt, don't sync it out!