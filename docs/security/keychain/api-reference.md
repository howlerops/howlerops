# Keychain API Reference

Quick reference for using the credential management API in SQL Studio.

## Frontend Usage (TypeScript/React)

### Import

```typescript
import * as App from '../../wailsjs/go/main/App'
```

### Store Password

```typescript
// Store a password for a connection
try {
  await App.StorePassword('connection-id-123', 'my-secure-password')
  console.log('Password stored successfully')
} catch (error) {
  console.error('Failed to store password:', error)
}
```

**Parameters:**
- `connectionID` (string): Unique identifier for the database connection
- `password` (string): The password to store securely

**Returns:** `Promise<void>`

**Errors:**
- Empty connectionID or password
- Keychain access denied
- Keychain service unavailable

---

### Retrieve Password

```typescript
// Get a password for a connection
try {
  const password = await App.GetPassword('connection-id-123')
  console.log('Password retrieved successfully')
} catch (error) {
  console.error('Failed to retrieve password:', error)
  // Error could mean password not found or keychain unavailable
}
```

**Parameters:**
- `connectionID` (string): Unique identifier for the database connection

**Returns:** `Promise<string>` - The stored password

**Errors:**
- Empty connectionID
- Password not found in keychain
- Keychain access denied
- Keychain service unavailable

---

### Delete Password

```typescript
// Delete a password from keychain
try {
  await App.DeletePassword('connection-id-123')
  console.log('Password deleted successfully')
} catch (error) {
  console.error('Failed to delete password:', error)
}
```

**Parameters:**
- `connectionID` (string): Unique identifier for the database connection

**Returns:** `Promise<void>`

**Notes:**
- Idempotent operation (safe to call multiple times)
- Does not error if password doesn't exist

**Errors:**
- Empty connectionID
- Keychain access denied
- Keychain service unavailable

---

### Check Password Exists

```typescript
// Check if a password exists in keychain
const exists = await App.HasPassword('connection-id-123')
if (exists) {
  console.log('Password found in keychain')
} else {
  console.log('No password stored for this connection')
}
```

**Parameters:**
- `connectionID` (string): Unique identifier for the database connection

**Returns:** `Promise<boolean>` - true if password exists, false otherwise

**Notes:**
- Returns false for empty connectionID
- Does not throw errors

---

## Backend Usage (Go)

### Access via App Instance

```go
// In app.go or any method with access to *App

// Store password
err := a.credentialService.StorePassword("connection-id-123", "my-password")
if err != nil {
    log.Error("Failed to store password:", err)
}

// Get password
password, err := a.credentialService.GetPassword("connection-id-123")
if err != nil {
    log.Error("Failed to get password:", err)
}

// Delete password
err = a.credentialService.DeletePassword("connection-id-123")
if err != nil {
    log.Error("Failed to delete password:", err)
}

// Check if password exists
exists := a.credentialService.HasPassword("connection-id-123")
```

### Direct Service Usage

```go
import "github.com/sql-studio/sql-studio/services"

// Create service
logger := logrus.New()
credService := services.NewCredentialService(logger)

// Use the service
err := credService.StorePassword("conn-id", "password")
```

---

## Error Handling

### Common Errors

| Error | Description | Solution |
|-------|-------------|----------|
| `credential not found` | Password doesn't exist in keychain | Normal - store a password first |
| `permission denied` | Access to keychain denied | Check system permissions |
| `keychain service unavailable` | OS keychain not available | See platform notes below |
| `invalid input parameters` | Empty connectionID or password | Validate inputs before calling |

### Platform-Specific Notes

#### macOS
- Uses Keychain Access.app
- May prompt for password on first access
- Credentials stored in login keychain
- No additional setup required

#### Windows
- Uses Windows Credential Manager
- May prompt for permission on first access
- Credentials stored in Windows Vault
- No additional setup required

#### Linux
- Uses Secret Service API (freedesktop.org)
- Requires `libsecret-1-0` package
- Install with: `sudo apt-get install libsecret-1-0`
- Uses system keyring (GNOME Keyring, KWallet, etc.)

---

## Integration Examples

### React Component Example

```typescript
import { useState, useEffect } from 'react'
import * as App from '../../wailsjs/go/main/App'

function ConnectionForm({ connectionId }: { connectionId: string }) {
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  // Load password on mount
  useEffect(() => {
    const loadPassword = async () => {
      try {
        const stored = await App.GetPassword(connectionId)
        setPassword(stored)
      } catch (error) {
        console.log('No stored password found')
      }
    }
    loadPassword()
  }, [connectionId])

  // Save password
  const handleSave = async () => {
    setLoading(true)
    try {
      await App.StorePassword(connectionId, password)
      alert('Password saved securely')
    } catch (error) {
      alert('Failed to save password: ' + error)
    } finally {
      setLoading(false)
    }
  }

  // Delete password
  const handleDelete = async () => {
    setLoading(true)
    try {
      await App.DeletePassword(connectionId)
      setPassword('')
      alert('Password removed from keychain')
    } catch (error) {
      alert('Failed to delete password: ' + error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <input
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        disabled={loading}
      />
      <button onClick={handleSave} disabled={loading}>
        Save Password
      </button>
      <button onClick={handleDelete} disabled={loading}>
        Remove Password
      </button>
    </div>
  )
}
```

### Store Integration Example

```typescript
// In your connection store (Zustand/Redux/etc.)
import * as App from '../../wailsjs/go/main/App'

interface Connection {
  id: string
  host: string
  username: string
  // password NOT stored in state - use keychain
}

// Save connection with password
async function saveConnection(connection: Connection, password: string) {
  // Store password in keychain
  await App.StorePassword(connection.id, password)

  // Store connection WITHOUT password in local state
  const { password: _, ...safeConnection } = connection
  // ... save safeConnection to store
}

// Load connection with password
async function loadConnection(connectionId: string) {
  // Load connection from store
  const connection = getConnectionFromStore(connectionId)

  // Load password from keychain
  try {
    const password = await App.GetPassword(connectionId)
    return { ...connection, password }
  } catch (error) {
    // Password not found - user will need to enter it
    return connection
  }
}

// Delete connection
async function deleteConnection(connectionId: string) {
  // Delete password from keychain
  await App.DeletePassword(connectionId)

  // Delete connection from store
  // ... remove from store
}
```

---

## Security Best Practices

1. **Never Log Passwords**: The API handles this, but don't log passwords in your code either
2. **Use Unique IDs**: Use actual connection IDs as the keychain identifier
3. **Clear Passwords**: When user logs out, call `DeletePassword` for all connections
4. **Check Availability**: Use try/catch to handle keychain unavailability gracefully
5. **UI Feedback**: Inform users when keychain is unavailable (e.g., missing libsecret on Linux)

---

## Testing

### Unit Test Example

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import * as App from '../../wailsjs/go/main/App'

describe('Credential Management', () => {
  const testId = 'test-connection-' + Date.now()
  const testPassword = 'test-password-123'

  beforeEach(async () => {
    // Clean up from previous tests
    try {
      await App.DeletePassword(testId)
    } catch (e) {
      // Ignore if not found
    }
  })

  it('should store and retrieve password', async () => {
    await App.StorePassword(testId, testPassword)
    const retrieved = await App.GetPassword(testId)
    expect(retrieved).toBe(testPassword)
  })

  it('should delete password', async () => {
    await App.StorePassword(testId, testPassword)
    await App.DeletePassword(testId)

    // Should throw when getting deleted password
    await expect(App.GetPassword(testId)).rejects.toThrow()
  })

  it('should check password existence', async () => {
    await App.StorePassword(testId, testPassword)
    expect(await App.HasPassword(testId)).toBe(true)

    await App.DeletePassword(testId)
    expect(await App.HasPassword(testId)).toBe(false)
  })
})
```

---

## Troubleshooting

### Password Not Found Error

```typescript
// Handle password not found gracefully
try {
  const password = await App.GetPassword(connectionId)
  return password
} catch (error) {
  if (error.toString().includes('not found')) {
    // Normal - prompt user to enter password
    return null
  }
  // Other error - keychain issue
  throw error
}
```

### Keychain Unavailable on Linux

```bash
# Install libsecret
sudo apt-get install libsecret-1-0

# Or use Homebrew on Linux
brew install libsecret
```

### Permission Denied

- **macOS**: Check System Preferences > Security & Privacy > Privacy > Automation
- **Windows**: Run as administrator if needed
- **Linux**: Check that your user is in the appropriate groups for keyring access

---

## Migration Guide

### Migrating from localStorage

```typescript
// One-time migration from localStorage to keychain
async function migrateToKeychain() {
  const connections = JSON.parse(localStorage.getItem('connections') || '[]')

  for (const conn of connections) {
    if (conn.password) {
      // Store in keychain
      await App.StorePassword(conn.id, conn.password)

      // Remove from localStorage
      delete conn.password
    }
  }

  // Save cleaned connections
  localStorage.setItem('connections', JSON.stringify(connections))
  console.log('Migration complete')
}
```

---

## Performance Considerations

1. **Keychain Access is Async**: Always use await
2. **Cache in Memory**: For repeated access, cache passwords in memory (securely)
3. **Batch Operations**: Process multiple passwords sequentially or in batches
4. **Error Recovery**: Implement retry logic for transient errors

---

## API Status

- ✅ Fully implemented and tested
- ✅ TypeScript types generated
- ✅ Cross-platform support (macOS, Windows, Linux)
- ✅ Production ready

For more details, see [KEYCHAIN_INTEGRATION_SUMMARY.md](./KEYCHAIN_INTEGRATION_SUMMARY.md)
